package ipace

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

const representationWording = "I authorise the I-PACE Owners' Advocacy Group to represent my interests in preparatory legal engagement with JLR. This does not authorise issuing proceedings or accepting costs on my behalf."

type memberContactProfile struct {
	Name                    string    `json:"name" firestore:"name"`
	Phone                   string    `json:"phone" firestore:"phone"`
	AddressLine1            string    `json:"addressLine1" firestore:"addressLine1"`
	AddressLine2            string    `json:"addressLine2" firestore:"addressLine2"`
	City                    string    `json:"city" firestore:"city"`
	Region                  string    `json:"region" firestore:"region"`
	PostalCode              string    `json:"postalCode" firestore:"postalCode"`
	Country                 string    `json:"country" firestore:"country"`
	Represent               bool      `json:"represent" firestore:"represent"`
	RepresentationVersion   string    `json:"representationVersion" firestore:"representationVersion"`
	RepresentationUpdatedAt time.Time `json:"representationUpdatedAt" firestore:"representationUpdatedAt"`
	UpdatedAt               time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// Keep mandate changes separately from the single contact profile. The event
// references the member by its parent document and deliberately contains no PII.
type representationChoiceEvent struct {
	Represent bool      `firestore:"represent"`
	Version   string    `firestore:"version"`
	Wording   string    `firestore:"wording"`
	At        time.Time `firestore:"at"`
}

var ukPostcode = regexp.MustCompile(`(?i)^(GIR\s*0AA|(?:[A-Z]{1,2}[0-9][A-Z0-9]?\s*[0-9][A-Z]{2}))$`)

func isUKCountry(country string) bool {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "GB", "UK", "UNITED KINGDOM":
		return true
	}
	return false
}

func validateMemberContactProfile(p *memberContactProfile) string {
	p.Name = cleanString(p.Name, 160)
	p.Phone = cleanString(p.Phone, 40)
	p.AddressLine1 = cleanString(p.AddressLine1, 160)
	p.AddressLine2 = cleanString(p.AddressLine2, 160)
	p.City = cleanString(p.City, 100)
	p.Region = cleanString(p.Region, 100)
	p.PostalCode = strings.ToUpper(cleanString(p.PostalCode, 24))
	p.Country = cleanString(p.Country, 100)
	if p.Country == "" {
		return "Choose a country"
	}
	if p.Represent {
		if !isUKCountry(p.Country) {
			return "Representation is currently available only to members with a UK address"
		}
		if p.Name == "" || p.AddressLine1 == "" || p.City == "" || !ukPostcode.MatchString(p.PostalCode) {
			return "A name and complete UK postal address are required for representation"
		}
	}
	return ""
}

func representationChoiceChanged(previous, next memberContactProfile) bool {
	if previous.Represent != next.Represent {
		return true
	}
	if !next.Represent {
		return false
	}
	return previous.RepresentationVersion != next.RepresentationVersion ||
		previous.Name != next.Name || previous.AddressLine1 != next.AddressLine1 ||
		previous.AddressLine2 != next.AddressLine2 || previous.City != next.City ||
		previous.Region != next.Region || previous.PostalCode != next.PostalCode ||
		previous.Country != next.Country
}

