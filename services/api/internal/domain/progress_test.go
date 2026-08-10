package domain

import "testing"

func TestStatusFlags(t *testing.T) {
	cases := []struct {
		status         Status
		trainingPassed bool
		examPassed     bool
		examFinished   bool
	}{
		{StatusNotStarted, false, false, false},
		{StatusTrainingInProgress, false, false, false},
		{StatusTrainingPassed, true, false, false},
		{StatusExamInProgress, true, false, false},
		{StatusExamPassed, true, true, true},
		{StatusExamFailed, true, false, true},
	}
	for _, c := range cases {
		if got := c.status.IsTrainingPassed(); got != c.trainingPassed {
			t.Errorf("%s.IsTrainingPassed() = %v, ожидалось %v", c.status, got, c.trainingPassed)
		}
		if got := c.status.IsExamPassed(); got != c.examPassed {
			t.Errorf("%s.IsExamPassed() = %v, ожидалось %v", c.status, got, c.examPassed)
		}
		if got := c.status.IsExamFinished(); got != c.examFinished {
			t.Errorf("%s.IsExamFinished() = %v, ожидалось %v", c.status, got, c.examFinished)
		}
	}
}

func TestNextStatusAfterAnswer(t *testing.T) {
	if got := NextStatusAfterAnswer(StatusNotStarted, 1, 6); got != StatusTrainingInProgress {
		t.Errorf("после первого ответа = %v, ожидалось training_in_progress", got)
	}
	if got := NextStatusAfterAnswer(StatusTrainingInProgress, 6, 6); got != StatusTrainingPassed {
		t.Errorf("после последнего ответа = %v, ожидалось training_passed", got)
	}
}

func TestNextStatusDoesNotDowngradeFinishedExam(t *testing.T) {
	for _, s := range []Status{StatusExamPassed, StatusExamFailed, StatusExamInProgress} {
		if got := NextStatusAfterAnswer(s, 1, 6); got != s {
			t.Errorf("статус %v не должен откатываться, получено %v", s, got)
		}
	}
}

func TestStatusForVerdict(t *testing.T) {
	if got := StatusForVerdict(VerdictPassed); got != StatusExamPassed {
		t.Errorf("passed -> %v", got)
	}
	if got := StatusForVerdict(VerdictFailed); got != StatusExamFailed {
		t.Errorf("failed -> %v", got)
	}
}
