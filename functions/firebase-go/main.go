package ipace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/iterator"
)

var relationshipValues = []string{
	"current-owner-one",
	"current-owner-multiple",
	"former-owner",
	"prospective-buyer",
	"helping-owner",
	"trade-specialist",
	"other",
}

var skillValues = []string{
	"legal",
	"technical",
	"data",
	"media",
	"web",
	"consumer-rights",
	"dealer",
	"general",
}

var countryValues = []string{
	"GB", "IE", "DE", "FR", "NL", "NO", "SE", "DK", "AT", "CH", "BE", "ES", "IT", "PT", "US", "CA", "AU", "NZ", "other",
}

var modelYearValues = []string{"2018", "2019", "2020", "2021", "2022", "2023", "2024"}
var sohSourceValues = []string{"dealer-report", "diagnostic-app", "service-paperwork", "jlr-communication", "estimate-unsure"}

func init() {
	functions.HTTP("Api", Api)
	functions.HTTP("SendMagicLink", SendMagicLink)
	functions.HTTP("SubmitJoin", SubmitJoin)
	functions.HTTP("SubmitVehicleBasics", SubmitVehicleBasics)
	functions.HTTP("SubmitSOH", SubmitSOH)
	functions.HTTP("UpsertServiceEvent", UpsertServiceEvent)
	functions.HTTP("MemberData", MemberData)
	functions.HTTP("AdminData", AdminData)
	functions.HTTP("PublicStats", PublicStats)
}

func Api(w http.ResponseWriter, r *http.Request) {
	switch strings.TrimRight(r.URL.Path, "/") {
	case "/api/send-magic-link":
		SendMagicLink(w, r)
	case "/api/submit-join":
		SubmitJoin(w, r)
	case "/api/submit-vehicle-basics":
		SubmitVehicleBasics(w, r)
	case "/api/submit-soh":
		SubmitSOH(w, r)
	case "/api/upsert-service-event":
		UpsertServiceEvent(w, r)
	case "/api/member-data":
		MemberData(w, r)
	case "/api/admin-data":
		AdminData(w, r)
	case "/api/public-stats":
		PublicStats(w, r)
	default:
		if cors(w, r) {
			return
		}
		if rejectDisallowedOrigin(w, r) {
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "API route not found"})
	}
}

