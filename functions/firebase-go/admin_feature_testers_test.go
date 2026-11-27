package ipace

import (
	"context"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type fakeFeatureTesterClient struct{ email string }

func (f *fakeFeatureTesterClient) GetUser(_ context.Context, uid string) (*firebaseauth.UserRecord, error) {
	return &firebaseauth.UserRecord{UserInfo: &firebaseauth.UserInfo{UID: uid}}, nil
}
func (f *fakeFeatureTesterClient) GetUserByEmail(_ context.Context, email string) (*firebaseauth.UserRecord, error) {
	f.email = email
	return &firebaseauth.UserRecord{UserInfo: &firebaseauth.UserInfo{UID: "resolved-uid"}}, nil
}
func (f *fakeFeatureTesterClient) SetCustomUserClaims(context.Context, string, map[string]any) error {
	return nil
}

func TestFeatureTesterWithoutJoinUIDResolvesByEmail(t *testing.T) {
	client := &fakeFeatureTesterClient{}
	member, err := resolveFeatureTester(context.Background(), client, featureTesterApprovalRequest{Email: "member@example.org"})
	if err != nil || client.email != "member@example.org" || member.UID != "resolved-uid" {
		t.Fatalf("email lookup failed: %#v %v", member, err)
	}
}

func TestFeatureTesterClaimsPreserveOtherRoles(t *testing.T) {
	claims := featureTesterClaims(map[string]any{"admin": true, "roles": []any{"editor", memberExperiencesPreviewRole}}, true)
	if claims["admin"] != true || !userHasRole(&firebaseUser{Claims: claims}, "editor") || !userHasRole(&firebaseUser{Claims: claims}, memberExperiencesPreviewRole) {
		t.Fatalf("approval lost an existing claim or role: %#v", claims)
	}
	revoked := featureTesterClaims(claims, false)
	if userHasRole(&firebaseUser{Claims: revoked}, memberExperiencesPreviewRole) || !userHasRole(&firebaseUser{Claims: revoked}, "editor") || revoked["admin"] != true {
		t.Fatalf("revocation changed unrelated access: %#v", revoked)
	}
}
