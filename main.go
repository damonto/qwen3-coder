package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	token         string
	tokenPath     string
	listenAddress string
	tm            *TokenManager
)

type Model struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerBy string `json:"owner_by"`
	Created int64  `json:"created"`
	Object  string `json:"object"`
}

type ModelResponse struct {
	Data   []Model `json:"data"`
	Object string  `json:"object"`
}

func init() {
	flag.StringVar(&listenAddress, "listen", ":9527", "listen address")
	flag.StringVar(&token, "token", "", "The token (API key) for authentication")
	flag.StringVar(&tokenPath, "token-path", "./token.json", "The path where the token is stored")
	flag.Parse()
}

func main() {
	tm = NewToken(tokenPath)
	if err := tm.Authorize(); err != nil {
		panic(err)
	}

	slog.Info("starting server", "listenAddress", listenAddress)

	http.HandleFunc("/v1/", withAuth(forward))
	http.HandleFunc("/v1/models", withAuth(func(w http.ResponseWriter, r *http.Request) {
		response := ModelResponse{
			Data: []Model{
				{ID: "qwen3-coder-plus", Name: "Qwen3-Coder", OwnerBy: "qwen", Created: 1732711466, Object: "model"},
				{ID: "qwen3-coder-flash", Name: "Qwen3-Coder Flash", OwnerBy: "qwen", Created: 1732711466, Object: "model"},
			},
			Object: "list",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
	}))
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		panic(err)
	}
}

func forward(w http.ResponseWriter, r *http.Request) {
	token, err := tm.GetToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	target := token.GetResourceURL()
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery
	targetURL := target.String()

	slog.Info("forwarding request", "targetURL", targetURL)

	if err := forwardRequest(w, r, targetURL, token); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			if _, err := tm.Refresh(token); err != nil {
				respondError(w, http.StatusInternalServerError, err)
				return
			}
			if err := forwardRequest(w, r, targetURL, token); err != nil {
				respondError(w, http.StatusInternalServerError, err)
				return
			}
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}
}

func forwardRequest(w http.ResponseWriter, r *http.Request, target string, token *DeviceToken) error {
	request, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		return err
	}

	for name, values := range r.Header {
		for _, v := range values {
			request.Header.Add(name, v)
		}
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	for name, values := range response.Header {
		for _, v := range values {
			w.Header().Set(name, v)
		}
	}
	w.WriteHeader(response.StatusCode)

	_, copyErr := io.Copy(w, response.Body)
	return copyErr
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