func SendMagicLink(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		logEvent("send-magic-link", "warn", "request rejected: method not allowed", map[string]any{
			"method": r.Method,
			"origin": r.Header.Get("Origin"),
		})
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}

	var req magicLinkRequest
	if err := decodeJSON(r, &req); err != nil {
		logEvent("send-magic-link", "warn", "request rejected: invalid body", map[string]any{
			"origin": r.Header.Get("Origin"),
			"error":  err.Error(),
		})
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}

	email := cleanEmail(req.Email)
	if !isEmail(email) {
		logEvent("send-magic-link", "warn", "request rejected: invalid email", map[string]any{
			"origin":      r.Header.Get("Origin"),
			"emailMasked": maskedEmail(req.Email),
		})
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Valid email address required"})
		return
	}

	fields := emailLogFields(email)
	fields["origin"] = r.Header.Get("Origin")

	joinCount, err := joinSubmissionCount(r.Context(), emailFingerprint(email))
	if err != nil {
		fields["error"] = err.Error()
		logEvent("send-magic-link", "warn", "registration check failed; email link suppressed", fields)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if joinCount == 0 {
		logEvent("send-magic-link", "info", "email link suppressed for unregistered address", fields)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	fields["joinSubmissionCount"] = joinCount
	logEvent("send-magic-link", "info", "firebase email link handoff starting", fields)

	if err := sendFirebaseEmailLink(r.Context(), email, r.Header.Get("Origin")); err != nil {
		fields := emailLogFields(email)
		fields["origin"] = r.Header.Get("Origin")
		fields["error"] = err.Error()
		logEvent("send-magic-link", "warn", "firebase email link handoff failed", fields)
	} else {
		fields := emailLogFields(email)
		fields["origin"] = r.Header.Get("Origin")
		logEvent("send-magic-link", "info", "firebase email link handoff accepted", fields)
	}

	// Do not expose whether Firebase Auth recognised the account.
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func SubmitJoin(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}

	var req joinRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	if cleanString(req.BotField, 100) != "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	email := cleanEmail(req.Email)
	name := cleanString(req.Name, 200)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Name is required"})
		return
	}
	if !isEmail(email) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Valid email address required"})
		return
	}
	if req.ConsentContact != "yes" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Contact consent is required"})
		return
	}
	if req.ConsentNotLegal != "yes" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Participation acknowledgement is required"})
		return
	}

	user, err := optionalUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Invalid sign-in token"})
		return
	}

	now := time.Now().UTC()
	record := joinRecord{
		ID:             submissionID("join"),
		Type:           "join",
		CreatedAt:      now,
		UpdatedAt:      now,
		IdentityUserID: "",
		UserEmailHash:  emailFingerprint(email),
		Contact: contactRecord{
			Name:    name,
			Email:   email,
			Country: cleanEnum(req.Country, countryValues),
		},
		Membership: membershipRecord{
			Relationship: cleanEnum(req.Relationship, relationshipValues),
			Skills:       cleanEnums([]string(req.Skills), skillValues),
		},
		Consents: consentRecord{
			Contact:            true,
			NotLegalClaim:      true,
			AnonymisedAnalysis: req.ConsentData == "yes",
		},
		Review: reviewRecord{Status: "new", VerificationLevel: "self-reported"},
	}
	if user != nil {
		record.IdentityUserID = user.UID
	}

	previousJoinCount, err := joinSubmissionCount(r.Context(), record.UserEmailHash)
	if err != nil {
		fields := emailLogFields(email)
		fields["error"] = err.Error()
		logEvent("submit-join", "warn", "repeat join check failed", fields)
	} else if previousJoinCount > 0 {
		fields := emailLogFields(email)
		fields["previousJoinCount"] = previousJoinCount
		fields["signedIn"] = user != nil
		logEvent("submit-join", "info", "repeat join email received", fields)
	}

	if err := saveJoin(r.Context(), record); err != nil {
		fields := emailLogFields(email)
		fields["error"] = err.Error()
		logEvent("submit-join", "error", "record save failed", fields)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Could not save submission"})
		return
	}
	if err := regeneratePublicStatsSnapshot(r.Context()); err != nil {
		logEvent("submit-join", "warn", "public snapshot regeneration failed", map[string]any{"error": err.Error()})
	}

	magicLinkSent := true
	if user == nil {
		fields := emailLogFields(email)
		fields["origin"] = r.Header.Get("Origin")
		logEvent("submit-join", "info", "firebase email link handoff starting", fields)
		if err := sendFirebaseEmailLink(r.Context(), email, r.Header.Get("Origin")); err != nil {
			magicLinkSent = false
			fields := emailLogFields(email)
			fields["origin"] = r.Header.Get("Origin")
			fields["error"] = err.Error()
			logEvent("submit-join", "warn", "firebase email link handoff failed", fields)
		} else {
			fields := emailLogFields(email)
			fields["origin"] = r.Header.Get("Origin")
			logEvent("submit-join", "info", "firebase email link handoff accepted", fields)
		}
	} else {
		if err := regenerateMemberSnapshot(r.Context(), user.UID, user.Email); err != nil {
			logEvent("submit-join", "warn", "snapshot regeneration failed", map[string]any{"uid": user.UID, "error": err.Error()})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"id":            record.ID,
		"magicLinkSent": magicLinkSent,
		"signedIn":      user != nil,
	})
}

