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

	t.Setenv("MEMBER_FEATURE_ACCESS", `{"owners-stories":"admin,member-experiences-preview"}`)
	if featureEnabledForUser("owners-stories", member) {
		t.Fatal("ordinary member received a preview-only feature")
	}
	if !featureEnabledForUser("owners-stories", admin) || !featureEnabledForUser("owners-stories", preview) {
		t.Fatal("admin or named preview role could not access the feature")
	}

	t.Setenv("MEMBER_FEATURE_ACCESS", `{"owners-stories":"all"}`)
	if !featureEnabledForUser("owners-stories", member) {
		t.Fatal("released feature was not available to a member")
	}

	t.Setenv("MEMBER_FEATURE_ACCESS", `{"owners-stories":"off"}`)
	if featureEnabledForUser("owners-stories", admin) {
		t.Fatal("disabled feature was available to an admin")
	}
}

func TestMalformedMemberFeatureConfigFailsClosed(t *testing.T) {
	t.Setenv("MEMBER_FEATURE_ACCESS", `{broken`)
	if featureEnabledForUser("owners-stories", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}) {
		t.Fatal("malformed feature configuration enabled the feature")
	}
}

func TestMissingMemberFeatureConfigFailsClosed(t *testing.T) {
	t.Setenv("MEMBER_FEATURE_ACCESS", "")
	if featureEnabledForUser("owners-stories", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}) {
		t.Fatal("missing feature configuration enabled the feature")
	}
}

func TestMemberFeaturesReturnsOnlyResolvedBooleans(t *testing.T) {
	original := memberFeaturesRequireUser
	t.Cleanup(func() { memberFeaturesRequireUser = original })
	memberFeaturesRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "preview", Claims: map[string]any{"roles": []any{"member-experiences-preview"}}}, nil
	}
	t.Setenv("MEMBER_FEATURE_ACCESS", `{"owners-stories":"member-experiences-preview"}`)
	request := httptest.NewRequest(http.MethodGet, "/api/member/features", nil)
	response := httptest.NewRecorder()
	MemberFeatures(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"owners-stories":true`) {
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

func TestStoryVisibilityRulesNeverPermitAnonymousAuthoring(t *testing.T) {
	original := memberFeaturesRequireUser
	t.Cleanup(func() { memberFeaturesRequireUser = original })
	memberFeaturesRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return nil, errors.New("unauthenticated")
	}
	for _, rule := range []string{"all", "public"} {
		t.Run(rule, func(t *testing.T) {
			t.Setenv("MEMBER_FEATURE_ACCESS", `{"owners-stories":"`+rule+`"}`)
			if got := featureEnabledForUser("owners-stories", nil); got != (rule == "public") {
				t.Fatalf("anonymous visibility for %s = %v", rule, got)
			}
			if !featureEnabledForUser("owners-stories", &firebaseUser{UID: "member"}) {
				t.Fatalf("signed-in member denied for %s", rule)
			}
			features := httptest.NewRecorder()
			MemberFeatures(features, httptest.NewRequest(http.MethodGet, "/api/member/features", nil))
			if rule == "public" && (features.Code != http.StatusOK || !strings.Contains(features.Body.String(), `"owners-stories":true`)) {
				t.Fatalf("public visibility response: %d %s", features.Code, features.Body.String())
			}
			if rule == "all" && features.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous members-only response: %d", features.Code)
			}
			for _, handler := range []http.HandlerFunc{MemberStories, MemberStoryUpload, MemberStoryUploadComplete, MemberStoryWithdraw} {
				w := httptest.NewRecorder()
				handler(w, httptest.NewRequest(http.MethodPost, "/api/member/stories", strings.NewReader(`{}`)))
				if w.Code != http.StatusUnauthorized {
					t.Fatalf("anonymous authoring with %s returned %d", rule, w.Code)
				}
			}
		})
	}
}
