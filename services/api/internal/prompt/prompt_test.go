package prompt

import (
	"strings"
	"testing"
)

func TestTurnDirectiveHoldsBackBeforePressureTurn(t *testing.T) {
	d := TurnDirective(DifficultyHard, 1)

	if !strings.Contains(d, "доверие") {
		t.Errorf("на первом ходу hard должна быть фаза доверия, получено: %q", d)
	}
	if strings.Contains(d, "обязан применить приём") {
		t.Errorf("приём не должен требоваться на первом ходу hard: %q", d)
	}
}

func TestTurnDirectiveDemandsAttackAtPressureTurn(t *testing.T) {
	cases := []struct {
		difficulty Difficulty
		turn       int
	}{
		{DifficultyEasy, 1},
		{DifficultyMedium, 2},
		{DifficultyHard, 3},
	}
	for _, c := range cases {
		d := TurnDirective(c.difficulty, c.turn)
		if !strings.Contains(d, "обязан применить приём") {
			t.Errorf("%s, ход %d: приём должен требоваться, получено: %q", c.difficulty, c.turn, d)
		}
	}
}

func TestTurnDirectiveRotatesAttacks(t *testing.T) {
	seen := map[string]bool{}
	for turn := 3; turn < 3+len(attacks); turn++ {
		seen[TurnDirective(DifficultyHard, turn)] = true
	}
	if len(seen) != len(attacks) {
		t.Errorf("директивы должны меняться по ходам: уникальных %d из %d", len(seen), len(attacks))
	}
}

func TestTurnDirectiveNeverPanicsOnHighTurns(t *testing.T) {
	for turn := 1; turn <= 50; turn++ {
		if TurnDirective(DifficultyEasy, turn) == "" {
			t.Fatalf("пустая директива на ходу %d", turn)
		}
	}
}

func TestSystemIncludesRoleAndDifficulty(t *testing.T) {
	seller := System(RoleSeller, DifficultyHard)
	buyer := System(RoleBuyer, DifficultyEasy)

	if !strings.Contains(seller, "ПРОДАВЕЦ") {
		t.Error("промпт для роли seller должен описывать жертву-продавца")
	}
	if !strings.Contains(buyer, "ПОКУПАТЕЛЬ") {
		t.Error("промпт для роли buyer должен описывать жертву-покупателя")
	}
	if !strings.Contains(seller, "ВЫСОКИЙ") || !strings.Contains(buyer, "НАЧАЛЬНЫЙ") {
		t.Error("промпт должен включать блок выбранного уровня сложности")
	}
	if !strings.Contains(seller, ".example") {
		t.Error("промпт обязан требовать вымышленные домены")
	}
}

func TestParseDifficultyDefaultsToMedium(t *testing.T) {
	d, err := ParseDifficulty("")
	if err != nil || d != DifficultyMedium {
		t.Errorf("пустая сложность должна давать medium, получено %v (%v)", d, err)
	}
	if _, err := ParseDifficulty("insane"); err == nil {
		t.Error("неизвестная сложность должна отвергаться")
	}
}
