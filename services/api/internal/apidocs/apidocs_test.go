package apidocs

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestSpecIsNotEmpty(t *testing.T) {
	if len(spec) == 0 {
		t.Fatal("openapi.yaml не вшит в бинарник")
	}
	if !strings.HasPrefix(string(spec), "openapi:") {
		t.Errorf("спецификация должна начинаться с версии openapi, получено %q",
			string(spec[:min(40, len(spec))]))
	}
}

func TestSpecDeclaresAPIServer(t *testing.T) {
	text := string(spec)
	if !strings.Contains(text, "url: /api") {
		t.Error("в спецификации не объявлен сервер /api")
	}
	if strings.Contains(text, "url: /api/v1") {
		t.Error("алиас /api/v1 больше не должен быть в спецификации")
	}
}

func TestSpecCoversEveryDocumentedPath(t *testing.T) {
	text := string(spec)
	paths := []string{
		"/me", "/users", "/progress/reset", "/progress/reset/{role}",
		"/training/current-step", "/training/answer",
		"/exam/start", "/exam/restart", "/exam/message", "/exam/finish",
		"/results", "/healthz",
	}
	for _, p := range paths {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(p) + `:`).MatchString(text) {
			t.Errorf("путь %s не описан в спецификации", p)
		}
	}
}

func TestSpecListsEveryErrorCode(t *testing.T) {
	text := string(spec)
	codes := []string{
		"USER_NOT_FOUND", "TRAINING_NOT_PASSED", "TRAINING_ALREADY_PASSED",
		"STEP_MISMATCH", "INVALID_OPTION", "SESSION_NOT_FOUND",
		"SESSION_ALREADY_FINISHED", "MESSAGE_TOO_LONG", "RATE_LIMITED",
		"RESULTS_NOT_READY", "LLM_UNAVAILABLE",
	}
	for _, c := range codes {
		if !strings.Contains(text, c) {
			t.Errorf("код ошибки %s не описан в спецификации", c)
		}
	}
}

func TestSpecHandlerServesYAML(t *testing.T) {
	rec := httptest.NewRecorder()
	SpecHandler(rec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("код = %d, ожидался 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "yaml") {
		t.Errorf("Content-Type = %q, ожидался yaml", ct)
	}
	if rec.Body.Len() != len(spec) {
		t.Errorf("отдано %d байт, в спецификации %d", rec.Body.Len(), len(spec))
	}
}

func TestUIHandlerServesHTMLPointingAtSpec(t *testing.T) {
	rec := httptest.NewRecorder()
	UIHandler(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("код = %d, ожидался 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, ожидался text/html", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "url: 'openapi.yaml'") {
		t.Error("страница должна ссылаться на openapi.yaml относительным путём")
	}
	if !strings.Contains(body, "withCredentials: true") {
		t.Error("без withCredentials кука сессии не уйдёт и Try it out не сработает")
	}
	if !strings.Contains(body, "id=\"offline\"") {
		t.Error("нужен запасной блок на случай, когда CDN недоступен")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
