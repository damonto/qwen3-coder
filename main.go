package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var (
	apiKey        string
	tokenPath     string
	listenAddress string
	tm            *TokenManager
	Version       string
)

type Model struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnedBy string `json:"owned_by"`
	Created int64  `json:"created"`
	Object  string `json:"object"`
}

type ModelResponse struct {
	Data   []Model `json:"data"`
	Object string  `json:"object"`
}

func init() {
	flag.StringVar(&listenAddress, "listen", ":9527", "listen address")
	flag.StringVar(&apiKey, "api-key", "", "The API key for authentication")
	flag.StringVar(&tokenPath, "token-path", "./data/token.json", "The path where the token is stored")
}

func main() {
	flag.Parse()

	tm = NewToken(tokenPath)
	if err := tm.Authorize(); err != nil {
		panic(err)
	}

	slog.Info("starting server", "listenAddress", listenAddress, "version", Version)

	http.HandleFunc("/v1/models", withAuth(func(w http.ResponseWriter, r *http.Request) {
		response := ModelResponse{
			Data: []Model{
				{ID: "coder-model", Name: "Qwen3-Coder-Plus", OwnedBy: "qwen", Created: 1732711466, Object: "model"},
				{ID: "vision-model", Name: "Qwen3-VL-Max", OwnedBy: "qwen", Created: 1732711466, Object: "model"},
			},
			Object: "list",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
	}))
	http.HandleFunc("/v1/chat/completions", withAuth(rerouteImage(forward)))

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
	target := url.URL{Scheme: "https", Host: token.ResourceURL}
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery
	targetURL := target.String()

	slog.Info("forwarding request", "targetURL", targetURL)

	if err := forwardRequest(w, r, targetURL, token); err != nil {
		if errors.Is(err, ErrUnauthorized) {
			if _, err := tm.Refresh(token); err != nil {
				respondError(w, 0, err)
				return
			}
			if err := forwardRequest(w, r, targetURL, token); err != nil {
				respondError(w, 0, err)
				return
			}
		} else {
			respondError(w, 0, err)
			return
		}
	}
}

func forwardRequest(w http.ResponseWriter, r *http.Request, target string, token *DeviceToken) error {
	request, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		return err
	}

	copyHeaders(request.Header, r.Header)

	request.Header.Set("User-Agent", "QwenCode/0.4.0 (linux; x64)")
	if token != nil && token.AccessToken != "" {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	}

	client := &http.Client{
		Timeout:   0,
		Transport: http.DefaultTransport,
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	copyHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)

	if strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		return forwardStreamingResponse(w, response)
	}
	_, err = io.Copy(w, response.Body)
	return err
}

func forwardStreamingResponse(w http.ResponseWriter, response *http.Response) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("ResponseWriter does not support flushing")
	}
	scanner := bufio.NewScanner(response.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		slog.Debug("received", "text", scanner.Text())
		if _, err := w.Write(append(scanner.Bytes(), '\n')); err != nil {
			return err
		}
		flusher.Flush()
	}
	return scanner.Err()
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
