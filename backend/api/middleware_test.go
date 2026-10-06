package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCorsMiddlewareHandlesPreflight(t *testing.T) {
	called := false
	handler := CorsMiddleware(func(http.ResponseWriter, *http.Request) { called = true })
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest(http.MethodOptions, "/devices", nil))
	if response.Code != http.StatusOK || called {
		t.Fatalf("preflight should return 200 without invoking handler")
	}
	if response.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("missing CORS origin header")
	}
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "middleware-test-secret")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()})
	signed, err := token.SignedString([]byte("middleware-test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	called := false
	handler := AuthMiddleware(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/devices", nil)
	request.Header.Set("Authorization", "Bearer "+signed)
	response := httptest.NewRecorder()
	handler(response, request)
	if !called {
		t.Fatalf("valid token was rejected: %d %s", response.Code, response.Body.String())
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	response := httptest.NewRecorder()
	AuthMiddleware(func(http.ResponseWriter, *http.Request) {})(response, httptest.NewRequest(http.MethodGet, "/devices", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}
