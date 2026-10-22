package ipace

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

var memberFeatureNames = []string{"member-experiences"}
var memberFeaturesRequireUser = requireUser

func featureEnabledForUser(name string, user *firebaseUser) bool {
	raw := strings.TrimSpace(os.Getenv("MEMBER_FEATURE_ACCESS"))
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
		case "all":
			return true
		case "off", "":
			continue
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

func MemberFeatures(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	user, err := memberFeaturesRequireUser(r.Context(), r)
	if err != nil {
		writeMemberAuthorizationError(w, err)
		return
	}
	features := map[string]bool{}
	for _, name := range memberFeatureNames {
		features[name] = featureEnabledForUser(name, user)
	}
	writeJSON(w, http.StatusOK, map[string]any{"features": features})
}
