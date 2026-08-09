package llm

import (
	"context"
	"encoding/json"
	"errors"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

var ErrUnavailable = errors.New("модель недоступна")

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client interface {
	Stream(ctx context.Context, msgs []Message, onToken func(string) error) error
	Complete(ctx context.Context, msgs []Message, schema json.RawMessage) (string, error)
}

func Collect(ctx context.Context, c Client, msgs []Message) (string, error) {
	var sb []byte
	err := c.Stream(ctx, msgs, func(tok string) error {
		sb = append(sb, tok...)
		return nil
	})
	return string(sb), err
}
