package server

import (
	"errors"
	"net/http"

	"github.com/Teerapat-M/rsp-project/internal/hub"
	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/passcode"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/gin-gonic/gin"
)

type apiError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Passcode string `json:"passcode,omitempty"`
}

func writeAPIError(c *gin.Context, status int, code, message string, extra map[string]any) {
	body := apiError{Code: code, Message: message}
	if extra != nil {
		if p, ok := extra["passcode"].(string); ok {
			body.Passcode = p
		}
	}
	c.AbortWithStatusJSON(status, body)
}

func writeErr(c *gin.Context, err error, passcode string) {
	status, code, msg := classify(err)
	extra := map[string]any{}
	if passcode != "" && (code == "WRONG_PASSCODE" || code == "ROOM_FULL") {
		extra["passcode"] = passcode
	}
	writeAPIError(c, status, code, msg, extra)
}

func classify(err error) (int, string, string) {
	switch {
	case errors.Is(err, ident.ErrNoPlayer):
		return http.StatusUnauthorized, "UNAUTHORIZED", "missing player identity"
	case errors.Is(err, rooms.ErrWrongPasscode):
		return http.StatusNotFound, "WRONG_PASSCODE", "We couldn't find a room with that passcode."
	case errors.Is(err, rooms.ErrRoomFull):
		return http.StatusConflict, "ROOM_FULL", "Room already has 2 players."
	case errors.Is(err, rooms.ErrNotFound), errors.Is(err, rooms.ErrClosed):
		return http.StatusNotFound, "NOT_FOUND", "Room not found"
	case errors.Is(err, rooms.ErrNotInRoom):
		return http.StatusNotFound, "NOT_FOUND", "Not in this room"
	case errors.Is(err, rooms.ErrInvalidWins), errors.Is(err, rooms.ErrInvalidTurn):
		return http.StatusBadRequest, "VALIDATION", err.Error()
	case errors.Is(err, passcode.ErrExhausted):
		return http.StatusServiceUnavailable, "CONFLICT", "could not allocate a passcode"
	case errors.Is(err, hub.ErrPhase3):
		return http.StatusNotImplemented, "NOT_IMPLEMENTED", err.Error()
	default:
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
}
