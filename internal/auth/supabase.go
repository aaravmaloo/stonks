package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SupabaseClient struct {
	baseURL    string
	anonKey    string
	httpClient *http.Client
}

type Session struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	TokenType    string       `json:"token_type"`
	User         SupabaseUser `json:"user"`
}

type SupabaseUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type signUpResponse struct {
	Session
}

func NewSupabaseClient(baseURL, anonKey string) *SupabaseClient {
	return &SupabaseClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		anonKey: anonKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *SupabaseClient) SignUp(ctx context.Context, email, password string) (Session, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	var out signUpResponse
	if err := c.postJSON(ctx, "/auth/v1/signup", payload, &out); err != nil {
		return Session{}, err
	}
	return out.Session, nil
}

func (c *SupabaseClient) Login(ctx context.Context, email, password string) (Session, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}
	var out Session
	if err := c.postJSON(ctx, "/auth/v1/token?grant_type=password", payload, &out); err != nil {
		return Session{}, err
	}
	return out, nil
}

func (c *SupabaseClient) VerifyAccessToken(ctx context.Context, accessToken string) (SupabaseUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return SupabaseUser{}, err
	}
	req.Header.Set("apikey", c.anonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SupabaseUser{}, fmt.Errorf("verify token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return SupabaseUser{}, fmt.Errorf("verify token status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var user SupabaseUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return SupabaseUser{}, fmt.Errorf("decode user: %w", err)
	}
	return user, nil
}

func (c *SupabaseClient) postJSON(ctx context.Context, path string, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.anonKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("supabase request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("supabase status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
