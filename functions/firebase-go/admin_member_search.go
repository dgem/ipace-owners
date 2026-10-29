package ipace

import (
	"net/http"
	"sort"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type adminMemberSearchResult struct {
	IdentityUserID string   `json:"identityUserId"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"displayName"`
	Email          string   `json:"email"`
	Country        string   `json:"country"`
	Registrations  []string `json:"registrations"`
	Expertise      string   `json:"expertise"`
	WillingToHelp  bool     `json:"willingToHelp"`
	PressStory     bool     `json:"pressStory"`
	StoryCount     int      `json:"storyCount"`
	Matched        []string `json:"matched"`
}

func AdminMemberSearch(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if _, err := requireAdmin(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	query := strings.ToLower(cleanString(r.URL.Query().Get("q"), 160))
	if len([]rune(query)) < 2 {
		writeJSON(w, 400, map[string]any{"error": "Enter at least two characters"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not search members"})
		return
	}
	joins := []joinRecord{}
	vehicles := []vehicleRecord{}
	stories := []memberStory{}
	if readCollection(r.Context(), db.Collection("joinSubmissions").OrderBy("createdAt", firestore.Desc), &joins) != nil || readCollection(r.Context(), db.Collection("vehicles").OrderBy("createdAt", firestore.Desc), &vehicles) != nil || readCollection(r.Context(), db.Collection("memberStories").OrderBy("updatedAt", firestore.Desc), &stories) != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not search members"})
		return
	}
	results := map[string]*adminMemberSearchResult{}
	for _, join := range joins {
		key := join.IdentityUserID
		if key == "" {
			key = join.UserEmailHash
		}
		if key == "" {
			continue
		}
		if results[key] == nil {
			results[key] = &adminMemberSearchResult{IdentityUserID: join.IdentityUserID, Name: join.Contact.Name, DisplayName: join.Membership.DisplayName, Email: join.Contact.Email, Country: join.Contact.Country, Expertise: join.Membership.Expertise, WillingToHelp: join.Membership.WillingToHelp, PressStory: join.Membership.PressStory, Registrations: []string{}, Matched: []string{}}
		}
	}
	profileIter := db.Collection("memberProfiles").Documents(r.Context())
	for {
		doc, profileErr := profileIter.Next()
		if profileErr == iterator.Done {
			break
		}
		if profileErr != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not search members"})
			return
		}
		var profile memberContactProfile
		if doc.DataTo(&profile) != nil {
			continue
		}
		if result := results[doc.Ref.ID]; result != nil {
			if profile.Name != "" {
				result.Name = profile.Name
			}
			if profile.Country != "" {
				result.Country = profile.Country
			}
		}
	}
	for _, vehicle := range vehicles {
		if result := results[vehicle.IdentityUserID]; result != nil && vehicle.Vehicle.Registration != "" {
			result.Registrations = appendUnique(result.Registrations, vehicle.Vehicle.Registration)
		}
	}
	for _, story := range stories {
		if result := results[story.IdentityUserID]; result != nil {
			result.StoryCount++
		}
	}
	matched := []adminMemberSearchResult{}
	for _, result := range results {
		fields := map[string]string{"name": result.Name, "display name": result.DisplayName, "email": result.Email, "country": result.Country, "registration": strings.Join(result.Registrations, " "), "expertise": result.Expertise}
		if result.WillingToHelp {
			fields["preference"] = "willing to help volunteer"
		}
		if result.PressStory {
			fields["press"] = "press story media"
		}
		for label, value := range fields {
			if strings.Contains(strings.ToLower(value), query) {
				result.Matched = appendUnique(result.Matched, label)
			}
		}
		if len(result.Matched) > 0 {
			matched = append(matched, *result)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return strings.ToLower(matched[i].Name+matched[i].Email) < strings.ToLower(matched[j].Name+matched[j].Email)
	})
	if len(matched) > 100 {
		matched = matched[:100]
	}
	writeJSON(w, 200, map[string]any{"query": query, "results": matched})
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
