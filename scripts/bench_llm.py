import json
import os
import sys
import urllib.error
import urllib.request

OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://127.0.0.1:11434")

SYSTEM = (
    "Ты играешь роль мошенника на сайте объявлений в обучающем тренажёре по "
    "кибербезопасности. Пользователь — продавец iPhone. Твоя цель в игре — уговорить "
    "его перейти по ссылке для «получения оплаты». Ты вежлив, настойчив, торопишь. "
    "Никогда не признавайся, что ты мошенник. Отвечай кратко, 1-2 предложения, "
    "как в реальном чате, только на русском."
)
TURNS = [
    "Здравствуйте! Да, айфон ещё в продаже.",
    "А почему не через Авито Доставку?",
]

def post(path: str, payload: dict, timeout: int = 180) -> dict:
    req = urllib.request.Request(
        OLLAMA_URL + path,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read())

def installed_models() -> list[str]:
    with urllib.request.urlopen(OLLAMA_URL + "/api/tags", timeout=10) as resp:
        return [m["name"] for m in json.loads(resp.read()).get("models", [])]

def bench(model: str) -> None:
    print(f"═══ {model}")
    history = [{"role": "system", "content": SYSTEM}]
    total_tokens = 0
    total_seconds = 0.0

    for turn, user_text in enumerate(TURNS, 1):
        history.append({"role": "user", "content": user_text})
        try:
            d = post("/api/chat", {
                "model": model,
                "messages": history,
                "stream": False,
                "options": {"num_predict": 120, "temperature": 0.8},
            })
        except (urllib.error.URLError, TimeoutError) as e:
            print(f"  ОШИБКА: {e}\n")
            return

        if "error" in d:
            print(f"  ОШИБКА: {d['error']}\n")
            return

        msg = d.get("message", {})
        content = (msg.get("content") or "").strip()
        thinking = (msg.get("thinking") or "").strip()

        if not content:
            verdict = ("reasoning-модель, весь вывод ушёл в thinking"
                       if thinking else "пустой ответ")
            print(f"  ВЕРДИКТ : {verdict} — НЕ ГОДИТСЯ для чата\n")
            return

        total_tokens += d.get("eval_count", 0)
        total_seconds += (d.get("eval_duration", 0) or 0) / 1e9
        if turn == 1:
            print(f"  загрузка: {d.get('load_duration', 0) / 1e9:.1f}s")
        print(f"  ход {turn}  : {content}")
        history.append({"role": "assistant", "content": content})

    rate = total_tokens / total_seconds if total_seconds else 0
    print(f"  tok/s   : {rate:.1f}")
    print()

def main() -> int:
    models = sys.argv[1:]
    if not models:
        try:
            models = installed_models()
        except urllib.error.URLError as e:
            print(f"Ollama недоступен на {OLLAMA_URL}: {e}")
            return 1

    print(f"Endpoint: {OLLAMA_URL}\n")
    for m in models:
        bench(m)
    return 0

if __name__ == "__main__":
    sys.exit(main())
