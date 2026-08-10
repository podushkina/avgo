package domain

type Status string

const (
	StatusNotStarted         Status = "not_started"
	StatusTrainingInProgress Status = "training_in_progress"
	StatusTrainingPassed     Status = "training_passed"
	StatusExamInProgress     Status = "exam_in_progress"
	StatusExamPassed         Status = "exam_passed"
	StatusExamFailed         Status = "exam_failed"
)

func (s Status) IsTrainingPassed() bool {
	switch s {
	case StatusTrainingPassed, StatusExamInProgress, StatusExamPassed, StatusExamFailed:
		return true
	default:
		return false
	}
}

func (s Status) IsExamPassed() bool { return s == StatusExamPassed }

func (s Status) IsExamFinished() bool {
	return s == StatusExamPassed || s == StatusExamFailed
}

// RoleProgress — публичный прогресс роли в API. Указатель шагов обучения
// наружу не отдаётся: клиенту достаточно status, а шаги живут в /training/*.
type RoleProgress struct {
	Status Status `json:"status"`
}

func NextStatusAfterAnswer(current Status, answeredSteps, totalSteps int) Status {
	if current.IsTrainingPassed() {
		return current
	}
	if totalSteps > 0 && answeredSteps >= totalSteps {
		return StatusTrainingPassed
	}
	return StatusTrainingInProgress
}

func StatusForVerdict(v Verdict) Status {
	if v == VerdictPassed {
		return StatusExamPassed
	}
	return StatusExamFailed
}
