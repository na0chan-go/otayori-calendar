package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/na0chan-go/otayori-calendar/backend/internal/auth"
	"github.com/na0chan-go/otayori-calendar/backend/internal/config"
)

func TestGoogleLoginSetsOAuthStateCookie(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/auth/google/login", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}

	location := rec.Header().Get(echo.HeaderLocation)
	if !strings.Contains(location, "accounts.google.com") {
		t.Fatalf("expected google redirect, got %q", location)
	}
	if !strings.Contains(location, "state=") {
		t.Fatalf("expected redirect location to include state, got %q", location)
	}

	cookie := findCookie(rec.Result().Cookies(), auth.OAuthStateCookieName)
	if cookie == nil {
		t.Fatal("expected oauth state cookie to be set")
	}
	if cookie.Value == "" {
		t.Fatal("expected oauth state cookie value")
	}
	if !cookie.HttpOnly {
		t.Fatal("expected oauth state cookie to be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
}

func TestGoogleCallbackRequiresMatchingOAuthStateCookie(t *testing.T) {
	server := newTestServer()
	state, err := server.states.Create(time.Now())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+state+"&code=dummy", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !server.states.Consume(state, time.Now()) {
		t.Fatal("state should not be consumed when browser cookie is missing")
	}
}

func newTestServer() *Server {
	return NewServer(config.Config{
		AppEnv:             "test",
		Port:               "8080",
		DatabaseURL:        "postgres://unused",
		FrontendURL:        "http://localhost:5173",
		SessionSecret:      "test-secret",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://localhost:8080/auth/google/callback",
		GoogleCalendarID:   "primary",
		DefaultTimeZone:    "Asia/Tokyo",
	}, nil)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
