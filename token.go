package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	OAuthDeviceAuthEndpoint           = "https://chat.qwen.ai/api/v1/oauth2/device_authorization"
	OAuthDeviceAuthDeviceCodeEndpoint = "https://chat.qwen.ai/api/v1/oauth2/device/code"
	OAuthDeviceAuthTokenEndpoint      = "https://chat.qwen.ai/api/v1/oauth2/token"
	OAuthClientID                     = "f0304373b74a44d2b584a3fb70ca9e56"
)

type TokenManager struct {
	tokenPath string
	lock      sync.Mutex
}

type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
}

type DeviceToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	ResourceURL  string    `json:"resource_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

var (
	ErrDeviceTokenSlowDown = errors.New("slow down")
	ErrDeviceTokenPending  = errors.New("pending")
)

func NewToken(tokenPath string) *TokenManager {
	return &TokenManager{
		tokenPath: tokenPath,
	}
}

func (tm *TokenManager) Authorize() error {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	_, err := os.Stat(tm.tokenPath)
	if err != nil && os.IsNotExist(err) {
		var token DeviceToken
		if err := token.Authorize(); err != nil {
			return err
		}
		if err := tm.save(&token); err != nil {
			return err
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (tm *TokenManager) GetToken() (*DeviceToken, error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	f, err := os.OpenFile(tm.tokenPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var token DeviceToken
	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}
	if !token.IsValid() {
		if err := token.Refresh(); err != nil {
			defer tm.clear()
			return nil, err
		}
		if err := tm.save(&token); err != nil {
			return nil, err
		}
	}
	return &token, nil
}

func (tm *TokenManager) Refresh(token *DeviceToken) (*DeviceToken, error) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if err := token.Refresh(); err != nil {
		defer tm.clear()
		return nil, err
	}
	if err := tm.save(token); err != nil {
		return nil, err
	}
	return token, nil
}

func (tm *TokenManager) save(token *DeviceToken) error {
	f, err := os.OpenFile(tm.tokenPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

func (tm *TokenManager) clear() error {
	return os.Remove(tm.tokenPath)
}

func (dt *DeviceToken) IsValid() bool {
	if dt.AccessToken == "" {
		return false
	}
	return time.Until(dt.ExpiresAt) > 300*time.Second
}

func (dt *DeviceToken) Refresh() error {
	if dt.AccessToken == "" {
		return errors.New("no access token available")
	}
	if err := dt.sendRequest(OAuthDeviceAuthTokenEndpoint, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {OAuthClientID},
		"refresh_token": {dt.RefreshToken},
	}, dt); err != nil {
		return err
	}
	dt.ExpiresAt = time.Now().Add(time.Duration(dt.ExpiresIn) * time.Second)
	return nil
}

func (dt *DeviceToken) Authorize() error {
	codeVerifier := mustGenerateCodeVerifier()
	var response DeviceAuthorizationResponse
	if err := dt.sendRequest(OAuthDeviceAuthDeviceCodeEndpoint, url.Values{
		"client_id":             {OAuthClientID},
		"scope":                 {"openid profile email model.completion"},
		"code_challenge":        {generateCodeChallenge(codeVerifier)},
		"code_challenge_method": {"S256"},
	}, &response); err != nil {
		return fmt.Errorf("unable to send device code authorization request: %w", err)
	}
	fmt.Printf("Authorization URL: %s\n", response.VerificationURIComplete)
	return dt.pollDeviceToken(&response, codeVerifier)
}

func (dt *DeviceToken) pollDeviceToken(response *DeviceAuthorizationResponse, codeVerifier string) error {
	timeout := time.Duration(response.ExpiresIn) * time.Second
	deadline := time.Now().Add(timeout)
	interval := 2 * time.Second
	values := url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code":   {response.DeviceCode},
		"client_id":     {OAuthClientID},
		"code_verifier": {codeVerifier},
	}
	for {
		if time.Now().After(deadline) {
			break
		}
		err := dt.sendRequest(OAuthDeviceAuthTokenEndpoint, values, dt)
		if err != nil {
			switch {
			case errors.Is(err, ErrDeviceTokenPending):
			case errors.Is(err, ErrDeviceTokenSlowDown):
				interval = max(interval*2, 10*time.Second)
			default:
				return err
			}
			time.Sleep(interval)
			continue
		}
		if dt.AccessToken != "" {
			dt.ExpiresAt = time.Now().Add(time.Duration(dt.ExpiresIn) * time.Second)
			return nil
		}
	}
	return fmt.Errorf("device token not received after %s", timeout)
}

func (dt *DeviceToken) sendRequest(endpoint string, values url.Values, output any) error {
	request, err := http.NewRequest("POST", endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}

	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("x-request-id", mustGenerateUUID())

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		var errorResponse ErrorResponse
		if err := json.NewDecoder(response.Body).Decode(&errorResponse); err != nil {
			return err
		}
		if response.StatusCode == 400 && errorResponse.Error == "authorization_pending" {
			return ErrDeviceTokenPending
		}
		if response.StatusCode == 429 && errorResponse.Error == "slow_down" {
			return ErrDeviceTokenSlowDown
		}
		return fmt.Errorf("code: %s description: %s", errorResponse.Error, errorResponse.Description)
	}
	return json.NewDecoder(response.Body).Decode(output)
}

func mustGenerateUUID() string {
	uuid := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, uuid); err != nil {
		panic(err)
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:])
}

func mustGenerateCodeVerifier() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(codeVerifier string) string {
	h := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
