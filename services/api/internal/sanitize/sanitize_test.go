package sanitize

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTextKeepsExampleDomains(t *testing.T) {
	in := "Перейдите на avito-dostavka-pay.example/get и подтвердите"
	if got := Text(in); got != in {
		t.Errorf("домен .example должен сохраняться, получено %q", got)
	}
}

func TestTextReplacesRealDomains(t *testing.T) {
	cases := []string{
		"Вот ссылка: https://sberbank-secure.ru/pay",
		"зайдите на evil-phishing.com/order",
		"откройте avito.ru.secure-pay.net/x",
	}
	for _, in := range cases {
		got := Text(in)
		if strings.Contains(got, ".ru/") || strings.Contains(got, ".com/") || strings.Contains(got, ".net/") {
			t.Errorf("реальный домен не заменён: %q -> %q", in, got)
		}
		if !strings.Contains(got, placeholder) {
			t.Errorf("ожидалась подстановка заглушки: %q -> %q", in, got)
		}
	}
}

func TestTextLeavesPlainProseAlone(t *testing.T) {
	in := "Здравствуйте! Товар ещё актуален? Цена 45 000 руб."
	if got := Text(in); got != in {
		t.Errorf("обычный текст не должен меняться: %q -> %q", in, got)
	}
}

func TestTextDropsForeignScript(t *testing.T) {
	in := "Да, конечно! Пользовался этим сервисом之前对话被中断了 раньше."
	got := Text(in)

	if strings.ContainsAny(got, "之前对话被中断了") {
		t.Errorf("иероглифы должны вырезаться, получено %q", got)
	}
	if !strings.Contains(got, "Да, конечно!") || !strings.Contains(got, "раньше.") {
		t.Errorf("русский текст должен сохраняться, получено %q", got)
	}
}

func TestTextKeepsPureRussian(t *testing.T) {
	in := "Да, конечно! Пользовался этим сервисом раньше."
	if got := Text(in); got != in {
		t.Errorf("чистый русский не должен меняться: %q -> %q", in, got)
	}
}

func TestStreamerDropsForeignScriptMidStream(t *testing.T) {
	var sb strings.Builder
	s := NewStreamer(func(chunk string) error {
		sb.WriteString(chunk)
		return nil
	})

	for _, tok := range []string{"Да, ", "конечно", "! ", "对话", "被中断 ", "давайте ", "оформим"} {
		_ = s.Push(tok)
	}
	_ = s.Close()

	got := sb.String()
	if strings.ContainsAny(got, "对话被中断") {
		t.Errorf("иероглифы просочились в поток: %q", got)
	}
	if !strings.Contains(got, "давайте") {
		t.Errorf("русские слова потерялись: %q", got)
	}
}

