package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPErrorResponse struct {
	Err Error `json:"error"`
}

func (e HTTPErrorResponse) Error() string {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(e); err != nil {
		return err.Error()
	}
	return buf.String()
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
	if _, ok := err.(HTTPErrorResponse); !ok {
		err = HTTPErrorResponse{Err: Error{Code: "internal_server_error", Type: "invalid_request_error", Message: err.Error()}}
	}
	// Delete the content-type first
	w.Header().Del("Content-Type")
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, err.Error(), statusCode)
}
