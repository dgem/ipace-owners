package ipace

import "testing"

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
