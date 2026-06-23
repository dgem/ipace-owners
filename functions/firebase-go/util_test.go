package ipace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOriginAllowed(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://staging.example.com")
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-staging")

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://ipace-owners.org", true},
		{"https://staging.example.com", true},
		{"https://ipace-owners-staging--pr-20-ef2wibc5.web.app", true},
		{"https://ipace-owners-staging--pr-20-ef2wibc5.firebaseapp.com", true},
		{"https://example.web.app", false},
		{"https://ipace-owners-production--pr-20-ef2wibc5.web.app", false},
		{"http://localhost:5000", true},
		{"https://evil.example", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := originAllowed(tc.origin); got != tc.want {
			t.Fatalf("originAllowed(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}

func TestEmailContinueURLUsesAllowedRequestOrigin(t *testing.T) {
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-staging")
	t.Setenv("FIREBASE_EMAIL_CONTINUE_URL", "https://stage.ipace-owners.org/account/")

	got := emailContinueURLForOrigin("https://ipace-owners-staging--pr-20-ef2wibc5.web.app")

	if got != "https://ipace-owners-staging--pr-20-ef2wibc5.web.app/account/" {
		t.Fatalf("continue URL = %q", got)
	}
}

func TestEmailContinueURLFallsBackForDisallowedOrigin(t *testing.T) {
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-staging")
	t.Setenv("FIREBASE_EMAIL_CONTINUE_URL", "https://stage.ipace-owners.org/account/")

	got := emailContinueURLForOrigin("https://evil.example")

	if got != "https://stage.ipace-owners.org/account/" {
		t.Fatalf("continue URL = %q", got)
	}
}

func TestCleaners(t *testing.T) {
	if got := cleanEmail(" TEST@Example.COM "); got != "test@example.com" {
		t.Fatalf("cleanEmail() = %q", got)
	}
	if !isEmail("member@example.com") {
		t.Fatal("isEmail rejected valid email")
	}
	if isEmail("not-an-email") {
		t.Fatal("isEmail accepted invalid email")
	}
	if got := cleanEnum("GB", countryValues); got != "GB" {
		t.Fatalf("cleanEnum() = %q", got)
	}
	if got := cleanEnum("XX", countryValues); got != "" {
		t.Fatalf("cleanEnum(invalid) = %q", got)
	}
	if got := cleanDate("2026-06-16"); got != "2026-06-16" {
		t.Fatalf("cleanDate() = %q", got)
	}
	if got := cleanDate("16/06/2026"); got != "" {
		t.Fatalf("cleanDate(invalid) = %q", got)
	}
	if got := cleanInt("123", 0, 500); got == nil || *got != 123 {
		t.Fatalf("cleanInt() = %v", got)
	}
	if got := cleanInt("999", 0, 500); got != nil {
		t.Fatalf("cleanInt(out of range) = %v", *got)
	}
	if got := cleanDecimal("88.86", 0, 100); got == nil || *got != 88.9 {
		t.Fatalf("cleanDecimal() = %v", got)
	}
}

func TestEmailLogHelpersDoNotExposeRawEmail(t *testing.T) {
	email := "Driver.Name@example.co.uk"
	if got := maskedEmail(email); got != "d***@e***.uk" {
		t.Fatalf("maskedEmail() = %q", got)
	}

	fields := emailLogFields(email)
	if fields["emailHash"] == "" {
		t.Fatal("emailLogFields missing emailHash")
	}
	if fields["emailMasked"] != "d***@e***.uk" {
		t.Fatalf("emailMasked = %q", fields["emailMasked"])
	}
	for _, value := range fields {
		if value == cleanEmail(email) {
			t.Fatal("emailLogFields exposed raw email")
		}
	}
}

func TestURLHost(t *testing.T) {
	if got := urlHost("https://stage.ipace-owners.org/account/?x=1"); got != "stage.ipace-owners.org" {
		t.Fatalf("urlHost() = %q", got)
	}
}

func TestHMACDoesNotExposeRawVIN(t *testing.T) {
	vin := "SADHA2B10K1F12345"
	digest := hmacValue(vin, "test-secret")
	if digest == "" {
		t.Fatal("hmacValue returned empty digest")
	}
	if digest == vin {
		t.Fatal("hmacValue returned raw VIN")
	}
	if len(digest) != 64 {
		t.Fatalf("hmacValue length = %d, want 64", len(digest))
	}
}

func TestVehicleIdentifiersRequireVINOrRegistration(t *testing.T) {
	vin, registration, ignoredVIN, message := vehicleIdentifiers(vehicleRequest{})

	if vin != "" || registration != "" || ignoredVIN {
		t.Fatalf("vehicleIdentifiers returned identifiers for empty request: vin=%q registration=%q ignored=%v", vin, registration, ignoredVIN)
	}
	if message != "VIN or registration is required" {
		t.Fatalf("message = %q", message)
	}
}

func TestVehicleIdentifiersRejectVINOnlyInvalid(t *testing.T) {
	vin, registration, ignoredVIN, message := vehicleIdentifiers(vehicleRequest{VIN: "BADVIN"})

	if vin != "" || registration != "" || ignoredVIN {
		t.Fatalf("vehicleIdentifiers returned identifiers for invalid VIN-only request: vin=%q registration=%q ignored=%v", vin, registration, ignoredVIN)
	}
	if message != "VIN must be 17 characters and cannot contain I, O, or Q" {
		t.Fatalf("message = %q", message)
	}
}

func TestVehicleIdentifiersIgnoreInvalidOptionalVINWithRegistration(t *testing.T) {
	vin, registration, ignoredVIN, message := vehicleIdentifiers(vehicleRequest{
		VIN:          "BADVIN",
		Registration: " hv69 voo ",
	})

	if vin != "" {
		t.Fatalf("vin = %q, want empty", vin)
	}
	if registration != "HV69 VOO" {
		t.Fatalf("registration = %q", registration)
	}
	if !ignoredVIN {
		t.Fatal("invalid optional VIN was not reported as ignored")
	}
	if message != "" {
		t.Fatalf("message = %q", message)
	}
}

func TestVehicleIdentifiersNormalizeValidVIN(t *testing.T) {
	vin, registration, ignoredVIN, message := vehicleIdentifiers(vehicleRequest{
		VIN:          "sadha2b10-k1f12345",
		Registration: " hv69 voo ",
	})

	if vin != "SADHA2B10K1F12345" {
		t.Fatalf("vin = %q", vin)
	}
	if registration != "HV69 VOO" {
		t.Fatalf("registration = %q", registration)
	}
	if ignoredVIN {
		t.Fatal("valid VIN should not be ignored")
	}
	if message != "" {
		t.Fatalf("message = %q", message)
	}
}

func TestVehicleDateValidationRejectsFutureDates(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		req  vehicleRequest
		want string
	}{
		{
			name: "owned since",
			req:  vehicleRequest{OwnedSince: "2026-06-25"},
			want: "Owned since cannot be in the future",
		},
		{
			name: "first registration",
			req:  vehicleRequest{FirstReg: "2026-06-25"},
			want: "First registration date cannot be in the future",
		},
		{
			name: "soh date",
			req:  vehicleRequest{SOHDate: "2026-06-25"},
			want: "State of Health measurement date cannot be in the future",
		},
		{
			name: "today accepted",
			req:  vehicleRequest{OwnedSince: "2026-06-24", FirstReg: "2026-06-24", SOHDate: "2026-06-24"},
			want: "",
		},
		{
			name: "past accepted",
			req:  vehicleRequest{OwnedSince: "2026-06-23", FirstReg: "2020-01-01", SOHDate: "2026-01-01"},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := vehicleDateValidationMessage(tc.req, now); got != tc.want {
				t.Fatalf("vehicleDateValidationMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProjectIDFallbacks(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("FIREBASE_PROJECT_ID", "firebase-project")
	t.Setenv("PROJECT_ID", "generic-project")

	if got := projectID(); got != "firebase-project" {
		t.Fatalf("projectID() = %q, want firebase-project", got)
	}
}

func TestFirestoreDatabaseIDUsesConfiguredNamedDatabase(t *testing.T) {
	t.Setenv("FIRESTORE_DATABASE_ID", "ipace-owners-production")
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-production")

	if got := firestoreDatabaseID(); got != "ipace-owners-production" {
		t.Fatalf("firestoreDatabaseID() = %q", got)
	}
}

func TestFirestoreDatabaseIDFallsBackToProjectID(t *testing.T) {
	t.Setenv("FIRESTORE_DATABASE_ID", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-staging")

	if got := firestoreDatabaseID(); got != "ipace-owners-staging" {
		t.Fatalf("firestoreDatabaseID() = %q", got)
	}
}

func TestIdentityToolkitErrorMessage(t *testing.T) {
	body := `{"error":{"code":400,"message":"INVALID_CONTINUE_URI : Continue URL is not whitelisted.","status":"INVALID_ARGUMENT"}}`

	got := identityToolkitErrorMessage(strings.NewReader(body))

	if !strings.Contains(got, "INVALID_ARGUMENT") || !strings.Contains(got, "INVALID_CONTINUE_URI") {
		t.Fatalf("identityToolkitErrorMessage() = %q", got)
	}
}

func TestIdentityToolkitSuccessFields(t *testing.T) {
	fields := identityToolkitSuccessFields(
		[]byte(`{"email":"driver@example.com"}`),
		"driver@example.com",
		"https://stage.ipace-owners.org/account/",
	)

	if fields["providerEmailMatched"] != true {
		t.Fatalf("providerEmailMatched = %v", fields["providerEmailMatched"])
	}
	if fields["continueHost"] != "stage.ipace-owners.org" {
		t.Fatalf("continueHost = %v", fields["continueHost"])
	}
	if fields["emailMasked"] != "d***@e***.com" {
		t.Fatalf("emailMasked = %v", fields["emailMasked"])
	}
	if fields["responseBytes"] != len(`{"email":"driver@example.com"}`) {
		t.Fatalf("responseBytes = %v", fields["responseBytes"])
	}
}

func TestFirebaseEmailLinkPayloadUsesCustomHostingDomain(t *testing.T) {
	payload := firebaseEmailLinkPayload(
		"driver@example.com",
		"https://ipace-owners.org/account/",
		"ipace-owners.org",
	)

	if payload["linkDomain"] != "ipace-owners.org" {
		t.Fatalf("linkDomain = %v", payload["linkDomain"])
	}
	if payload["continueUrl"] != "https://ipace-owners.org/account/" {
		t.Fatalf("continueUrl = %v", payload["continueUrl"])
	}
	if payload["canHandleCodeInApp"] != true {
		t.Fatalf("canHandleCodeInApp = %v", payload["canHandleCodeInApp"])
	}
}

func TestFirebaseEmailLinkPayloadOmitsCustomDomainForPreview(t *testing.T) {
	payload := firebaseEmailLinkPayload(
		"driver@example.com",
		"https://ipace-owners-staging--pr-20-ef2wibc5.web.app/account/",
		"",
	)

	if _, present := payload["linkDomain"]; present {
		t.Fatal("linkDomain should be omitted for Firebase Hosting previews")
	}
}

func TestCORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/send-magic-link", nil)
	req.Header.Set("Origin", "https://ipace-owners.org")
	rec := httptest.NewRecorder()

	if !cors(rec, req) {
		t.Fatal("cors did not handle preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ipace-owners.org" {
		t.Fatalf("allow-origin = %q", got)
	}
}

func TestApiRoutesKnownAndUnknownPaths(t *testing.T) {
	known := httptest.NewRequest(http.MethodPost, "/api/send-magic-link", strings.NewReader(`{}`))
	known.Header.Set("Origin", "https://ipace-owners.org")
	knownRec := httptest.NewRecorder()

	Api(knownRec, known)

	if knownRec.Code != http.StatusBadRequest {
		t.Fatalf("known route status = %d, want handler response 400", knownRec.Code)
	}

	unknown := httptest.NewRequest(http.MethodGet, "/api/not-a-route", nil)
	unknown.Header.Set("Origin", "https://ipace-owners.org")
	unknownRec := httptest.NewRecorder()

	Api(unknownRec, unknown)

	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want 404", unknownRec.Code)
	}
}