// MemberProfile stores one contact profile per authenticated person. Verification
// status is deliberately kept in a separate admin-only collection.
func MemberProfile(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	user, err := requireUser(r.Context(), r)
	if err != nil {
		writeMemberAuthorizationError(w, err)
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not load profile"})
		return
	}
	ref := db.Collection("memberProfiles").Doc(user.UID)
	if r.Method == http.MethodGet {
		var profile memberContactProfile
		if doc, err := ref.Get(r.Context()); err == nil {
			_ = doc.DataTo(&profile)
		}
		writeJSON(w, 200, map[string]any{"profile": profile, "representationWording": representationWording})
		return
	}
	var profile memberContactProfile
	if err := decodeJSON(r, &profile); err != nil {
		writeJSON(w, 400, map[string]any{"error": "Invalid profile"})
		return
	}
	if msg := validateMemberContactProfile(&profile); msg != "" {
		writeJSON(w, 400, map[string]any{"error": msg})
		return
	}
	var previous memberContactProfile
	if doc, err := ref.Get(r.Context()); err == nil {
		_ = doc.DataTo(&previous)
	}
	now := time.Now().UTC()
	if profile.Represent {
		profile.RepresentationVersion = "2026-10-02-v1"
	} else {
		profile.RepresentationVersion = ""
	}
	choiceChanged := representationChoiceChanged(previous, profile)
	if choiceChanged {
		profile.RepresentationUpdatedAt = now
	} else {
		profile.RepresentationUpdatedAt = previous.RepresentationUpdatedAt
	}
	profile.UpdatedAt = now
	batch := db.Batch()
	batch.Set(ref, profile)
	if choiceChanged {
		batch.Set(ref.Collection("representationChoices").NewDoc(), representationChoiceEvent{
			Represent: profile.Represent,
			Version:   profile.RepresentationVersion,
			Wording:   representationWording,
			At:        now,
		})
	}
	if profile.Name != previous.Name || profile.AddressLine1 != previous.AddressLine1 || profile.AddressLine2 != previous.AddressLine2 || profile.City != previous.City || profile.Region != previous.Region || profile.PostalCode != previous.PostalCode || profile.Country != previous.Country {
		batch.Set(db.Collection("memberVerification").Doc(user.UID), map[string]any{"verified": false, "updatedAt": now, "reason": "profile-changed"}, firestore.MergeAll)
	}
	if _, err := batch.Commit(r.Context()); err != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not save profile"})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// MemberAddressLookup searches an address or postcode and resolves a selected address.
// The provider key remains on the server; manual entry is always available.
func MemberAddressLookup(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if _, err := requireUser(r.Context(), r); err != nil {
		writeMemberAuthorizationError(w, err)
		return
	}
	key := os.Getenv("IDEAL_POSTCODES_API_KEY")
	if key == "" {
		writeJSON(w, 503, map[string]any{"error": "Address lookup is unavailable. Enter your address manually."})
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	var endpoint string
	if id != "" {
		if len(id) > 100 || !addressID.MatchString(id) {
			writeJSON(w, 400, map[string]any{"error": "Invalid address selection"})
			return
		}
		endpoint = "https://api.ideal-postcodes.co.uk/v1/autocomplete/addresses/" + url.PathEscape(id) + "/gbr?api_key=" + url.QueryEscape(key)
	} else {
		if len(query) < 3 || len(query) > 160 {
			writeJSON(w, 400, map[string]any{"error": "Enter at least three characters of an address or postcode"})
			return
		}
		context := addressSearchContext(r.URL.Query().Get("country"))
		endpoint = "https://api.ideal-postcodes.co.uk/v1/autocomplete/addresses?api_key=" + url.QueryEscape(key) + "&query=" + url.QueryEscape(query) + "&context=" + context
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": "Address lookup failed. Enter your address manually."})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		writeJSON(w, 502, map[string]any{"error": "No addresses found. Enter your address manually."})
		return
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&envelope) != nil {
		writeJSON(w, 502, map[string]any{"error": "Address lookup failed"})
		return
	}
	if id != "" {
		var address struct {
			Line1    string `json:"line_1"`
			Line2    string `json:"line_2"`
			PostTown string `json:"post_town"`
			County   string `json:"county"`
			Postcode string `json:"postcode"`
			Country  string `json:"country"`
		}
		if json.Unmarshal(envelope.Result, &address) != nil {
			writeJSON(w, 502, map[string]any{"error": "Address lookup failed"})
			return
		}
		writeJSON(w, 200, map[string]any{"address": address})
		return
	}
	var suggestions struct {
		Hits []struct {
			ID         string `json:"id"`
			Suggestion string `json:"suggestion"`
		} `json:"hits"`
	}
	if json.Unmarshal(envelope.Result, &suggestions) != nil {
		writeJSON(w, 502, map[string]any{"error": "Address lookup failed"})
		return
	}
	writeJSON(w, 200, map[string]any{"suggestions": suggestions.Hits})
}

var addressID = regexp.MustCompile(`^[a-zA-Z0-9_-][a-zA-Z0-9_|:-]*$`)

func addressSearchContext(country string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "UNITED STATES", "US", "USA":
		return "USA"
	case "CANADA", "CA", "CAN":
		return "CAN"
	case "IRELAND", "IE", "IRL":
		return "IRL"
	case "FRANCE", "FR", "FRA":
		return "FRA"
	case "GERMANY", "DE", "DEU":
		return "DEU"
	case "AUSTRALIA", "AU", "AUS":
		return "AUS"
	case "NEW ZEALAND", "NZ", "NZL":
		return "NZL"
	default:
		return "GBR"
	}
}

// Admin-only verification status. Members cannot set or read this record.
func AdminMemberVerification(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return
	}
	admin, err := requireAdmin(r.Context(), r)
	if err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	var req struct {
		UID      string `json:"uid"`
		Verified bool   `json:"verified"`
	}
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.UID) == "" {
		writeJSON(w, 400, map[string]any{"error": "Member ID required"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not update verification"})
		return
	}
	if _, err := db.Collection("memberProfiles").Doc(req.UID).Get(r.Context()); err != nil {
		writeJSON(w, 404, map[string]any{"error": "Member profile not found"})
		return
	}
	_, err = db.Collection("memberVerification").Doc(req.UID).Set(r.Context(), map[string]any{"verified": req.Verified, "updatedAt": time.Now().UTC(), "adminUID": admin.UID}, firestore.MergeAll)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not update verification"})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
