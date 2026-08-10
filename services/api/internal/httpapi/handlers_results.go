package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avito-antifraud/api/internal/apierr"
	"github.com/avito-antifraud/api/internal/domain"
	"github.com/avito-antifraud/api/internal/exam"
	"github.com/avito-antifraud/api/internal/storage"
)

type trainingResult struct {
	CorrectSteps int                      `json:"correctSteps"`
	TotalSteps   int                      `json:"totalSteps"`
	Answers      []storage.TrainingAnswer `json:"answers"`
}

type examResult struct {
	Verdict          string   `json:"verdict"`
	Explanation      string   `json:"explanation"`
	CyclesPassed     int      `json:"cyclesPassed"`
	CriticalMistakes []string `json:"criticalMistakes"`
	EndReason        string   `json:"endReason"`
}

type resultPayload struct {
	Role       string         `json:"role"`
	Training   trainingResult `json:"training"`
	Exam       examResult     `json:"exam"`
	Score      int            `json:"score"`
	Grade      string         `json:"grade"`
	Strengths  []string       `json:"strengths"`
	Weaknesses []string       `json:"weaknesses"`
	Tips       []string       `json:"tips"`
}

func (s *Server) finalize(
	ctx context.Context, user storage.User, role storage.Role,
	sess storage.ExamSession, outcome domain.ExamOutcome,
) error {
	if err := s.store.FinishSession(ctx, sess.ID, outcome); err != nil {
		return err
	}
	if err := s.store.SetStatus(ctx, user.ID, role, domain.StatusForVerdict(outcome.Verdict)); err != nil {
		return err
	}

	answers, err := s.store.Answers(ctx, user.ID, role)
	if err != nil {
		return err
	}
	total, err := s.store.TotalSteps(ctx, role)
	if err != nil {
		return err
	}

	correct := 0
	for _, a := range answers {
		if a.IsCorrect {
			correct++
		}
	}

	score := domain.Score(correct, total, domain.ExamScore(outcome))
	critical := make([]string, 0, len(outcome.Mistakes))
	for _, m := range outcome.Mistakes {
		if m.IsCritical() {
			critical = append(critical, string(m))
		}
	}

	messages, err := s.store.Messages(ctx, sess.ID)
	if err != nil {
		return err
	}
	review := s.reviewer.Build(ctx, exam.ReviewInput{
		Outcome:      outcome,
		Messages:     toLLMHistory(messages),
		CorrectSteps: correct,
		TotalSteps:   total,
	})

	payload := resultPayload{
		Role: string(role),
		Training: trainingResult{
			CorrectSteps: correct,
			TotalSteps:   total,
			Answers:      answers,
		},
		Exam: examResult{
			Verdict:          string(outcome.Verdict),
			Explanation:      outcome.Explanation,
			CyclesPassed:     sess.Cycle,
			CriticalMistakes: critical,
			EndReason:        string(outcome.Reason),
		},
		Score:      score,
		Grade:      domain.Grade(score),
		Strengths:  review.Strengths,
		Weaknesses: review.Weaknesses,
		Tips:       review.Tips,
	}

	return s.store.SaveResult(ctx, user.ID, role, payload)
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
	}
	if r.Method == http.MethodPost && !decodeJSON(w, r, &req) {
		return
	}

	role, err := s.roleFrom(r, req.Role)
	if err != nil {
		s.fail(w, "разбор роли", err)
		return
	}
	user, err := s.currentUser(r)
	if err != nil {
		s.fail(w, "выборка пользователя", err)
		return
	}

	raw, err := s.store.Result(r.Context(), user.ID, role)
	if errors.Is(err, storage.ErrNotFound) {
		apierr.Write(w, apierr.ErrResultsNotReady)
		return
	}
	if err != nil {
		s.fail(w, "выборка результата", err)
		return
	}

	var payload resultPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		s.fail(w, "разбор результата", err)
		return
	}
	writeJSON(w, payload)
}
