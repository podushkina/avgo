package httpapi

import (
	"errors"
	"net/http"

	"github.com/avito-antifraud/api/internal/apierr"
	"github.com/avito-antifraud/api/internal/storage"
)

type variant struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

func (s *Server) handleCurrentStep(w http.ResponseWriter, r *http.Request) {
	role, err := s.roleFrom(r, "")
	if err != nil {
		s.fail(w, "разбор роли", err)
		return
	}

	user, err := s.currentUser(r)
	if err != nil {
		s.fail(w, "выборка пользователя", err)
		return
	}

	progress, err := s.store.Progress(r.Context(), user.ID, role)
	if err != nil {
		s.fail(w, "выборка прогресса", err)
		return
	}
	if progress.Status.IsTrainingPassed() {
		apierr.Write(w, apierr.ErrTrainingAlreadyPassed.WithDetails(progress.Public()))
		return
	}

	stepNo := progress.CurrentStep + 1
	step, err := s.store.StepByNumber(r.Context(), role, stepNo)
	if errors.Is(err, storage.ErrNotFound) {
		apierr.Write(w, apierr.ErrTrainingAlreadyPassed.WithDetails(progress.Public()))
		return
	}
	if err != nil {
		s.fail(w, "выборка шага", err)
		return
	}

	variants := make([]variant, 0, len(step.Options))
	for _, o := range step.Options {
		variants = append(variants, variant{ID: o.ID, Text: o.Text})
	}

	writeJSON(w, map[string]any{
		"currentStep": step.StepNo,
		"totalSteps":  progress.TotalSteps,
		"productName": step.ProductName,
		"message":     step.Message,
		"variants":    variants,
	})
}

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role        string `json:"role"`
		AnswerID    int64  `json:"answer_id"`
		AnswerIDAlt int64  `json:"answerId"`
		StepNumber  int    `json:"stepNumber"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	role, err := s.roleFrom(r, req.Role)
	if err != nil {
		s.fail(w, "разбор роли", err)
		return
	}

	answerID := req.AnswerID
	if answerID == 0 {
		answerID = req.AnswerIDAlt
	}
	if answerID == 0 {
		apierr.Write(w, apierr.BadRequest("answer_id обязателен"))
		return
	}

	user, err := s.currentUser(r)
	if err != nil {
		s.fail(w, "выборка пользователя", err)
		return
	}

	step, isCorrect, answered, err := s.store.RecordAnswer(
		r.Context(), user.ID, role, req.StepNumber, answerID)

	switch {
	case errors.Is(err, storage.ErrStepMismatch):
		progress, pErr := s.store.Progress(r.Context(), user.ID, role)
		if pErr != nil {
			s.fail(w, "выборка прогресса", pErr)
			return
		}
		apierr.Write(w, apierr.ErrStepMismatch.WithDetails(progress.Public()))
		return
	case errors.Is(err, storage.ErrNotFound):
		apierr.Write(w, apierr.ErrInvalidOption)
		return
	case err != nil:
		s.fail(w, "сохранение ответа", err)
		return
	}

	total, err := s.store.TotalSteps(r.Context(), role)
	if err != nil {
		s.fail(w, "подсчёт шагов", err)
		return
	}

	writeJSON(w, map[string]any{
		"isCorrect":          isCorrect,
		"explanation":        step.Explanation,
		"correctId":          step.CorrectID(),
		"currentStep":        answered,
		"totalSteps":         total,
		"isTrainingFinished": answered >= total,
	})
}
