package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
}

func rerouteImage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		r.Body.Close()

		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}

		hasImage := false
		for _, message := range req.Messages {
			for _, content := range message.Content {
				if content.Type == "image_url" {
					hasImage = true
					break
				}
			}
		}

		if hasImage && req.Model != "vision-model" {
			slog.Info("The request contains an image, rerouting to vision model", "model", "qwen3-vl-max")
			var rawRequest map[string]json.RawMessage
			if err := json.Unmarshal(body, &rawRequest); err != nil {
				respondError(w, http.StatusBadRequest, err)
				return
			}
			rawRequest["model"] = json.RawMessage(`"vision-model"`)

			newBody, err := json.Marshal(rawRequest)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(newBody))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		next(w, r)
	}
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondError(w, http.StatusUnauthorized, ErrMissingToken)
			return
		}
		bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
		if bearerToken != token {
			respondError(w, http.StatusUnauthorized, ErrInvalidToken)
			return
		}
		next(w, r)
	}
}
