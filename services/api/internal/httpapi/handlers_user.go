package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/avito-antifraud/api/internal/apierr"
)

type userPayload struct {
	Name   string `json:"name"`
	Age    string `json:"age"`
	Gender string `json:"gender"`
	Buyer  any    `json:"buyer"`
	Seller any    `json:"seller"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.currentUser(r)
	if err != nil {
		if errors.Is(err, apierr.ErrUserNotFound) {
			writeJSON(w, map[string]any{"exists": false, "user": nil})
			return
		}
		s.fail(w, "выборка пользователя", err)
		return
	}

	buyer, seller, err := s.progressPair(r.Context(), user.ID)
	if err != nil {
		s.fail(w, "выборка прогресса", err)
		return
	}

	writeJSON(w, map[string]any{
		"exists": true,
		"user": userPayload{
			Name:   user.Name,
			Age:    user.AgeGroup,
			Gender: user.Gender,
			Buyer:  buyer,
			Seller: seller,
		},
	})
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Age      string `json:"age"`
		AgeGroup string `json:"ageGroup"`
		Gender   string `json:"gender"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		apierr.Write(w, apierr.BadRequest("имя обязательно"))
		return
	}
	if len([]rune(req.Name)) > 100 {
		apierr.Write(w, apierr.BadRequest("имя слишком длинное"))
		return
	}

	age := strings.TrimSpace(req.Age)
	if age == "" {
		age = strings.TrimSpace(req.AgeGroup)
	}

	gender := strings.TrimSpace(req.Gender)
	if gender != "male" && gender != "female" {
		apierr.Write(w, apierr.BadRequest("gender должен быть male или female"))
		return
	}

	token := s.token(r)
	if token == "" {
		generated, err := newToken()
		if err != nil {
			s.fail(w, "генерация токена", err)
			return
		}
		token = generated
	}

	user, err := s.store.UpsertUser(r.Context(), token, req.Name, age, gender)
	if err != nil {
		s.fail(w, "создание пользователя", err)
		return
	}
	s.setCookie(w, r, token)

	buyer, seller, err := s.progressPair(r.Context(), user.ID)
	if err != nil {
		s.fail(w, "выборка прогресса", err)
		return
	}

	writeJSON(w, map[string]any{
		"user": userPayload{
			Name:   user.Name,
			Age:    user.AgeGroup,
			Gender: user.Gender,
			Buyer:  buyer,
			Seller: seller,
		},
	})
}

func (s *Server) handleResetProgress(w http.ResponseWriter, r *http.Request) {
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

	if err := s.store.ResetProgress(r.Context(), user.ID, role); err != nil {
		s.fail(w, "сброс прогресса", err)
		return
	}

	progress, err := s.store.Progress(r.Context(), user.ID, role)
	if err != nil {
		s.fail(w, "выборка прогресса", err)
		return
	}

	writeJSON(w, progress.Public())
}
