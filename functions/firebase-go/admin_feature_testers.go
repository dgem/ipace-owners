package ipace

import (
	"context"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
)

const memberExperiencesPreviewRole = "member-experiences-preview"

type featureTesterApprovalRequest struct {
	UID      string `json:"uid"`
	Email    string `json:"email"`
	Approved bool   `json:"approved"`
}

type featureTesterClaimsClient interface {
	GetUser(context.Context, string) (*firebaseauth.UserRecord, error)
	GetUserByEmail(context.Context, string) (*firebaseauth.UserRecord, error)
	SetCustomUserClaims(context.Context, string, map[string]any) error
}

var featureTesterClaimsClientFor = func(ctx context.Context) (featureTesterClaimsClient, error) {
	return firebaseAuth(ctx)
}

// featureTesterClaims preserves existing claims (especially admin) while adding or
// removing only the preview role controlled by this queue.
func featureTesterClaims(existing map[string]any, approved bool) map[string]any {
	claims := map[string]any{}
	for key, value := range existing {
		claims[key] = value
	}
	roles := []string{}
	switch values := existing["roles"].(type) {
	case []any:
		for _, value := range values {
			if role, ok := value.(string); ok && strings.TrimSpace(role) != "" && role != memberExperiencesPreviewRole {
				roles = appendUnique(roles, role)
			}
		}
	case []string:
		for _, role := range values {
			if strings.TrimSpace(role) != "" && role != memberExperiencesPreviewRole {
				roles = appendUnique(roles, role)
			}
		}
	}
	if approved {
		roles = append(roles, memberExperiencesPreviewRole)
	}
	if len(roles) == 0 {
		delete(claims, "roles")
	} else {
		claims["roles"] = roles
	}
	return claims
}

func AdminFeatureTesterApproval(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if _, err := requireAdmin(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	var input featureTesterApprovalRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request"})
		return
	}
	input.UID = cleanString(input.UID, 128)
	input.Email = cleanString(input.Email, 254)
	if input.UID == "" && input.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Member identity is required"})
		return
	}
	client, err := featureTesterClaimsClientFor(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Could not update feature access"})
		return
	}
	member, err := resolveFeatureTester(r.Context(), client, input)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Member account was not found"})
		return
	}
	input.UID = member.UID
	if err := client.SetCustomUserClaims(r.Context(), input.UID, featureTesterClaims(member.CustomClaims, input.Approved)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not save feature access"})
		return
	}
	message := "Feature access revoked"
	if input.Approved {
		message = "Feature access approved. The member must refresh their sign-in token before access changes."
	}
	writeJSON(w, http.StatusOK, map[string]any{"uid": input.UID, "approved": input.Approved, "message": message})
}

func resolveFeatureTester(ctx context.Context, client featureTesterClaimsClient, input featureTesterApprovalRequest) (*firebaseauth.UserRecord, error) {
	if input.UID != "" {
		return client.GetUser(ctx, input.UID)
	}
	return client.GetUserByEmail(ctx, input.Email)
}