func SubmitVehicleBasics(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}

	user, err := requireUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return
	}

	var req vehicleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}

	vin, registration, ignoredInvalidVIN, validationMessage := vehicleIdentifiers(req)
	if validationMessage != "" {
		logEvent("submit-vehicle-basics", "warn", "request rejected: invalid vehicle identifiers", map[string]any{
			"uid":             user.UID,
			"reason":          validationMessage,
			"hasVin":          cleanString(req.VIN, 40) != "",
			"vinLength":       len(normalizeVIN(req.VIN)),
			"hasRegistration": cleanString(req.Registration, 40) != "",
		})
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": validationMessage})
		return
	}
	if ignoredInvalidVIN {
		logEvent("submit-vehicle-basics", "warn", "invalid optional VIN ignored for registration-based save", map[string]any{
			"uid":             user.UID,
			"vinLength":       len(normalizeVIN(req.VIN)),
			"hasRegistration": true,
		})
	}
	if validationMessage := vehicleDateValidationMessage(req, time.Now().UTC()); validationMessage != "" {
		logEvent("submit-vehicle-basics", "warn", "request rejected: future vehicle date", map[string]any{
			"uid":    user.UID,
			"reason": validationMessage,
		})
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": validationMessage})
		return
	}

	pepper := os.Getenv("VIN_PEPPER")
	if vin != "" && pepper == "" && registration == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Vehicle identifier storage is not configured. Provide registration instead, or try again later."})
		return
	}

	now := time.Now().UTC()
	vehicle := vehicleDetails{
		Registration:          registration,
		Country:               cleanEnum(req.Country, countryValues),
		ModelYear:             cleanEnum(req.ModelYear, modelYearValues),
		Mileage:               cleanInt(req.Mileage, 0, 500000),
		OwnedSince:            cleanDate(req.OwnedSince),
		FirstRegistrationDate: cleanDate(req.FirstReg),
	}
	if vin != "" && pepper != "" {
		vehicle.VINHash = hmacValue(vin, pepper)
		vehicle.VINLast6 = vin[len(vin)-6:]
	}

	record := vehicleRecord{
		ID:             submissionID("vehicle"),
		Type:           "vehicle-basics",
		CreatedAt:      now,
		UpdatedAt:      now,
		IdentityUserID: user.UID,
		UserEmailHash:  emailFingerprint(user.Email),
		Vehicle:        vehicle,
		Battery: batteryDetails{
			StateOfHealth:        cleanDecimal(req.SOH, 0, 100),
			MeasuredAt:           cleanDate(req.SOHDate),
			MileageAtMeasurement: cleanInt(req.SOHMileage, 0, 500000),
			Source:               cleanEnum(req.SOHSource, sohSourceValues),
		},
		Review: reviewRecord{Status: "new", VerificationLevel: "self-reported"},
	}

	if err := saveVehicle(r.Context(), record, initialBatteryReading(record)); err != nil {
		logEvent("submit-vehicle-basics", "error", "record save failed", map[string]any{"uid": user.UID, "error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Could not save vehicle basics"})
		return
	}
	if err := regenerateMemberSnapshot(r.Context(), user.UID, user.Email); err != nil {
		logEvent("submit-vehicle-basics", "warn", "snapshot regeneration failed", map[string]any{"uid": user.UID, "error": err.Error()})
	}
	if err := regeneratePublicStatsSnapshot(r.Context()); err != nil {
		logEvent("submit-vehicle-basics", "warn", "public snapshot regeneration failed", map[string]any{"error": err.Error()})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": record.ID})
}

func normalizeVIN(value string) string {
	value = strings.ToUpper(cleanString(value, 40))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	return value
}

func vehicleIdentifiers(req vehicleRequest) (string, string, bool, string) {
	vin := normalizeVIN(req.VIN)
	registration := strings.ToUpper(cleanString(req.Registration, 40))
	hasVIN := vin != ""
	hasRegistration := registration != ""

	if !hasVIN && !hasRegistration {
		return "", "", false, "VIN or registration is required"
	}
	if hasVIN && !vinRE.MatchString(vin) {
		if hasRegistration {
			return "", registration, true, ""
		}
		return "", "", false, "VIN must be 17 characters and cannot contain I, O, or Q"
	}
	return vin, registration, false, ""
}

func vehicleDateValidationMessage(req vehicleRequest, now time.Time) string {
	if dateIsFuture(req.OwnedSince, now) {
		return "Owned since cannot be in the future"
	}
	if dateIsFuture(req.FirstReg, now) {
		return "First registration date cannot be in the future"
	}
	if dateIsFuture(req.SOHDate, now) {
		return "State of Health measurement date cannot be in the future"
	}
	return ""
}

func dateIsFuture(value string, now time.Time) bool {
	value = cleanString(value, 20)
	if value == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return parsed.After(today)
}

