package domain

import "testing"

func scenarios() []Scenario {
	opts := func(correctAt int) []Option {
		o := []Option{
			{Text: "a", Verdict: VerdictDangerous, Outcome: "плохо"},
			{Text: "b", Verdict: VerdictRisky, Outcome: "так себе"},
		}
		o[correctAt] = Option{Text: o[correctAt].Text, Verdict: VerdictSafe, Outcome: "хорошо"}
		return o
	}
	return []Scenario{
		{ID: 1, Title: "т1", Question: "в1", Options: opts(1), CorrectOption: 1,
			Explanation: "п1", RedFlags: []string{"флаг-1"}},
		{ID: 2, Title: "т2", Question: "в2", Options: opts(0), CorrectOption: 0,
			Explanation: "п2", RedFlags: []string{"флаг-2"}},
		{ID: 3, Title: "т3", Question: "в3", Options: opts(1), CorrectOption: 1,
			Explanation: "п3", RedFlags: []string{"флаг-1", "флаг-3"}},
	}
}

func TestScoreAllCorrect(t *testing.T) {
	res := Score(scenarios(), []Answer{{1, 1}, {2, 0}, {3, 1}})

	if res.Correct != 3 || res.Total != 3 {
		t.Fatalf("получено %d/%d, ожидалось 3/3", res.Correct, res.Total)
	}
	if res.Percent != 100 {
		t.Errorf("percent = %v, ожидалось 100", res.Percent)
	}
	if !res.Perfect {
		t.Error("Perfect должен быть true")
	}
	if res.Score != 30 || res.MaxScore != 30 {
		t.Errorf("очки = %d/%d, ожидалось 30/30", res.Score, res.MaxScore)
	}
	if len(res.Mistakes) != 0 {
		t.Errorf("ошибок быть не должно, получено %d", len(res.Mistakes))
	}
	if len(res.RedFlags) != 0 {
		t.Errorf("пропущенных признаков быть не должно, получено %d", len(res.RedFlags))
	}
}

func TestScoreRoundsPercentToOneDecimal(t *testing.T) {
	res := Score(scenarios(), []Answer{{1, 1}, {2, 0}, {3, 0}})

	if res.Percent != 66.7 {
		t.Errorf("percent = %v, ожидалось 66.7", res.Percent)
	}
	if res.Perfect {
		t.Error("Perfect должен быть false")
	}
}

func TestScoreAwardsPartialCreditForRiskyChoice(t *testing.T) {
	res := Score(scenarios()[:1], []Answer{{1, 0}})

	if res.Score != 0 {
		t.Errorf("за опасный вариант ожидалось 0 очков, получено %d", res.Score)
	}

	risky := []Scenario{{
		ID: 1, Question: "в", CorrectOption: 0,
		Options: []Option{
			{Text: "a", Verdict: VerdictSafe},
			{Text: "b", Verdict: VerdictRisky},
		},
	}}
	res = Score(risky, []Answer{{1, 1}})
	if res.Score != 4 {
		t.Errorf("за рискованный вариант ожидалось 4 очка, получено %d", res.Score)
	}
	if res.Correct != 0 {
		t.Errorf("рискованный вариант не должен считаться верным, Correct = %d", res.Correct)
	}
}

func TestScoreCountsUnansweredAsMistakes(t *testing.T) {
	res := Score(scenarios(), []Answer{{1, 1}})

	if res.Total != 3 {
		t.Fatalf("Total = %d, ожидалось 3", res.Total)
	}
	if res.Correct != 1 {
		t.Fatalf("Correct = %d, ожидалось 1", res.Correct)
	}
	if len(res.Mistakes) != 2 {
		t.Fatalf("ошибок = %d, ожидалось 2", len(res.Mistakes))
	}
	for _, m := range res.Mistakes {
		if m.Answered {
			t.Errorf("вопрос %d помечен отвеченным, хотя ответа не было", m.ScenarioID)
		}
		if m.Points != 0 {
			t.Errorf("за неотвеченный вопрос %d начислено %d очков", m.ScenarioID, m.Points)
		}
	}
}

