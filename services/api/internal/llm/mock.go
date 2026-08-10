package llm

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync/atomic"
)

type Mock struct {
	replies []string
	calls   atomic.Uint64
}

func NewMock() *Mock {
	return &Mock{replies: []string{
		"Здравствуйте! Товар меня полностью устраивает, готов оформить прямо сейчас.",
		"Я уже оформил доставку через сервис. Подтвердите получение оплаты по ссылке: avito-dostavka-pay.example/get",
		"Деньги зарезервированы, но подтвердить нужно в течение 10 минут, иначе бронь сорвётся.",
		"Просто введите номер карты и код из СМС на странице — это стандартная процедура сервиса.",
		"Не переживайте, это официальная страница. Давайте быстрее, я уже оплатил.",
		"Хорошо, тогда давайте продолжим в мессенджере, там мне удобнее скинуть реквизиты.",
	}}
}

func (m *Mock) Stream(ctx context.Context, msgs []Message, onToken func(string) error) error {
	currentCall := m.calls.Add(1) - 1
	reply := m.replies[currentCall%uint64(len(m.replies))]

	for i, word := range strings.Fields(reply) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if i > 0 {
			word = " " + word
		}
		if err := onToken(word); err != nil {
			return err
		}
	}
	return nil
}

var mockPatterns = map[string]*regexp.Regexp{
	"leaked_sms_code": regexp.MustCompile(
		`(?i)(код|пароль)\D{0,20}\d{4,6}|\bмой код\b|вот код|код из смс[:\s-]*\d`),
	"leaked_card_data": regexp.MustCompile(
		`(?i)\b\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{4}\b|cvc|cvv`),
	"agreed_to_pay_via_link": regexp.MustCompile(
		`(?i)переш[её]л|перешла|открыл ссылк|ввёл данн|ввел данн|заполнил форм|оплатил по ссылк`),
	"agreed_off_platform": regexp.MustCompile(
		`(?i)перевед[уё]|оплачу на карт|скину денег|отправил деньги|внесу предоплат|готов внести`),
	"shared_phone": regexp.MustCompile(`\+7\d{10}|\b8\d{10}\b`),
	"moved_to_messenger": regexp.MustCompile(
		`(?i)(да|хорошо|ок|давайте|можно)\W{0,15}(whatsapp|вотсап|ватсап|telegram|телеграм)|мой (ватсап|телеграм)`),
}

var mockRefusedAndEnded = regexp.MustCompile(
	`(?i)всего доброго|до свидания|прощайте|разговор окончен|больше не пишите|` +
		`закрываю|прекращаю общение|пожалуюсь|напишу в поддержку|отправляю жалобу|в бан`)

func (m *Mock) Complete(_ context.Context, msgs []Message, _ json.RawMessage) (string, error) {
	last := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleUser {
			last = msgs[i].Content
			break
		}
	}

	out := map[string]bool{}
	held := true
	for code, re := range mockPatterns {
		hit := re.MatchString(last)
		out[code] = hit
		if hit {
			held = false
		}
	}
	out["held_boundary"] = held
	out["refused_and_ended"] = held && mockRefusedAndEnded.MatchString(last)

	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