func MemberData(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	user, err := requireUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return
	}
	snapshot, err := loadMemberSnapshot(r.Context(), user.UID, user.Email)
	if err != nil {
		logEvent("member-data", "error", "snapshot load failed", map[string]any{"uid": user.UID, "error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load member data"})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func AdminData(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	user, err := requireUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return
	}
	if !isAdmin(user) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}

	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not connect to data store"})
		return
	}
	data := adminData{JoinRecords: []joinRecord{}, VehicleRecords: []vehicleRecord{}}
	if err := readCollection(r.Context(), db.Collection("joinSubmissions").OrderBy("createdAt", firestore.Desc), &data.JoinRecords); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load join submissions"})
		return
	}
	if err := readCollection(r.Context(), db.Collection("vehicles").OrderBy("createdAt", firestore.Desc), &data.VehicleRecords); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load vehicle submissions"})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

var sendFirebaseEmailLink = sendFirebaseEmailLinkRequest

func sendFirebaseEmailLinkRequest(ctx context.Context, email string, origin string) error {
	apiKey := os.Getenv("FIREBASE_WEB_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("FIREBASE_WEB_API_KEY is not configured")
	}
	continueURL := emailContinueURLForOrigin(origin)
	linkDomain := os.Getenv("FIREBASE_EMAIL_LINK_DOMAIN")
	if linkDomain == "" {
		linkDomain = firebaseEmailLinkDomainForContinueURL(continueURL)
	}
	fields := emailLogFields(email)
	fields["continueHost"] = urlHost(continueURL)
	if linkDomain != "" {
		fields["linkDomain"] = linkDomain
	}
	if resendEmailConfigured() {
		fields["provider"] = "resend"
		logEvent("firebase-email-link", "info", "custom email link request prepared", fields)
		actionLink, err := generateFirebaseEmailSignInLink(ctx, email, continueURL, linkDomain)
		if err == nil {
			err = sendResendMagicLinkEmail(ctx, email, actionLink, continueURL)
		}
		if err == nil {
			logEvent("firebase-email-link", "info", "custom email link request accepted", fields)
			return nil
		}
		fields["error"] = err.Error()
		logEvent("firebase-email-link", "warn", "custom email link failed; falling back to Firebase default email", fields)
		delete(fields, "error")
	}
	fields["provider"] = "firebase-default"
	logEvent("firebase-email-link", "info", "identity toolkit request prepared", fields)
	payload := firebaseEmailLinkPayload(email, continueURL, linkDomain)
	body, _ := json.Marshal(payload)
	endpoint := "https://identitytoolkit.googleapis.com/v1/accounts:sendOobCode?key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(res.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("identity toolkit response read failed: %w", readErr)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("identity toolkit returned %d: %s", res.StatusCode, identityToolkitErrorMessage(strings.NewReader(string(responseBody))))
	}
	fields = identityToolkitSuccessFields(responseBody, email, continueURL)
	logEvent("firebase-email-link", "info", "identity toolkit request accepted", fields)
	return nil
}

func firebaseEmailLinkPayload(email string, continueURL string, linkDomain string) map[string]any {
	payload := map[string]any{
		"requestType":        "EMAIL_SIGNIN",
		"email":              email,
		"continueUrl":        continueURL,
		"canHandleCodeInApp": true,
	}
	if linkDomain != "" {
		payload["linkDomain"] = linkDomain
	}
	return payload
}

func identityToolkitSuccessFields(body []byte, email string, continueURL string) map[string]any {
	var response struct {
		Email string `json:"email"`
	}
	_ = json.Unmarshal(body, &response)
	fields := emailLogFields(email)
	fields["continueHost"] = urlHost(continueURL)
	fields["providerEmailMatched"] = cleanEmail(response.Email) == cleanEmail(email)
	fields["responseBytes"] = len(body)
	return fields
}

func identityToolkitErrorMessage(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil || len(data) == 0 {
		return "empty error response"
	}
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err == nil {
		message := cleanString(parsed.Error.Message, 300)
		status := cleanString(parsed.Error.Status, 100)
		if message != "" && status != "" {
			return status + ": " + message
		}
		if message != "" {
			return message
		}
		if status != "" {
			return status
		}
	}
	return cleanString(string(data), 300)
}

