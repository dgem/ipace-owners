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
	preview := &firebaseUser{UID: "preview-1", Claims: map[string]any{"roles": []any{"early-adopter"}}}

	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"early-adopter"}`)
	t.Setenv("MEMBER_FEATURE_ACCESS", "")
	if featureEnabledForUser("member-experiences", member) {
		t.Fatal("ordinary member received a preview-only feature")
	}
	if !featureEnabledForUser("member-experiences", admin) || !featureEnabledForUser("member-experiences", preview) {
		t.Fatal("admin or named preview role could not access the feature")
	}

	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"all"}`)
	if !featureEnabledForUser("member-experiences", member) {
		t.Fatal("released feature was not available to a member")
	}

	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"off"}`)
	if !featureEnabledForUser("member-experiences", admin) {
		t.Fatal("admin did not bypass the feature rule")
	}
	if featureEnabledForUser("member-experiences", member) {
		t.Fatal("disabled feature was available to an ordinary member")
	}

	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"public"}`)
	if !featureEnabledForUser("member-experiences", nil) {
		t.Fatal("public feature was unavailable to an anonymous visitor")
	}
	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"all"}`)
	if featureEnabledForUser("member-experiences", nil) {
		t.Fatal("signed-in audience rule was available anonymously")
	}
}

func TestMalformedMemberFeatureConfigFailsClosed(t *testing.T) {
	t.Setenv("FEATURE_ACCESS", `{broken`)
	if featureEnabledForUser("member-experiences", &firebaseUser{UID: "member", Claims: map[string]any{}}) {
		t.Fatal("malformed feature configuration enabled the feature")
	}
	if !featureEnabledForUser("member-experiences", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}) {
		t.Fatal("admin did not bypass malformed feature configuration")
	}
}

func TestMissingMemberFeatureConfigFailsClosed(t *testing.T) {
	t.Setenv("FEATURE_ACCESS", "")
	t.Setenv("MEMBER_FEATURE_ACCESS", "")
	if featureEnabledForUser("member-experiences", &firebaseUser{UID: "member", Claims: map[string]any{}}) {
		t.Fatal("missing feature configuration enabled the feature")
	}
	if !featureEnabledForUser("member-experiences", &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}) {
		t.Fatal("admin did not bypass missing feature configuration")
	}
}

func TestMemberFeaturesReturnsOnlyResolvedBooleans(t *testing.T) {
	original := memberFeaturesOptionalUser
	t.Cleanup(func() { memberFeaturesOptionalUser = original })
	memberFeaturesOptionalUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "preview", Claims: map[string]any{"roles": []any{"early-adopter"}}}, nil
	}
	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"early-adopter","merchandise":"early-adopter"}`)
	request := httptest.NewRequest(http.MethodGet, "/api/member/features", nil)
	response := httptest.NewRecorder()
	MemberFeatures(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"member-experiences":true`) || !strings.Contains(response.Body.String(), `"merchandise":true`) {
		t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestMemberFeaturesCanResolvePublicAudienceWithoutAuthentication(t *testing.T) {
	original := memberFeaturesOptionalUser
	t.Cleanup(func() { memberFeaturesOptionalUser = original })
	memberFeaturesOptionalUser = func(context.Context, *http.Request) (*firebaseUser, error) { return nil, nil }
	t.Setenv("FEATURE_ACCESS", `{"public-page":"public","member-experiences":"all"}`)
	request := httptest.NewRequest(http.MethodGet, "/api/member/features", nil)
	response := httptest.NewRecorder()
	MemberFeatures(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"public-page":true`) || !strings.Contains(response.Body.String(), `"member-experiences":false`) {
		t.Fatalf("unexpected anonymous response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestMerchandiseAudienceTokenDoesNotConsumeReceiptBearer(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/shop/checkout", nil)
	request.Header.Set("Authorization", "Bearer opaque-order-receipt")
	user, err := optionalFeatureUser(request.Context(), request)
	if err != nil || user != nil {
		t.Fatalf("receipt bearer was treated as a Firebase token: user=%v err=%v", user, err)
	}
}

func TestFeatureAudienceAllowsEarlyAdoptersAndAdmins(t *testing.T) {
	original := featureAudienceOptionalUser
	t.Cleanup(func() { featureAudienceOptionalUser = original })
	t.Setenv("FEATURE_ACCESS", `{"member-experiences":"early-adopter","merchandise":"off"}`)
	for _, user := range []*firebaseUser{
		{UID: "preview", Claims: map[string]any{"roles": []any{"early-adopter"}}},
		{UID: "admin", Claims: map[string]any{"admin": true}},
	} {
		featureAudienceOptionalUser = func(context.Context, *http.Request) (*firebaseUser, error) { return user, nil }
		response := httptest.NewRecorder()
		if !requireFeatureAudience(response, httptest.NewRequest(http.MethodGet, "/shop/", nil), "member-experiences") {
			t.Fatalf("feature gate denied %s: %s", user.UID, response.Body)
		}
	}
	featureAudienceOptionalUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "admin", Claims: map[string]any{"admin": true}}, nil
	}
	response := httptest.NewRecorder()
	if !requireFeatureAudience(response, httptest.NewRequest(http.MethodGet, "/shop/", nil), "merchandise") {
		t.Fatalf("admin did not bypass disabled rule: %s", response.Body)
	}
}
