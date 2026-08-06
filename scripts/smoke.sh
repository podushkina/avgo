#!/usr/bin/env bash
set -uo pipefail

BASE="${BASE:-http://localhost:8080}"
fails=0

check() {
    local name="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        printf '  ✓ %s\n' "$name"
    else
        printf '  ✗ %s (ожидалось %s, получено %s)\n' "$name" "$expected" "$actual"
        fails=$((fails + 1))
    fi
}

echo "Проверяем $BASE"

echo "[1] Здоровье сервисов"
check "api-service"  "200" "$(curl -so /dev/null -w '%{http_code}' "$BASE/api/healthz")"
check "ai-service"   "200" "$(curl -so /dev/null -w '%{http_code}' "$BASE/api/dialog/healthz")"
check "фронтенд"     "200" "$(curl -so /dev/null -w '%{http_code}' "$BASE/")"

echo "[2] Пользователь"
USER_ID=$(curl -s -X POST "$BASE/api/users" -H 'Content-Type: application/json' \
    -d '{"external_id":"smoke-test-user"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')
check "создан" "ok" "$([ -n "$USER_ID" ] && echo ok || echo empty)"

echo "[3] Сценарии"
SCENARIOS=$(curl -s "$BASE/api/scenarios?role=seller")
COUNT=$(echo "$SCENARIOS" | python3 -c 'import sys,json; print(len(json.load(sys.stdin)))')
check "загружены" "6" "$COUNT"
LEAK=$(echo "$SCENARIOS" | python3 -c '
import sys, json
keys = set()
for s in json.load(sys.stdin):
    keys |= set(s.keys())
print("leak" if keys & {"correct_option", "explanation", "red_flags"} else "clean")')
check "правильные ответы скрыты" "clean" "$LEAK"

echo "[4] Проверка одного ответа"
FIRST_ID=$(echo "$SCENARIOS" | python3 -c 'import sys,json; print(json.load(sys.stdin)[0]["id"])')
CHECK=$(curl -s -X POST "$BASE/api/scenarios/$FIRST_ID/check" \
    -H 'Content-Type: application/json' -d '{"option":0}')
HAS_OUTCOME=$(echo "$CHECK" | python3 -c '
import sys, json
d = json.load(sys.stdin)
print("ok" if d.get("your_outcome") and d.get("explanation") else "missing")')
check "возвращает последствие и разбор" "ok" "$HAS_OUTCOME"

echo "[5] Отправка попытки"
ATTEMPT=$(echo "$SCENARIOS" | python3 -c '
import sys, json
scenarios = json.load(sys.stdin)
print(json.dumps({"answers": [{"scenario_id": s["id"], "option": 0} for s in scenarios]}))')
RESULT=$(curl -s -X POST "$BASE/api/attempts" -H 'Content-Type: application/json' \
    -d "$(python3 -c "
import json, sys
body = json.loads('''$ATTEMPT''')
body['user_id'] = '$USER_ID'
body['role'] = 'seller'
print(json.dumps(body))")")
TOTAL=$(echo "$RESULT" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("total"))')
check "счёт посчитан по всем вопросам" "6" "$TOTAL"

echo "[6] Прогресс"
HISTORY=$(curl -s "$BASE/api/progress?user_id=$USER_ID" | python3 -c 'import sys,json; print(len(json.load(sys.stdin)))')
check "попытка сохранена" "ok" "$([ "$HISTORY" -ge 1 ] && echo ok || echo none)"

echo "[7] Диалог с ИИ"
SESSION=$(curl -s -X POST "$BASE/api/dialog/sessions" -H 'Content-Type: application/json' \
    -d '{"role":"seller","difficulty":"medium"}')
SESSION_ID=$(echo "$SESSION" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("session_id",""))')
check "сессия создана" "ok" "$([ -n "$SESSION_ID" ] && echo ok || echo empty)"

if [ -n "$SESSION_ID" ]; then
    STREAM=$(curl -sN --max-time 120 -X POST "$BASE/api/dialog/sessions/$SESSION_ID/messages" \
        -H 'Content-Type: application/json' -d '{"text":"Здравствуйте! Да, товар в наличии."}')
    check "модель ответила потоком" "ok" \
        "$(echo "$STREAM" | grep -q '^event: done' && echo ok || echo fail)"

    REPORT=$(curl -s -X POST "$BASE/api/dialog/sessions/$SESSION_ID/finish" \
        -H 'Content-Type: application/json' -d '{}')
    check "разбор диалога получен" "ok" \
        "$(echo "$REPORT" | python3 -c 'import sys,json; d=json.load(sys.stdin); print("ok" if "report" in d else "fail")')"
fi

echo
if [ "$fails" -eq 0 ]; then
    echo "Все проверки пройдены."
else
    echo "Провалено проверок: $fails"
fi
exit "$fails"
