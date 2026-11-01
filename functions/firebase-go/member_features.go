package ipace

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

var memberFeaturesOptionalUser = optionalUser
var featureAudienceOptionalUser = optionalFeatureUser

func optionalFeatureUser(ctx context.Context, r *http.Request) (*firebaseUser, error) {
	token := strings.TrimSpace(r.Header.Get("X-Firebase-ID-Token"))
	if token == "" {
		return nil, nil
	}
	request := r.Clone(ctx)
	request.Header = r.Header.Clone()
	request.Header.Set("Authorization", "Bearer "+token)
	return optionalUser(ctx, request)
}

func featureEnabledForUser(name string, user *firebaseUser) bool {
	if isAdmin(user) {
		return true
	}
	raw := strings.TrimSpace(os.Getenv("FEATURE_ACCESS"))
	if raw == "" {
		// Keep compatibility with already deployed environments during rollout.
		raw = strings.TrimSpace(os.Getenv("MEMBER_FEATURE_ACCESS"))
	}
	if raw == "" {
		return false
	}
	access := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &access); err != nil {
		return false
	}
	rule, exists := access[name]
	if !exists {
		return false
	}
	for _, value := range strings.Split(strings.ToLower(rule), ",") {
		role := strings.TrimSpace(value)
		switch role {
		case "public", "anonymous", "unauthenticated":
			return true
		case "all":
			if user != nil && user.UID != "" {
				return true
			}
		case "off", "":
			continue
		case "early-adopter":
			if userHasRole(user, role) || (name == "member-experiences" && userHasRole(user, "member-experiences-preview")) {
				return true
			}
		default:
			if userHasRole(user, role) {
				return true
			}
		}
	}
	return false
}

func requireMemberFeature(w http.ResponseWriter, user *firebaseUser, name string) bool {
	if featureEnabledForUser(name, user) {
		return true
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": "Feature unavailable"})
	return false
}

func requireFeatureAudience(w http.ResponseWriter, r *http.Request, name string) bool {
	user, err := featureAudienceOptionalUser(r.Context(), r)
	if err != nil {
		writeMemberAuthorizationError(w, err)
		return false
	}
	if featureEnabledForUser(name, user) {
		return true
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": "Feature unavailable"})
	return false
}

func MemberFeatures(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	user, err := memberFeaturesOptionalUser(r.Context(), r)
	if err != nil {
		writeMemberAuthorizationError(w, err)
		return
	}
	access := map[string]string{}
	raw := strings.TrimSpace(os.Getenv("FEATURE_ACCESS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("MEMBER_FEATURE_ACCESS"))
	}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &access); err != nil {
			access = map[string]string{}
		}
	}
	features := make(map[string]bool, len(access)+2)
	// Admins bypass feature rules, so the browser must learn about known gated
	// features even when an older environment has not configured them yet.
	for _, name := range []string{"member-experiences", "merchandise"} {
		if _, exists := access[name]; !exists {
			access[name] = ""
		}
	}
	for name := range access {
		features[name] = featureEnabledForUser(name, user)
	}
	writeJSON(w, http.StatusOK, map[string]any{"features": features})
}
