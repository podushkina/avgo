package apierr

import (
	"encoding/json"
	"errors"
	"net/http"
)

const (
	CodeUserNotFound           = "USER_NOT_FOUND"
	CodeTrainingNotPassed      = "TRAINING_NOT_PASSED"
	CodeTrainingAlreadyPassed  = "TRAINING_ALREADY_PASSED"
	CodeStepMismatch           = "STEP_MISMATCH"
	CodeInvalidOption          = "INVALID_OPTION"
	CodeSessionNotFound        = "SESSION_NOT_FOUND"
	CodeSessionAlreadyFinished = "SESSION_ALREADY_FINISHED"
	CodeMessageTooLong         = "MESSAGE_TOO_LONG"
	CodeRateLimited            = "RATE_LIMITED"
	CodeResultsNotReady        = "RESULTS_NOT_READY"
	CodeLLMUnavailable         = "LLM_UNAVAILABLE"
	CodeBadRequest             = "BAD_REQUEST"
	CodeInternal               = "INTERNAL"
)

type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func (e *Error) WithDetails(d any) *Error {
	c := *e
	c.Details = d
	return &c
}

var (
	ErrUserNotFound = New(http.StatusUnauthorized, CodeUserNotFound,
		"Пользователь не найден, создайте профиль")
	ErrTrainingNotPassed = New(http.StatusConflict, CodeTrainingNotPassed,
		"Сначала нужно пройти обучение по этой роли")
	ErrTrainingAlreadyPassed = New(http.StatusConflict, CodeTrainingAlreadyPassed,
		"Обучение по этой роли уже пройдено")
	ErrStepMismatch = New(http.StatusConflict, CodeStepMismatch,
		"Шаг не совпадает с текущим")
	ErrInvalidOption = New(http.StatusBadRequest, CodeInvalidOption,
		"Вариант ответа не относится к текущему шагу")
	ErrSessionNotFound = New(http.StatusNotFound, CodeSessionNotFound,
		"Активная сессия экзамена не найдена")
	ErrSessionAlreadyFinished = New(http.StatusConflict, CodeSessionAlreadyFinished,
		"Экзамен уже завершён")
	ErrMessageTooLong = New(http.StatusBadRequest, CodeMessageTooLong,
		"Сообщение слишком длинное")
	ErrRateLimited = New(http.StatusTooManyRequests, CodeRateLimited,
		"Слишком много сообщений, подождите немного")
	ErrResultsNotReady = New(http.StatusNotFound, CodeResultsNotReady,
		"Результаты появятся после завершения экзамена")
	ErrLLMUnavailable = New(http.StatusServiceUnavailable, CodeLLMUnavailable,
		"Собеседник временно недоступен, попробуйте ещё раз")
	ErrInternal = New(http.StatusInternalServerError, CodeInternal,
		"Внутренняя ошибка сервиса")
)

func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, CodeBadRequest, message)
}

type envelope struct {
	Error *Error `json:"error"`
}

func Write(w http.ResponseWriter, err error) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		apiErr = ErrInternal
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(apiErr.Status)
	_ = json.NewEncoder(w).Encode(envelope{Error: apiErr})
}