func saveJoin(ctx context.Context, record joinRecord) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	_, err = db.Collection("joinSubmissions").Doc(record.ID).Set(ctx, record)
	if err != nil {
		return err
	}
	if record.IdentityUserID != "" {
		return upsertMember(ctx, record.IdentityUserID, record.Contact.Email, record.Contact.Name, record.Contact.Country, record.Membership.Relationship)
	}
	return nil
}

var joinSubmissionCount = countJoinSubmissions

func countJoinSubmissions(ctx context.Context, emailHash string) (int, error) {
	if emailHash == "" {
		return 0, nil
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return 0, err
	}
	iter := db.Collection("joinSubmissions").Where("userEmailHash", "==", emailHash).Documents(ctx)
	defer iter.Stop()
	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			return count, nil
		}
		if err != nil {
			return count, err
		}
		count++
	}
}

func saveVehicle(ctx context.Context, record vehicleRecord, reading *batteryReadingRecord) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	batch := db.Batch()
	batch.Set(db.Collection("vehicles").Doc(record.ID), record)
	if reading != nil {
		batch.Set(db.Collection("batteryReadings").Doc(reading.ID), reading)
	}
	batch.Set(db.Collection("members").Doc(record.IdentityUserID), map[string]any{
		"identityUserId": record.IdentityUserID,
		"updatedAt":      time.Now().UTC(),
	}, firestore.MergeAll)
	_, err = batch.Commit(ctx)
	return err
}

func upsertMember(ctx context.Context, uid string, email string, name string, country string, relationship string) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	update := map[string]any{
		"identityUserId": uid,
		"updatedAt":      time.Now().UTC(),
	}
	if email != "" {
		update["emailHash"] = emailFingerprint(email)
	}
	if name != "" {
		update["displayName"] = name
	}
	if country != "" {
		update["country"] = country
	}
	if relationship != "" {
		update["relationship"] = relationship
	}
	_, err = db.Collection("members").Doc(uid).Set(ctx, update, firestore.MergeAll)
	return err
}

