package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemberDataUsesPrivateNoStoreResponse(t *testing.T) {
	originalRequireUser := memberDataRequireUser
	originalLoadSnapshot := memberDataLoadSnapshot
	t.Cleanup(func() {
		memberDataRequireUser = originalRequireUser
		memberDataLoadSnapshot = originalLoadSnapshot
	})
	memberDataRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "member-uid", Email: "member@example.test"}, nil
	}
	memberDataLoadSnapshot = func(context.Context, string, string) (memberSnapshot, error) {
		return memberSnapshot{Email: "member@example.test"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/member-data", nil)
	res := httptest.NewRecorder()
	MemberData(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
	if res.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache control = %q", res.Header().Get("Cache-Control"))
	}
}
