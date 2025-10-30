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
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type Content struct {
	Type string `json:"type"`
}

func (r *Request) hasImage() (bool, error) {
	for _, message := range r.Messages {
		var c any
		if err := json.Unmarshal(message.Content, &c); err != nil {
			return false, err
		}
		switch c.(type) {
		case []any:
			var contents []Content
			if err := json.Unmarshal(message.Content, &contents); err != nil {
				return false, err
			}
			for _, content := range contents {
				if content.Type == "image_url" {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func rerouteImage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			respondError(w, http.StatusBadRequest, err)
			return
		}
		r.Body.Close()

		var request Request
		if err := json.Unmarshal(body, &request); err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}

		hasImage, err := request.hasImage()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}

		if hasImage && request.Model != "vision-model" {
			slog.Info("the request contains an image, rerouting to vision model", "model", "qwen3-vl-max")
			request.Model = "vision-model"
			body, err = json.Marshal(request)
			if err != nil {
				respondError(w, http.StatusInternalServerError, err)
				return
			}
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
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