func regenerateMemberSnapshot(ctx context.Context, uid string, email string) error {
	snapshot, err := buildMemberSnapshot(ctx, uid, email)
	if err != nil {
		return err
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	if _, err := db.Collection("memberSnapshots").Doc(uid).Set(ctx, snapshot); err != nil {
		return err
	}
	return writeSnapshotObject(ctx, snapshot)
}

func loadMemberSnapshot(ctx context.Context, uid string, email string) (memberSnapshot, error) {
	if snapshot, err := readSnapshotObject(ctx, uid); err == nil {
		return snapshot, nil
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return memberSnapshot{}, err
	}
	var snapshot memberSnapshot
	doc, err := db.Collection("memberSnapshots").Doc(uid).Get(ctx)
	if err == nil && doc.Exists() {
		if err := doc.DataTo(&snapshot); err == nil {
			return snapshot, nil
		}
	}
	snapshot, err = buildMemberSnapshot(ctx, uid, email)
	if err != nil {
		return memberSnapshot{}, err
	}
	_, _ = db.Collection("memberSnapshots").Doc(uid).Set(ctx, snapshot)
	_ = writeSnapshotObject(ctx, snapshot)
	return snapshot, nil
}

func buildMemberSnapshot(ctx context.Context, uid string, email string) (memberSnapshot, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return memberSnapshot{}, err
	}
	snapshot := memberSnapshot{
		IdentityUserID:  uid,
		Email:           email,
		GeneratedAt:     time.Now().UTC(),
		JoinRecords:     []joinRecord{},
		VehicleRecords:  []vehicleRecord{},
		BatteryReadings: []batteryReadingRecord{},
		ServiceEvents:   []serviceEventRecord{},
	}
	if email != "" {
		emailHash := emailFingerprint(email)
		iter := db.Collection("joinSubmissions").Where("userEmailHash", "==", emailHash).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return memberSnapshot{}, err
			}
			var record joinRecord
			if err := doc.DataTo(&record); err == nil {
				record.IdentityUserID = uid
				record.Contact.Email = ""
				snapshot.JoinRecords = append(snapshot.JoinRecords, record)
			}
		}
	}
	iter := db.Collection("vehicles").Where("identityUserId", "==", uid).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memberSnapshot{}, err
		}
		var record vehicleRecord
		if err := doc.DataTo(&record); err == nil {
			snapshot.VehicleRecords = append(snapshot.VehicleRecords, record)
		}
	}
	readingIter := db.Collection("batteryReadings").Where("identityUserId", "==", uid).Documents(ctx)
	for {
		doc, err := readingIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memberSnapshot{}, err
		}
		var record batteryReadingRecord
		if err := doc.DataTo(&record); err == nil {
			snapshot.BatteryReadings = append(snapshot.BatteryReadings, record)
		}
	}
	eventIter := db.Collection("serviceEvents").Where("identityUserId", "==", uid).Documents(ctx)
	for {
		doc, err := eventIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return memberSnapshot{}, err
		}
		var record serviceEventRecord
		if err := doc.DataTo(&record); err == nil {
			snapshot.ServiceEvents = append(snapshot.ServiceEvents, record)
		}
	}
	readVehicles := map[string]bool{}
	for _, reading := range snapshot.BatteryReadings {
		readVehicles[reading.VehicleID] = true
	}
	for _, vehicle := range snapshot.VehicleRecords {
		if !readVehicles[vehicle.ID] {
			if reading := initialBatteryReading(vehicle); reading != nil {
				snapshot.BatteryReadings = append(snapshot.BatteryReadings, *reading)
			}
		}
	}
	sort.Slice(snapshot.JoinRecords, func(i, j int) bool {
		return snapshot.JoinRecords[i].CreatedAt.After(snapshot.JoinRecords[j].CreatedAt)
	})
	sort.Slice(snapshot.VehicleRecords, func(i, j int) bool {
		return snapshot.VehicleRecords[i].CreatedAt.After(snapshot.VehicleRecords[j].CreatedAt)
	})
	sort.Slice(snapshot.BatteryReadings, func(i, j int) bool {
		return normalisedMeasurementDate(snapshot.BatteryReadings[i]).After(normalisedMeasurementDate(snapshot.BatteryReadings[j]))
	})
	sort.Slice(snapshot.ServiceEvents, func(i, j int) bool {
		if snapshot.ServiceEvents[i].OccurredAt == snapshot.ServiceEvents[j].OccurredAt {
			return snapshot.ServiceEvents[i].UpdatedAt.After(snapshot.ServiceEvents[j].UpdatedAt)
		}
		return snapshot.ServiceEvents[i].OccurredAt > snapshot.ServiceEvents[j].OccurredAt
	})
	return snapshot, nil
}

func readCollection[T any](ctx context.Context, query firestore.Query, dest *[]T) error {
	iter := query.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var item T
		if err := doc.DataTo(&item); err == nil {
			*dest = append(*dest, item)
		}
	}
}

func writeSnapshotObject(ctx context.Context, snapshot memberSnapshot) error {
	bucketName := os.Getenv("SNAPSHOT_BUCKET")
	if bucketName == "" {
		return nil
	}
	client, err := gcsClient(ctx)
	if err != nil {
		return err
	}
	writer := client.Bucket(bucketName).Object("members/" + snapshot.IdentityUserID + ".json").NewWriter(ctx)
	writer.ContentType = "application/json"
	if err := json.NewEncoder(writer).Encode(snapshot); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func readSnapshotObject(ctx context.Context, uid string) (memberSnapshot, error) {
	bucketName := os.Getenv("SNAPSHOT_BUCKET")
	if bucketName == "" {
		return memberSnapshot{}, fmt.Errorf("SNAPSHOT_BUCKET is not configured")
	}
	client, err := gcsClient(ctx)
	if err != nil {
		return memberSnapshot{}, err
	}
	reader, err := client.Bucket(bucketName).Object("members/" + uid + ".json").NewReader(ctx)
	if err != nil {
		return memberSnapshot{}, err
	}
	defer reader.Close()
	var snapshot memberSnapshot
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		return memberSnapshot{}, err
	}
	return snapshot, nil
}
