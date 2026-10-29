package ipace

import (
	"context"
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