func TestScoreIgnoresAnswersToForeignScenarios(t *testing.T) {
	res := Score(scenarios(), []Answer{{1, 1}, {2, 0}, {3, 1}, {999, 0}})

	if res.Correct != 3 || res.Total != 3 {
		t.Errorf("получено %d/%d, ожидалось 3/3", res.Correct, res.Total)
	}
}

func TestScoreIgnoresOutOfRangeOption(t *testing.T) {
	res := Score(scenarios()[:1], []Answer{{1, 42}})

	if res.Correct != 0 || res.Score != 0 {
		t.Errorf("вариант вне диапазона не должен приносить очков: %d/%d", res.Correct, res.Score)
	}
	if res.Reviews[0].Answered {
		t.Error("вариант вне диапазона не должен считаться ответом")
	}
}

func TestScoreCollectsMissedRedFlagsWithoutDuplicates(t *testing.T) {
	res := Score(scenarios(), []Answer{{1, 0}, {2, 0}, {3, 0}})

	want := map[string]bool{"флаг-1": true, "флаг-3": true}
	if len(res.RedFlags) != len(want) {
		t.Fatalf("пропущенных признаков = %v, ожидалось %v", res.RedFlags, want)
	}
	for _, f := range res.RedFlags {
		if !want[f] {
			t.Errorf("неожиданный признак %q", f)
		}
	}
}

func TestScoreReviewCarriesOutcomeOfChosenOption(t *testing.T) {
	res := Score(scenarios()[:1], []Answer{{1, 0}})

	rev := res.Reviews[0]
	if rev.YourOutcome != "плохо" {
		t.Errorf("последствие выбора = %q, ожидалось «плохо»", rev.YourOutcome)
	}
	if rev.CorrectOutcome != "хорошо" {
		t.Errorf("последствие верного варианта = %q, ожидалось «хорошо»", rev.CorrectOutcome)
	}
	if rev.Explanation != "п1" {
		t.Errorf("пояснение = %q", rev.Explanation)
	}
}

func TestScoreEmptyScenarios(t *testing.T) {
	res := Score(nil, []Answer{{1, 1}})

	if res.Total != 0 || res.Percent != 0 || res.Perfect {
		t.Errorf("на пустом наборе ожидалось 0/0, 0%%, Perfect=false, получено %+v", res)
	}
}

func TestLevelFor(t *testing.T) {
	cases := []struct {
		percent float64
		want    string
	}{
		{100, "Эксперт по безопасным сделкам"},
		{80, "Уверенный пользователь"},
		{50, "Есть пробелы"},
		{10, "Высокий риск"},
	}
	for _, c := range cases {
		if got := LevelFor(c.percent); got != c.want {
			t.Errorf("LevelFor(%v) = %q, ожидалось %q", c.percent, got, c.want)
		}
	}
}

func TestSuggestDifficulty(t *testing.T) {
	cases := []struct {
		percent float64
		want    Difficulty
	}{
		{100, DifficultyHard},
		{85, DifficultyHard},
		{84.9, DifficultyMedium},
		{50, DifficultyMedium},
		{49.9, DifficultyEasy},
		{0, DifficultyEasy},
	}
	for _, c := range cases {
		if got := SuggestDifficulty(c.percent); got != c.want {
			t.Errorf("SuggestDifficulty(%v) = %v, ожидалось %v", c.percent, got, c.want)
		}
	}
}

func TestParseRole(t *testing.T) {
	if _, err := ParseRole("buyer"); err != nil {
		t.Errorf("buyer должна разбираться: %v", err)
	}
	if _, err := ParseRole("admin"); err == nil {
		t.Error("admin не должна разбираться")
	}
}
