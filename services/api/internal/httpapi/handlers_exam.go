package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/avito-antifraud/api/internal/apierr"
	"github.com/avito-antifraud/api/internal/domain"
	"github.com/avito-antifraud/api/internal/exam"
	"github.com/avito-antifraud/api/internal/llm"
	"github.com/avito-antifraud/api/internal/prompt"
	"github.com/avito-antifraud/api/internal/sanitize"
	"github.com/avito-antifraud/api/internal/storage"
)

const (
	authorScammer = "scammer"
	authorUser    = "user"

	examDifficulty = prompt.DifficultyMedium
)

func (s *Server) requireTrainingPassed(
	r *http.Request, user storage.User, role storage.Role,
) error {
	progress, err := s.store.Progress(r.Context(), user.ID, role)
	if err != nil {
		return err
	}
	if !progress.Status.IsTrainingPassed() {
		return apierr.ErrTrainingNotPassed.WithDetails(progress.Public())
	}
	return nil
}

func (s *Server) examStateResponse(
	ctx context.Context, sess storage.ExamSession, opening string,
) (map[string]any, error) {
	messages, err := s.store.Messages(ctx, sess.ID)
	if err != nil {
		return nil, err
	}

	message := opening
	if message == "" {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Author == authorScammer {
				message = messages[i].Text
				break
			}
		}
	}

	var verdict, explanation any
	if sess.Verdict != nil {
		verdict = *sess.Verdict
	}
	if sess.Explanation != nil {
		explanation = *sess.Explanation
	}

	return map[string]any{
		"sessionId":   sess.ID,
		"message":     message,
		"messages":    messages,
		"isFinished":  sess.Status == "finished",
		"verdict":     verdict,
		"explanation": explanation,
		"cycle":       sess.Cycle,
		"maxCycles":   s.cfg.ExamMaxCycles,
	}, nil
}

func (s *Server) openSession(
	r *http.Request, user storage.User, role storage.Role,
) (storage.ExamSession, string, error) {
	persona := storage.Persona{Name: user.Name, AgeGroup: user.AgeGroup, Gender: user.Gender}

	sess, err := s.store.CreateSession(r.Context(), user.ID, role, persona)
	if err != nil {
		return storage.ExamSession{}, "", err
	}

	opening, err := s.generate(r.Context(), sess, role)
	if err != nil {
		return storage.ExamSession{}, "", err
	}
	if _, err := s.store.AddMessage(r.Context(), sess.ID, authorScammer, opening); err != nil {
		return storage.ExamSession{}, "", err
	}
	if err := s.store.SetStatus(r.Context(), user.ID, role, domain.StatusExamInProgress); err != nil {
		return storage.ExamSession{}, "", err
	}
	return sess, opening, nil
}

func (s *Server) handleExamStart(w http.ResponseWriter, r *http.Request) {
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
	if err := s.requireTrainingPassed(r, user, role); err != nil {
		s.fail(w, "проверка обучения", err)
		return
	}

	if err := s.store.ExpireStaleSessions(r.Context(), s.cfg.ExamSessionTTL); err != nil {
		s.log.Warn("протухшие сессии", "error", err)
	}

	sess, err := s.store.ActiveSession(r.Context(), user.ID, role)
	if err == nil {
		body, bErr := s.examStateResponse(r.Context(), sess, "")
		if bErr != nil {
			s.fail(w, "сборка ответа", bErr)
			return
		}
		writeJSON(w, body)
		return
	}
	if !errors.Is(err, storage.ErrNoActive) {
		s.fail(w, "выборка сессии", err)
		return
	}

	sess, opening, err := s.openSession(r, user, role)
	if err != nil {
		s.fail(w, "создание сессии", err)
		return
	}
	body, err := s.examStateResponse(r.Context(), sess, opening)
	if err != nil {
		s.fail(w, "сборка ответа", err)
		return
	}
	writeJSON(w, body)
}

