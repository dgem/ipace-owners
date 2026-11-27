package ipace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemberFeatureAccessSupportsReleaseAndRoleRules(t *testing.T) {
	member := &firebaseUser{UID: "member-1", Claims: map[string]any{}}
	admin := &firebaseUser{UID: "admin-1", Claims: map[string]any{"admin": true}}
	preview := &firebaseUser{UID: "preview-1", Claims: map[string]any{"roles": []any{"member-experiences-preview"}}}

	t.Setenv("MEMBER_FEATURE_ACCESS", `{"member-experiences":"admin,member-experiences-preview"}`)
	if featureEnabledForUser("member-experiences", member) {
		t.Fatal("ordinary member received a preview-only feature")
	}
	if !featureEnabledForUser("member-experiences", admin) || !featureEnabledForUser("member-experiences", preview) {
		t.Fatal("admin or named preview role could not access the feature")
	}

	t.Setenv("MEMBER_FEATURE_ACCESS", `{"member-experiences":"all"}`)
	if !featureEnabledForUser("member-experiences", member) {
		t.Fatal("released feature was not available to a member")
	}

	t.Setenv("MEMBER_FEATURE_ACCESS", `{"member-experiences":"off"}`)
	if featureEnabledForUser("member-experiences", admin) {
		t.Fatal("disabled feature was available to an admin")
	}
}

func TestMalformedMemberFeatureConfigFailsClosed(t *testing.T) {
	t.Setenv("MEMBER_FEATURE_ACCESS", `{broken`)
	if featureEnabledForUser("member-experiences", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}) {
		t.Fatal("malformed feature configuration enabled the feature")
	}
}

func TestMissingMemberFeatureConfigFailsClosed(t *testing.T) {
	t.Setenv("MEMBER_FEATURE_ACCESS", "")
	if featureEnabledForUser("member-experiences", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}) {
		t.Fatal("missing feature configuration enabled the feature")
	}
}

func TestMemberFeaturesReturnsOnlyResolvedBooleans(t *testing.T) {
	original := memberFeaturesRequireUser
	t.Cleanup(func() { memberFeaturesRequireUser = original })
	memberFeaturesRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "preview", Claims: map[string]any{"roles": []any{"member-experiences-preview"}}}, nil
	}
	t.Setenv("MEMBER_FEATURE_ACCESS", `{"member-experiences":"member-experiences-preview"}`)
	request := httptest.NewRequest(http.MethodGet, "/api/member/features", nil)
	response := httptest.NewRecorder()
	MemberFeatures(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"member-experiences":true`) {
		t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestOwnersStoriesRejectsAnonymousAndUnapprovedMembers(t *testing.T) {
	original := memberFeaturesRequireUser
	t.Cleanup(func() { memberFeaturesRequireUser = original })
	t.Setenv("MEMBER_FEATURE_ACCESS", `{"owners-stories":"admin,early-adopter,member-experiences-preview"}`)
	memberFeaturesRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) { return nil, errors.New("unauthenticated") }
	anonymous := httptest.NewRecorder()
	PublicStories(anonymous, httptest.NewRequest(http.MethodGet, "/api/stories", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous response: %d", anonymous.Code)
	}
	for _, tc := range []struct {
		name    string
		user    *firebaseUser
		allowed bool
	}{
		{"ordinary", &firebaseUser{UID: "member", Claims: map[string]any{}}, false},
		{"admin", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}, true},
		{"early adopter", &firebaseUser{UID: "early", Claims: map[string]any{"roles": []any{"early-adopter"}}}, true},
		{"preview tester", &firebaseUser{UID: "tester", Claims: map[string]any{"roles": []any{"member-experiences-preview"}}}, true},
	} {
		if got := featureEnabledForUser("owners-stories", tc.user); got != tc.allowed {
			t.Errorf("%s: got %v", tc.name, got)
		}
	}
	memberFeaturesRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "member", Claims: map[string]any{}}, nil
	}
	w := httptest.NewRecorder()
	PublicStories(w, httptest.NewRequest(http.MethodGet, "/api/stories", nil))
	if w.Code != http.StatusNotFound || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("unapproved response: %d %s", w.Code, w.Body.String())
	}
}
