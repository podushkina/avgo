package llm

import (
	"context"
	"strings"
)

type Mock struct {
	replies []string
	calls   int
}

func NewMock() *Mock {
	return &Mock{replies: []string{
		"Здравствуйте! Товар меня полностью устраивает, готов оформить прямо сейчас.",
		"Я уже оформил доставку через сервис. Подтвердите получение оплаты по ссылке: avito-dostavka-pay.example/get",
		"Деньги зарезервированы, но подтвердить нужно в течение 10 минут, иначе бронь сорвётся.",
		"Просто введите номер карты и код из СМС на странице — это стандартная процедура сервиса.",
		"Не переживайте, это официальная страница. Давайте быстрее, я уже оплатил.",
	}}
}

func (m *Mock) Stream(ctx context.Context, msgs []Message, onToken func(string) error) error {
	reply := m.replies[m.calls%len(m.replies)]
	m.calls++

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
