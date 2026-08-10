package exam

import (
	"strings"
	"testing"

	"github.com/avito-antifraud/api/internal/prompt"
	"github.com/avito-antifraud/api/internal/sanitize"
)

func TestDirectiveMarkerIsStripped(t *testing.T) {
	leaked := "Реплика мошенника. " + prompt.TurnDirective(prompt.DifficultyMedium, 1)

	got := sanitize.StripDirective(leaked)

	if strings.Contains(got, prompt.DirectiveMarker) {
		t.Fatalf("маркер из prompt не вырезается санитайзером: %q", got)
	}
	if !strings.Contains(got, "Реплика мошенника.") {
		t.Errorf("полезная часть реплики потеряна: %q", got)
	}
}

func TestEveryTurnDirectiveIsStrippable(t *testing.T) {
	for turn := 1; turn <= 10; turn++ {
		leaked := "Текст. " + prompt.TurnDirective(prompt.DifficultyHard, turn)
		if got := sanitize.StripDirective(leaked); strings.Contains(got, "ход №") {
			t.Errorf("ход %d: инструкция просочилась: %q", turn, got)
		}
	}
}
