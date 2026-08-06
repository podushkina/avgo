package analysis

import (
	"strings"
	"testing"

	"github.com/avito-antifraud/ai/internal/llm"
)

func dialog(pairs ...[2]string) []llm.Message {
	out := []llm.Message{{Role: llm.RoleSystem, Content: "системный промпт"}}
	for _, p := range pairs {
		if p[0] != "" {
			out = append(out, llm.Message{Role: llm.RoleAssistant, Content: p[0]})
		}
		if p[1] != "" {
			out = append(out, llm.Message{Role: llm.RoleUser, Content: p[1]})
		}
	}
	return out
}

func codes(fs []Finding) map[string]bool {
	m := map[string]bool{}
	for _, f := range fs {
		m[f.Code] = true
	}
	return m
}

func TestDetectsScammerTactics(t *testing.T) {
	msgs := dialog(
		[2]string{"Оплатите срочно по ссылке pay.example/get, иначе бронь сгорит", "нет"},
		[2]string{"Назовите код из СМС для подтверждения", "нет"},
		[2]string{"Давайте перейдём в WhatsApp", "нет"},
	)

	got := codes(Analyze(msgs).Tactics)
	for _, want := range []string{"phishing_link", "urgency", "sms_code", "off_platform"} {
		if !got[want] {
			t.Errorf("тактика %q не распознана, найдено: %v", want, got)
		}
	}
}

func TestDetectsUserMistakes(t *testing.T) {
	msgs := dialog(
		[2]string{"Назовите номер карты", "4276 1600 1234 5678"},
		[2]string{"И код из СМС", "код 445123"},
		[2]string{"Перейдите по ссылке", "перешёл, ввёл данные"},
	)

	rep := Analyze(msgs)
	got := codes(rep.Mistakes)
	for _, want := range []string{"shared_card", "shared_code", "followed_link"} {
		if !got[want] {
			t.Errorf("ошибка %q не распознана, найдено: %v", want, got)
		}
	}
	if rep.Survived {
		t.Error("Survived должен быть false, если пользователь допустил ошибки")
	}
}

func TestCleanDialogSurvives(t *testing.T) {
	msgs := dialog(
		[2]string{"Оплатите по ссылке pay.example/x", "Нет, только через доставку площадки"},
		[2]string{"Это долго, давайте быстрее", "Меня устраивает штатный способ"},
		[2]string{"Назовите код из СМС", "Код я не сообщаю никому"},
		[2]string{"Тогда переведите на карту", "Отказываюсь, оформляйте доставку"},
		[2]string{"Жаль", "Всего доброго"},
	)

	rep := Analyze(msgs)
	if !rep.Survived {
		t.Errorf("чистый диалог должен считаться пройденным, найдено: %v", codes(rep.Mistakes))
	}
	if rep.Turns != 5 {
		t.Errorf("реплик пользователя = %d, ожидалось 5", rep.Turns)
	}
	if !strings.Contains(rep.Verdict, "выдержали") {
		t.Errorf("вердикт = %q", rep.Verdict)
	}
}

func TestFindingsAreNotDuplicated(t *testing.T) {
	msgs := dialog(
		[2]string{"Срочно! Быстрее!", "хорошо"},
		[2]string{"Срочно, прямо сейчас!", "хорошо"},
	)

	var urgency int
	for _, f := range Analyze(msgs).Tactics {
		if f.Code == "urgency" {
			urgency++
		}
	}
	if urgency != 1 {
		t.Errorf("тактика urgency найдена %d раз, ожидался 1", urgency)
	}
}

func TestEmptyDialogHasNoFindings(t *testing.T) {
	rep := Analyze(dialog())

	if len(rep.Tactics) != 0 || len(rep.Mistakes) != 0 {
		t.Errorf("на пустом диалоге не должно быть находок: %+v", rep)
	}
	if rep.Turns != 0 {
		t.Errorf("Turns = %d, ожидалось 0", rep.Turns)
	}
	if !strings.Contains(rep.Verdict, "не начался") {
		t.Errorf("вердикт = %q", rep.Verdict)
	}
}

func TestAdviceMatchesMistakes(t *testing.T) {
	msgs := dialog([2]string{"Назовите код", "вот код 123456"})

	advice := strings.Join(Analyze(msgs).Advice, " ")
	if !strings.Contains(advice, "код") {
		t.Errorf("совет должен касаться кодов, получено: %q", advice)
	}
}

func TestAdviceIsNeverEmpty(t *testing.T) {
	if len(Analyze(dialog([2]string{"Привет", "Здравствуйте"})).Advice) == 0 {
		t.Error("совет должен быть даже при чистом прохождении")
	}
}

func TestQuoteIsTruncated(t *testing.T) {
	long := strings.Repeat("а", 400) + " перешёл по ссылке"
	msgs := dialog([2]string{"ссылка", long})

	for _, f := range Analyze(msgs).Mistakes {
		if len([]rune(f.Quote)) > quoteLimit+1 {
			t.Errorf("цитата длиной %d не обрезана", len([]rune(f.Quote)))
		}
	}
}
