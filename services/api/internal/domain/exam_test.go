package domain

import (
	"strings"
	"testing"
)

func TestMistakeIsCritical(t *testing.T) {
	critical := []Mistake{
		MistakeLeakedSMSCode, MistakeLeakedCardData,
		MistakeAgreedPayLink, MistakeAgreedOffPlatform,
	}
	for _, m := range critical {
		if !m.IsCritical() {
			t.Errorf("%s должна быть критической", m)
		}
	}

	soft := []Mistake{MistakeSharedPhone, MistakeMovedToMessenger}
	for _, m := range soft {
		if m.IsCritical() {
			t.Errorf("%s не должна быть критической", m)
		}
	}
}

func TestDecideCriticalFails(t *testing.T) {
	o := Decide([]Mistake{MistakeSharedPhone, MistakeLeakedSMSCode}, EndLimit, 2)

	if o.Verdict != VerdictFailed {
		t.Errorf("вердикт = %v, ожидался failed", o.Verdict)
	}
	if !strings.Contains(o.Explanation, "код из СМС") {
		t.Errorf("пояснение должно объяснять именно критическую ошибку: %q", o.Explanation)
	}
}

func TestDecideCleanPasses(t *testing.T) {
	o := Decide(nil, EndTacticsDone, 6)

	if o.Verdict != VerdictPassed {
		t.Errorf("вердикт = %v, ожидался passed", o.Verdict)
	}
	if ExamScore(o) != 1 {
		t.Errorf("examScore = %v, ожидался 1", ExamScore(o))
	}
}

func TestDecideSoftMistakesStillPass(t *testing.T) {
	o := Decide([]Mistake{MistakeSharedPhone}, EndTacticsDone, 6)

	if o.Verdict != VerdictPassed {
		t.Errorf("некритичная уступка не должна валить экзамен, получено %v", o.Verdict)
	}
	if ExamScore(o) != 0.5 {
		t.Errorf("examScore = %v, ожидался 0.5", ExamScore(o))
	}
}

func TestDecideRefusalPasses(t *testing.T) {
	o := Decide(nil, EndRefused, 3)

	if o.Verdict != VerdictPassed {
		t.Errorf("твёрдый отказ должен давать passed, получено %v", o.Verdict)
	}
	if o.Reason != EndRefused {
		t.Errorf("причина = %v, ожидалась EndRefused", o.Reason)
	}
	if !strings.Contains(o.Explanation, "вышли из разговора") {
		t.Errorf("пояснение должно объяснять именно выход из разговора: %q", o.Explanation)
	}
	if ExamScore(o) != 1 {
		t.Errorf("examScore = %v, ожидался 1", ExamScore(o))
	}
}

func TestDecideUserFinishPasses(t *testing.T) {
	o := Decide(nil, EndUserFinish, 4)

	if o.Verdict != VerdictPassed {
		t.Errorf("добровольный выход без ошибок = passed, получено %v", o.Verdict)
	}
	if ExamScore(o) != 1 {
		t.Errorf("examScore = %v, ожидался 1", ExamScore(o))
	}
}

func TestDecideImmediateExitIsHalfScored(t *testing.T) {
	o := Decide(nil, EndUserFinish, 0)

	if o.Verdict != VerdictPassed {
		t.Errorf("вердикт = %v, ожидался passed", o.Verdict)
	}
	if ExamScore(o) != 0.5 {
		t.Errorf("выход до первого приёма не должен давать полный балл, получено %v", ExamScore(o))
	}
	if !strings.Contains(o.Explanation, "не успели") {
		t.Errorf("пояснение должно отмечать, что проверки не было: %q", o.Explanation)
	}
}

func TestDecideCriticalWinsOverRefusal(t *testing.T) {
	o := Decide([]Mistake{MistakeLeakedCardData}, EndRefused, 3)

	if o.Verdict != VerdictFailed {
		t.Errorf("критическая ошибка важнее отказа, получено %v", o.Verdict)
	}
	if o.Reason != EndCritical {
		t.Errorf("причина = %v, ожидалась EndCritical", o.Reason)
	}
}

func TestScoreFormula(t *testing.T) {
	cases := []struct {
		correct, total int
		examScore      float64
		want           int
	}{
		{6, 6, 1, 100},
		{6, 6, 0, 60},
		{0, 6, 1, 40},
		{3, 6, 0.5, 50},
		{0, 6, 0, 0},
		{0, 0, 1, 40},
	}
	for _, c := range cases {
		if got := Score(c.correct, c.total, c.examScore); got != c.want {
			t.Errorf("Score(%d/%d, %v) = %d, ожидалось %d",
				c.correct, c.total, c.examScore, got, c.want)
		}
	}
}

func TestGrade(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "Эксперт"}, {90, "Эксперт"},
		{89, "Уверенный"}, {70, "Уверенный"},
		{69, "Осторожный"}, {40, "Осторожный"},
		{39, "Новичок"}, {0, "Новичок"},
	}
	for _, c := range cases {
		if got := Grade(c.score); got != c.want {
			t.Errorf("Grade(%d) = %q, ожидалось %q", c.score, got, c.want)
		}
	}
}

func TestFallbackTipsMatchMistakes(t *testing.T) {
	tips := FallbackTips([]Mistake{MistakeLeakedSMSCode})

	if len(tips) == 0 {
		t.Fatal("советы не должны быть пустыми")
	}
	if !strings.Contains(strings.Join(tips, " "), "Код из СМС") {
		t.Errorf("совет должен касаться кода из СМС: %v", tips)
	}
}

func TestFallbackTipsNeverEmptyAndCapped(t *testing.T) {
	if len(FallbackTips(nil)) == 0 {
		t.Error("при отсутствии ошибок совет всё равно должен быть")
	}

	all := []Mistake{
		MistakeLeakedSMSCode, MistakeLeakedCardData, MistakeAgreedPayLink,
		MistakeAgreedOffPlatform, MistakeSharedPhone, MistakeMovedToMessenger,
	}
	if got := len(FallbackTips(all)); got > 5 {
		t.Errorf("советов = %d, максимум 5", got)
	}
}

func TestHasCritical(t *testing.T) {
	if HasCritical(nil) {
		t.Error("пустой список не содержит критических ошибок")
	}
	if HasCritical([]Mistake{MistakeSharedPhone}) {
		t.Error("мягкая уступка не критическая")
	}
	if !HasCritical([]Mistake{MistakeSharedPhone, MistakeLeakedCardData}) {
		t.Error("критическая ошибка в списке должна находиться")
	}
}
