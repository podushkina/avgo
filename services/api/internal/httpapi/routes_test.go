package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/avito-antifraud/api/internal/apidocs"
	"github.com/avito-antifraud/api/internal/config"
)

func testServer() *Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(nil, nil, config.Config{ExamMaxCycles: 8, MessageLimit: 1000}, log)
}

func documentedPaths(t *testing.T) map[string]bool {
	t.Helper()

	out := map[string]bool{}
	inPaths := false
	for _, line := range strings.Split(string(apidocs.Spec()), "\n") {
		if line == "paths:" {
			inPaths = true
			continue
		}
		if inPaths && len(line) > 0 && line[0] != ' ' {
			break
		}
		if m := regexp.MustCompile(`^  (/\S*):\s*$`).FindStringSubmatch(line); m != nil {
			out[m[1]] = true
		}
	}
	if len(out) == 0 {
		t.Fatal("не удалось разобрать пути из спецификации")
	}
	return out
}

func TestEveryRouteIsDocumented(t *testing.T) {
	documented := documentedPaths(t)

	undocumented := []string{}
	for _, r := range testServer().routeTable() {
		if documented[r.path] {
			continue
		}
		if r.path == "/openapi.yaml" || r.path == "/docs" {
			continue
		}
		undocumented = append(undocumented, r.method+" "+r.path)
	}

	sort.Strings(undocumented)
	if len(undocumented) > 0 {
		t.Errorf("ручки зарегистрированы, но не описаны в openapi.yaml: %v", undocumented)
	}
}

func TestEveryDocumentedPathIsRouted(t *testing.T) {
	registered := map[string]bool{}
	for _, r := range testServer().routeTable() {
		registered[r.path] = true
	}

	missing := []string{}
	for path := range documentedPaths(t) {
		if !registered[path] {
			missing = append(missing, path)
		}
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("пути описаны в openapi.yaml, но не зарегистрированы: %v", missing)
	}
}

func TestUnknownAPIPathReturnsJSONError(t *testing.T) {
	handler := testServer().Routes()

	for _, path := range []string{"/api/nope", "/api/v1/nope", "/api/exam/nope"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: код = %d, ожидался 404", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("%s: Content-Type = %q, ожидался JSON", path, ct)
		}
		if !strings.Contains(rec.Body.String(), "NOT_FOUND") {
			t.Errorf("%s: тело не содержит код ошибки: %s", path, rec.Body.String())
		}
	}
}

func TestRoutesAreServedUnderBothPrefixes(t *testing.T) {
	handler := testServer().Routes()

	for _, prefix := range []string{"/api", "/api/v1"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, prefix+"/openapi.yaml", nil))

		if rec.Code != http.StatusOK {
			t.Errorf("%s/openapi.yaml: код = %d, ожидался 200", prefix, rec.Code)
		}
	}
}
