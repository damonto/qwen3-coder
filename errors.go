package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type HTTPErrorResponse struct {
	Err Error `json:"error"`
}

func (e HTTPErrorResponse) Error() string {
	return e.Err.Error()
}

type Error struct {
	Code    string `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Type, e.Code, e.Message)
}

var (
	ErrUnauthorized   = HTTPErrorResponse{Err: Error{Code: "unauthorized", Type: "invalid_request_error", Message: "unauthorized"}}
	ErrInternalServer = HTTPErrorResponse{Err: Error{Code: "internal_server_error", Type: "invalid_request_error", Message: "internal server error"}}
	ErrMissingToken   = HTTPErrorResponse{Err: Error{Code: "missing_api_key", Type: "invalid_request_error", Message: "missing access token"}}
	ErrInvalidToken   = HTTPErrorResponse{Err: Error{Code: "invalid_api_key", Type: "invalid_request_error", Message: "invalid access token or token expired"}}
)

func respondError(w http.ResponseWriter, statusCode int, err error) {
	httpErr, ok := err.(HTTPErrorResponse)
	if !ok {
		httpErr = HTTPErrorResponse{
			Err: Error{
				Code:    "internal_server_error",
				Type:    "invalid_request_error",
				Message: err.Error(),
			},
		}
	}

	if statusCode == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer realm=\"Qwen3 Coder\"")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(httpErr); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}
