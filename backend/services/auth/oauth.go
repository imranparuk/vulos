package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// OAuthProvider defines an OAuth2 provider (Google, GitHub, etc.)
type OAuthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
	RedirectURI  string
}

// OAuthToken is the token response from the provider.
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// OAuthUserInfo is the normalized user info from any provider.
type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
	Picture    string
}

// Providers returns all configured OAuth providers.
func Providers() map[string]*OAuthProvider {
	providers := make(map[string]*OAuthProvider)

	// Google
	if clientID := os.Getenv("GOOGLE_CLIENT_ID"); clientID != "" {
		providers["google"] = &OAuthProvider{
			Name:         "google",
			ClientID:     clientID,
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:       []string{"openid", "email", "profile"},
			RedirectURI:  getenv("OAUTH_REDIRECT_BASE", "http://localhost:8080") + "/api/auth/callback/google",
		}
	}

	// GitHub
	if clientID := os.Getenv("GITHUB_CLIENT_ID"); clientID != "" {
		providers["github"] = &OAuthProvider{
			Name:         "github",
			ClientID:     clientID,
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			AuthURL:      "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			UserInfoURL:  "https://api.github.com/user",
			Scopes:       []string{"user:email"},
			RedirectURI:  getenv("OAUTH_REDIRECT_BASE", "http://localhost:8080") + "/api/auth/callback/github",
		}
	}

	return providers
}

// AuthURL returns the URL to redirect the user to for login.
func (p *OAuthProvider) AuthorizeURL(state string) string {
	v := url.Values{}
	v.Set("client_id", p.ClientID)
	v.Set("redirect_uri", p.RedirectURI)
	v.Set("response_type", "code")
	v.Set("scope", joinScopes(p.Scopes))
	v.Set("state", state)
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")
	return p.AuthURL + "?" + v.Encode()
}

// Exchange trades an authorization code for tokens.
func (p *OAuthProvider) Exchange(ctx context.Context, code string) (*OAuthToken, error) {
	v := url.Values{}
	v.Set("client_id", p.ClientID)
	v.Set("client_secret", p.ClientSecret)
	v.Set("code", code)
	v.Set("redirect_uri", p.RedirectURI)
	v.Set("grant_type", "authorization_code")

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "POST", p.TokenURL, nil)
	req.URL.RawQuery = v.Encode()
	req.Header.Set("Accept", "application/json")

	// GitHub needs form post
	if p.Name == "github" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange error %d: %s", resp.StatusCode, body)
	}

	var token OAuthToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return &token, nil
}

// FetchUserInfo gets the user profile from the provider.
func (p *OAuthProvider) FetchUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", p.UserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("userinfo error %d: %s", resp.StatusCode, body)
	}

	var raw map[string]any
	json.Unmarshal(body, &raw)

	info := &OAuthUserInfo{}
	switch p.Name {
	case "google":
		info.ProviderID = strVal(raw, "id")
		info.Email = strVal(raw, "email")
		info.Name = strVal(raw, "name")
		info.Picture = strVal(raw, "picture")
	case "github":
		info.ProviderID = fmt.Sprintf("%v", raw["id"])
		info.Email = strVal(raw, "email")
		info.Name = strVal(raw, "login")
		info.Picture = strVal(raw, "avatar_url")
	default:
		info.ProviderID = strVal(raw, "sub")
		info.Email = strVal(raw, "email")
		info.Name = strVal(raw, "name")
	}
	return info, nil
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