func TestStreamerEmitsOnWordBoundaries(t *testing.T) {
	var out []string
	s := NewStreamer(func(chunk string) error {
		out = append(out, chunk)
		return nil
	})

	for _, tok := range []string{"Пере", "йдите", " на ", "evil", ".com", " сейчас"} {
		if err := s.Push(tok); err != nil {
			t.Fatalf("Push: %v", err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	full := strings.Join(out, "")
	if strings.Contains(full, "evil.com") {
		t.Errorf("ссылка просочилась в поток: %q", full)
	}
	if !strings.Contains(full, placeholder) {
		t.Errorf("заглушка не подставлена: %q", full)
	}
	if !strings.HasPrefix(full, "Перейдите на ") {
		t.Errorf("текст исказился: %q", full)
	}
}

func TestStreamerFlushesTailOnClose(t *testing.T) {
	var sb strings.Builder
	s := NewStreamer(func(chunk string) error {
		sb.WriteString(chunk)
		return nil
	})

	_ = s.Push("хвост без пробела")
	if strings.Contains(sb.String(), "пробела") {
		t.Error("последнее слово не должно уходить до Close: оно может быть частью ссылки")
	}
	_ = s.Close()

	if sb.String() != "хвост без пробела" {
		t.Errorf("получено %q", sb.String())
	}
}

func TestStreamerStopsOnRepeatedSentence(t *testing.T) {
	var sb strings.Builder
	s := NewStreamer(func(chunk string) error {
		sb.WriteString(chunk)
		return nil
	})

	sentence := "Перейдите по ссылке и подтвердите оплату. "
	var err error
	for _, tok := range strings.SplitAfter(sentence+sentence+"Ещё текст. ", " ") {
		if err = s.Push(tok); err != nil {
			break
		}
	}

	if !errors.Is(err, ErrRepeat) {
		t.Fatalf("повтор предложения должен прерывать поток, получено %v", err)
	}
	if strings.Count(sb.String(), "подтвердите оплату") > 2 {
		t.Errorf("зацикливание не обрезано: %q", sb.String())
	}
	if strings.Contains(sb.String(), "Ещё текст") {
		t.Errorf("после обрыва не должно быть новых токенов: %q", sb.String())
	}
}

func TestStreamerAllowsDistinctSentences(t *testing.T) {
	var sb strings.Builder
	s := NewStreamer(func(chunk string) error {
		sb.WriteString(chunk)
		return nil
	})

	text := "Здравствуйте, товар актуален? Готов забрать сегодня вечером. Цена подходит полностью."
	for _, tok := range strings.SplitAfter(text, " ") {
		if err := s.Push(tok); err != nil {
			t.Fatalf("разные предложения не должны прерывать поток: %v", err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if sb.String() != text {
		t.Errorf("текст исказился: %q", sb.String())
	}
}

func TestStreamerIgnoresShortRepeats(t *testing.T) {
	s := NewStreamer(func(string) error { return nil })

	for _, tok := range strings.SplitAfter("Да. Да. Ок. Ок. ", " ") {
		if err := s.Push(tok); err != nil {
			t.Fatalf("короткие повторы не должны считаться зацикливанием: %v", err)
		}
	}
}

func TestTrimToSentence(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Первое предложение. Второе оборвалось на полу",
			"Первое предложение."},
		{"Уже законченное предложение.", "Уже законченное предложение."},
		{"Вопрос закончен?", "Вопрос закончен?"},
		{"Восклицание! И обрывок", "Восклицание!"},
		{"Многоточие… и обрывок", "Многоточие…"},
		{"Совсем без границы предложения", "Совсем без границы предложения"},
		{"   ", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := TrimToSentence(c.in); got != c.want {
			t.Errorf("TrimToSentence(%q) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}

func TestTrimToSentenceKeepsMultibyteIntact(t *testing.T) {
	got := TrimToSentence("Здравствуйте… потом обрыв")

	if !strings.HasSuffix(got, "…") {
		t.Errorf("многоточие должно остаться целым: %q", got)
	}
	if !utf8.ValidString(got) {
		t.Errorf("строка повреждена на границе руны: %q", got)
	}
}

func TestStreamerCloseIsIdempotent(t *testing.T) {
	calls := 0
	s := NewStreamer(func(string) error { calls++; return nil })

	_ = s.Push("привет")
	_ = s.Close()
	_ = s.Close()

	if calls != 1 {
		t.Errorf("emit вызван %d раз, ожидался 1", calls)
	}
}

func TestTextReplacesSchemelessURL(t *testing.T) {
	got := Text("Нажмите здесь: https://confirm-payment.")

	if strings.Contains(got, "https://confirm-payment") {
		t.Errorf("адрес без домена верхнего уровня должен подменяться: %q", got)
	}
	if !strings.Contains(got, placeholder) {
		t.Errorf("ожидалась заглушка: %q", got)
	}
}

func TestTextDoesNotDoubleReplacePlaceholder(t *testing.T) {
	got := Text("Перейдите на https://" + placeholder + " и подтвердите")

	if strings.Count(got, "/order") != 1 {
		t.Errorf("заглушка не должна подменяться повторно: %q", got)
	}
}

func TestTextKeepsExampleWithScheme(t *testing.T) {
	in := "Вот ссылка https://avito-pay.example/get"
	if got := Text(in); got != in {
		t.Errorf("домен .example со схемой должен сохраняться: %q -> %q", in, got)
	}
}

func TestStripDirectiveCutsServicePrompt(t *testing.T) {
	in := "Привет! Как тебе смартфон? [Служебная инструкция] Это твой ход №1. Продолжай выстраивать доверие."
	got := StripDirective(in)

	if strings.Contains(got, "Служебная инструкция") || strings.Contains(got, "твой ход") {
		t.Errorf("служебная инструкция должна вырезаться: %q", got)
	}
	if !strings.Contains(got, "Как тебе смартфон?") {
		t.Errorf("реплика до маркера должна сохраняться: %q", got)
	}
}

func TestStripDirectiveCutsBareDirectiveText(t *testing.T) {
	got := StripDirective("Отличный выбор! Это твой ход №3. Примени приём.")

	if strings.Contains(got, "ход №3") {
		t.Errorf("инструкция без скобок тоже должна вырезаться: %q", got)
	}
	if !strings.Contains(got, "Отличный выбор!") {
		t.Errorf("полезный текст потерян: %q", got)
	}
}

func TestStripDirectiveKeepsCleanText(t *testing.T) {
	in := "Здравствуйте! Готов купить прямо сейчас."
	if got := StripDirective(in); got != in {
		t.Errorf("чистый текст не должен меняться: %q -> %q", in, got)
	}
}

func TestTextStripsDirective(t *testing.T) {
	got := Text("Привет! [Служебная инструкция] Это твой ход №2.")

	if strings.Contains(got, "Служебная") {
		t.Errorf("Text должен вырезать инструкцию: %q", got)
	}
}
