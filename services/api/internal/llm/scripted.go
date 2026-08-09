package llm

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
)

type Scripted struct {
	mock  Mock
	turns atomic.Uint64
}

var scriptedTurns = []string{
	"Здравствуйте! Отличное предложение, беру без торга. Как удобнее оформить?",
	"Я оформил доставку через сервис. Вам придёт подтверждение — посмотрите, пожалуйста.",
	"Что-то оплата зависла. Перейдите по ссылке avito-dostavka-pay.example/get и подтвердите карту.",
	"Вам сейчас придёт код в СМС для подтверждения зачисления. Продиктуйте его, пожалуйста.",
	"Без кода бухгалтерия не проведёт платёж. Давайте быстрее, у меня рабочий день заканчивается.",
	"Хорошо, тогда просто скиньте номер карты, срок и три цифры с обратной стороны.",
}

func NewScripted() *Scripted {
	return &Scripted{mock: *NewMock()}
}

func (s *Scripted) Stream(ctx context.Context, _ []Message, onToken func(string) error) error {
	turn := s.turns.Add(1) - 1
	reply := scriptedTurns[turn%uint64(len(scriptedTurns))]

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

func (s *Scripted) Complete(ctx context.Context, msgs []Message, schema json.RawMessage) (string, error) {
	return s.mock.Complete(ctx, msgs, schema)
}
