package ipace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminAuthorizeReturnsOnlyAuthorizationStatus(t *testing.T) {
	original := adminGateRequireAdmin
	t.Cleanup(func() { adminGateRequireAdmin = original })
	adminGateRequireAdmin = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "admin"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/authorize", nil)
	res := httptest.NewRecorder()
	AdminAuthorize(res, req)

	if res.Code != http.StatusOK || res.Body.String() != "{\"authorized\":true}\n" {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
	if res.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache control = %q", res.Header().Get("Cache-Control"))
	}
	if strings.Contains(res.Body.String(), "email") || strings.Contains(res.Body.String(), "uid") {
		t.Fatalf("authorization response exposed identity data: %s", res.Body.String())
	}
}

func TestAdminAuthorizeRejectsUnauthorisedRequests(t *testing.T) {
	original := adminGateRequireAdmin
	t.Cleanup(func() { adminGateRequireAdmin = original })
	adminGateRequireAdmin = func(context.Context, *http.Request) (*firebaseUser, error) {
		return nil, authorizationFailure(http.StatusForbidden, "Admin role required", errors.New("admin role required"))
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/authorize", nil)
	res := httptest.NewRecorder()
	AdminAuthorize(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}