func (s *Server) handleExamRestart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
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
	if err := s.requireTrainingPassed(r, user, role); err != nil {
		s.fail(w, "проверка обучения", err)
		return
	}

	if err := s.store.AbandonActive(r.Context(), user.ID, role); err != nil {
		s.fail(w, "закрытие сессии", err)
		return
	}

	sess, opening, err := s.openSession(r, user, role)
	if err != nil {
		s.fail(w, "создание сессии", err)
		return
	}
	body, err := s.examStateResponse(r.Context(), sess, opening)
	if err != nil {
		s.fail(w, "сборка ответа", err)
		return
	}
	writeJSON(w, body)
}

func (s *Server) handleExamMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
		Text string `json:"text"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	role, err := s.roleFrom(r, req.Role)
	if err != nil {
		s.fail(w, "разбор роли", err)
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		apierr.Write(w, apierr.BadRequest("текст сообщения пуст"))
		return
	}
	if len([]rune(text)) > s.cfg.MessageLimit {
		apierr.Write(w, apierr.ErrMessageTooLong)
		return
	}

	user, err := s.currentUser(r)
	if err != nil {
		s.fail(w, "выборка пользователя", err)
		return
	}

	sess, err := s.store.ActiveSession(r.Context(), user.ID, role)
	if errors.Is(err, storage.ErrNoActive) {
		apierr.Write(w, apierr.ErrSessionNotFound)
		return
	}
	if err != nil {
		s.fail(w, "выборка сессии", err)
		return
	}

	if !s.msgLimiter.Allow(sess.ID.String()) {
		apierr.Write(w, apierr.ErrRateLimited)
		return
	}

	if _, err := s.store.AddMessage(r.Context(), sess.ID, authorUser, text); err != nil {
		s.fail(w, "сохранение сообщения", err)
		return
	}

	verdict := s.classifier.Classify(r.Context(), text)
	mistakes := mergeMistakes(sess.Mistakes, verdict.Mistakes())

	cycle, err := s.store.BumpCycle(r.Context(), sess.ID, mistakes)
	if err != nil {
		s.fail(w, "счётчик цикла", err)
		return
	}

	tacticsFaced := tacticsFaced(cycle)
	reason, ended := endReason(mistakes, verdict, cycle, tacticsFaced, s.cfg.ExamMaxCycles)

	reply, err := s.generate(r.Context(), sess, role)
	if err != nil {
		s.fail(w, "ответ модели", err)
		return
	}
	if _, err := s.store.AddMessage(r.Context(), sess.ID, authorScammer, reply); err != nil {
		s.fail(w, "сохранение ответа", err)
		return
	}

	if !ended {
		writeJSON(w, map[string]any{
			"message":     reply,
			"isFinished":  false,
			"verdict":     nil,
			"explanation": nil,
			"cycle":       cycle,
			"maxCycles":   s.cfg.ExamMaxCycles,
		})
		return
	}

	outcome := domain.Decide(mistakes, reason, tacticsFaced)
	if err := s.finalize(r.Context(), user, role, sess, outcome); err != nil {
		s.fail(w, "завершение экзамена", err)
		return
	}

	writeJSON(w, map[string]any{
		"message":     reply,
		"isFinished":  true,
		"verdict":     string(outcome.Verdict),
		"explanation": outcome.Explanation,
		"cycle":       cycle,
		"maxCycles":   s.cfg.ExamMaxCycles,
	})
}

func (s *Server) handleExamFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
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

	sess, err := s.store.ActiveSession(r.Context(), user.ID, role)
	if errors.Is(err, storage.ErrNoActive) {
		apierr.Write(w, apierr.ErrSessionNotFound)
		return
	}
	if err != nil {
		s.fail(w, "выборка сессии", err)
		return
	}

	outcome := domain.Decide(sess.Mistakes, domain.EndUserFinish, tacticsFaced(sess.Cycle))
	if err := s.finalize(r.Context(), user, role, sess, outcome); err != nil {
		s.fail(w, "завершение экзамена", err)
		return
	}

	writeJSON(w, map[string]any{
		"verdict":     string(outcome.Verdict),
		"explanation": outcome.Explanation,
		"isFinished":  true,
		"cycle":       sess.Cycle,
		"maxCycles":   s.cfg.ExamMaxCycles,
	})
}

func toLLMHistory(messages []storage.ExamMessage) []llm.Message {
	out := make([]llm.Message, 0, len(messages))
	for _, m := range messages {
		role := llm.RoleAssistant
		if m.Author == authorUser {
			role = llm.RoleUser
		}
		out = append(out, llm.Message{Role: role, Content: m.Text})
	}
	return out
}

func tacticsFaced(cycle int) int {
	faced := cycle - prompt.PressureStartsAt(examDifficulty) + 1
	if faced < 0 {
		faced = 0
	}
	if faced > prompt.AttackCount() {
		faced = prompt.AttackCount()
	}
	return faced
}

func endReason(
	mistakes []domain.Mistake, v exam.Verdict, cycle, faced, maxCycles int,
) (domain.EndReason, bool) {
	switch {
	case domain.HasCritical(mistakes):
		return domain.EndCritical, true
	case v.RefusedAndEnded:
		return domain.EndRefused, true
	case faced >= prompt.AttackCount():
		return domain.EndTacticsDone, true
	case cycle >= maxCycles:
		return domain.EndLimit, true
	default:
		return "", false
	}
}

func mergeMistakes(existing, fresh []domain.Mistake) []domain.Mistake {
	seen := map[domain.Mistake]bool{}
	out := []domain.Mistake{}
	for _, group := range [][]domain.Mistake{existing, fresh} {
		for _, m := range group {
			if seen[m] {
				continue
			}
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

func (s *Server) generate(
	ctx context.Context, sess storage.ExamSession, role storage.Role,
) (string, error) {
	history, err := s.buildHistory(ctx, sess, role)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	streamer := sanitize.NewStreamer(func(chunk string) error {
		sb.WriteString(chunk)
		return nil
	})

	err = s.client.Stream(ctx, history, streamer.Push)
	if errors.Is(err, sanitize.ErrRepeat) {
		err = nil
	}
	if err == nil {
		err = streamer.Close()
	}
	if err != nil {
		s.log.Error("генерация реплики", "error", err)
		return "", apierr.ErrLLMUnavailable
	}

	reply := sanitize.TrimToSentence(sanitize.StripDirective(sb.String()))
	if reply == "" {
		return "", apierr.ErrLLMUnavailable
	}
	return reply, nil
}

func (s *Server) buildHistory(
	ctx context.Context, sess storage.ExamSession, role storage.Role,
) ([]llm.Message, error) {
	messages, err := s.store.Messages(ctx, sess.ID)
	if err != nil {
		return nil, err
	}

	promptRole := prompt.RoleSeller
	if role == storage.RoleBuyer {
		promptRole = prompt.RoleBuyer
	}

	history := []llm.Message{{
		Role:    llm.RoleSystem,
		Content: prompt.System(promptRole, examDifficulty) + personaBlock(sess.Persona),
	}}

	if len(messages) == 0 {
		history = append(history, llm.Message{
			Role:    llm.RoleSystem,
			Content: prompt.TurnDirective(examDifficulty, 1),
		})
		return history, nil
	}

	for _, m := range messages {
		role := llm.RoleAssistant
		if m.Author == authorUser {
			role = llm.RoleUser
		}
		history = append(history, llm.Message{Role: role, Content: m.Text})
	}

	history = append(history, llm.Message{
		Role:    llm.RoleSystem,
		Content: prompt.TurnDirective(examDifficulty, sess.Cycle+1),
	})
	return history, nil
}

func personaBlock(p storage.Persona) string {
	if p.Name == "" {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\nСобеседника зовут ")
	sb.WriteString(p.Name)
	sb.WriteString(". Обращайся к нему по имени хотя бы раз за диалог.")
	if p.AgeGroup != "" {
		sb.WriteString(" Возраст: ")
		sb.WriteString(p.AgeGroup)
		sb.WriteString(".")
	}
	return sb.String()
}
