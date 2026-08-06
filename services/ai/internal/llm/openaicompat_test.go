package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sseServer(t *testing.T, chunks []string, capture *chatRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, capture); err != nil {
				t.Errorf("тело запроса не разобралось: %v", err)
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			_, _ = fmt.Fprint(w, c)
		}
	}))
}

func chunk(text string) string {
	return fmt.Sprintf(`data: {"choices":[{"delta":{"content":%q}}]}`+"\n\n", text)
}

func collect(t *testing.T, srv *httptest.Server) (string, error) {
	t.Helper()
	c := NewOpenAICompat(srv.URL, "test-model", "", 5*time.Second)
	var sb strings.Builder
	err := c.Stream(context.Background(), []Message{{Role: RoleUser, Content: "привет"}},
		func(tok string) error {
			sb.WriteString(tok)
			return nil
		})
	return sb.String(), err
}

func TestStreamAssemblesTokens(t *testing.T) {
	srv := sseServer(t, []string{chunk("Здрав"), chunk("ствуйте"), chunk("!"), "data: [DONE]\n\n"}, nil)
	defer srv.Close()

	got, err := collect(t, srv)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if got != "Здравствуйте!" {
		t.Errorf("собрано %q, ожидалось «Здравствуйте!»", got)
	}
}

func TestStreamStopsAtDone(t *testing.T) {
	srv := sseServer(t, []string{chunk("раз"), "data: [DONE]\n\n", chunk("два")}, nil)
	defer srv.Close()

	got, err := collect(t, srv)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if got != "раз" {
		t.Errorf("собрано %q, ожидалось «раз»: чтение должно прекращаться на [DONE]", got)
	}
}

func TestStreamReportsEmptyResponse(t *testing.T) {
	srv := sseServer(t, []string{`data: {"choices":[{"delta":{}}]}` + "\n\n", "data: [DONE]\n\n"}, nil)
	defer srv.Close()

	_, err := collect(t, srv)
	if err == nil {
		t.Fatal("пустой ответ должен приводить к ошибке: так отсекаются reasoning-модели")
	}
	if !strings.Contains(err.Error(), "reasoning") {
		t.Errorf("ошибка = %v, ожидалось упоминание reasoning-модели", err)
	}
}

func TestStreamPropagatesModelError(t *testing.T) {
	srv := sseServer(t, []string{`data: {"error":{"message":"model not found"}}` + "\n\n"}, nil)
	defer srv.Close()

	_, err := collect(t, srv)
	if err == nil || !strings.Contains(err.Error(), "model not found") {
		t.Errorf("ошибка = %v, ожидалось «model not found»", err)
	}
}

func TestStreamReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error":"model not found"}`)
	}))
	defer srv.Close()

	_, err := collect(t, srv)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("ошибка = %v, ожидался код 404", err)
	}
}

func TestStreamSendsModelAndMessages(t *testing.T) {
	var got chatRequest
	srv := sseServer(t, []string{chunk("ок"), "data: [DONE]\n\n"}, &got)
	defer srv.Close()

	if _, err := collect(t, srv); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if got.Model != "test-model" {
		t.Errorf("model = %q, ожидалось test-model", got.Model)
	}
	if !got.Stream {
		t.Error("stream должен быть true")
	}
	if len(got.Messages) != 1 || got.Messages[0].Content != "привет" {
		t.Errorf("messages = %+v", got.Messages)
	}
}

func TestStreamStopsWhenCallbackFails(t *testing.T) {
	srv := sseServer(t, []string{chunk("раз"), chunk("два"), "data: [DONE]\n\n"}, nil)
	defer srv.Close()

	c := NewOpenAICompat(srv.URL, "m", "", 5*time.Second)
	calls := 0
	err := c.Stream(context.Background(), []Message{{Role: RoleUser, Content: "x"}},
		func(string) error {
			calls++
			return io.ErrClosedPipe
		})

	if err == nil {
		t.Fatal("ошибка обработчика должна прерывать стрим")
	}
	if calls != 1 {
		t.Errorf("обработчик вызван %d раз, ожидался 1", calls)
	}
}

func TestMockStreamsWords(t *testing.T) {
	m := NewMock()
	var sb strings.Builder
	if err := m.Stream(context.Background(), nil, func(tok string) error {
		sb.WriteString(tok)
		return nil
	}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if sb.Len() == 0 {
		t.Error("mock должен возвращать непустой ответ")
	}
}

func TestMockAdvancesThroughReplies(t *testing.T) {
	m := NewMock()
	first, second := "", ""
	_ = m.Stream(context.Background(), nil, func(t string) error { first += t; return nil })
	_ = m.Stream(context.Background(), nil, func(t string) error { second += t; return nil })

	if first == second {
		t.Error("mock должен выдавать разные реплики на последовательных ходах")
	}
}
