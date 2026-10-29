package ipace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMemberNotificationsListsAndDismissesVersionedNotices(t *testing.T) {
	originalRequire := memberNotificationsRequireUser
	originalLoad := memberNotificationsLoad
	originalDismiss := memberNotificationsDismiss
	t.Cleanup(func() {
		memberNotificationsRequireUser = originalRequire
		memberNotificationsLoad = originalLoad
		memberNotificationsDismiss = originalDismiss
	})
	memberNotificationsRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "member-1"}, nil
	}
	memberNotificationsLoad = func(context.Context, string) ([]string, error) {
		return []string{"survey:open:survey_1"}, nil
	}
	var storedUID, storedID string
	memberNotificationsDismiss = func(_ context.Context, uid, id string) error {
		storedUID, storedID = uid, id
		return nil
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/member/notifications", nil)
	getResponse := httptest.NewRecorder()
	MemberNotifications(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
	var listed struct {
		Dismissed []string `json:"dismissed"`
	}
	if err := json.Unmarshal(getResponse.Body.Bytes(), &listed); err != nil || !reflect.DeepEqual(listed.Dismissed, []string{"survey:open:survey_1"}) {
		t.Fatalf("unexpected GET response: %#v, err=%v", listed, err)
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/api/member/notifications", bytes.NewBufferString(`{"id":"cta:member-experiences:v1"}`))
	postRequest.Header.Set("Origin", "http://localhost:8080")
	postRequest.Header.Set("Content-Type", "application/json")
	postResponse := httptest.NewRecorder()
	MemberNotifications(postResponse, postRequest)
	if postResponse.Code != http.StatusOK || storedUID != "member-1" || storedID != "cta:member-experiences:v1" {
		t.Fatalf("dismissal was not stored: status=%d uid=%q id=%q body=%s", postResponse.Code, storedUID, storedID, postResponse.Body.String())
	}
}

func TestMemberNotificationsRejectsInvalidIDs(t *testing.T) {
	originalRequire := memberNotificationsRequireUser
	t.Cleanup(func() { memberNotificationsRequireUser = originalRequire })
	memberNotificationsRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "member-1"}, nil
	}
	request := httptest.NewRequest(http.MethodPost, "/api/member/notifications", bytes.NewBufferString(`{"id":"Unsafe notice!"}`))
	request.Header.Set("Origin", "http://localhost:8080")
	response := httptest.NewRecorder()
	MemberNotifications(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestMemberNotificationsRequiresAuthentication(t *testing.T) {
	originalRequire := memberNotificationsRequireUser
	t.Cleanup(func() { memberNotificationsRequireUser = originalRequire })
	memberNotificationsRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return nil, authorizationFailure(http.StatusUnauthorized, "Sign in required", nil)
	}
	response := httptest.NewRecorder()
	MemberNotifications(response, httptest.NewRequest(http.MethodGet, "/api/member/notifications", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}
