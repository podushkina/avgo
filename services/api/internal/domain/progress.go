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

type TrainingProgress struct {
	CurrentStep int `json:"currentStep"`
	TotalSteps  int `json:"totalSteps"`
}

type RoleProgress struct {
	Training         TrainingProgress `json:"training"`
	IsExamPassed     bool             `json:"isExamPassed"`
	Status           Status           `json:"status"`
	IsTrainingPassed bool             `json:"isTrainingPassed"`
}

func NewRoleProgress(status Status, currentStep, totalSteps int) RoleProgress {
	if currentStep < 0 {
		currentStep = 0
	}
	if totalSteps > 0 && currentStep > totalSteps {
		currentStep = totalSteps
	}
	return RoleProgress{
		Training:         TrainingProgress{CurrentStep: currentStep, TotalSteps: totalSteps},
		IsExamPassed:     status.IsExamPassed(),
		Status:           status,
		IsTrainingPassed: status.IsTrainingPassed(),
	}
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
