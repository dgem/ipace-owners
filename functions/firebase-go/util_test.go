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
	t.Setenv("FIREBASE_EMAIL_CONTINUE_URL", "https://stage.ipace-owners.org/member/account/")

	got := emailContinueURLForOrigin("https://ipace-owners-staging--pr-20-ef2wibc5.web.app")

	if got != "https://ipace-owners-staging--pr-20-ef2wibc5.web.app/member/account/" {
		t.Fatalf("continue URL = %q", got)
	}
}

func TestEmailContinueURLFallsBackForDisallowedOrigin(t *testing.T) {
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-staging")
	t.Setenv("FIREBASE_EMAIL_CONTINUE_URL", "https://stage.ipace-owners.org/member/account/")

	got := emailContinueURLForOrigin("https://evil.example")

	if got != "https://stage.ipace-owners.org/member/account/" {
		t.Fatalf("continue URL = %q", got)
	}
}

func TestFirebaseEmailLinkDomainForContinueURLUsesCustomHostingDomain(t *testing.T) {
	cases := []struct {
		name        string
		continueURL string
		want        string
	}{
		{
			name:        "production custom domain",
			continueURL: "https://ipace-owners.org/member/account/",
			want:        "ipace-owners.org",
		},
		{
			name:        "staging custom domain",
			continueURL: "https://stage.ipace-owners.org/member/account/",
			want:        "stage.ipace-owners.org",
		},
		{
			name:        "preview web app",
			continueURL: "https://ipace-owners-staging--pr-20-ef2wibc5.web.app/member/account/",
			want:        "",
		},
		{
			name:        "preview firebaseapp",
			continueURL: "https://ipace-owners-staging--pr-20-ef2wibc5.firebaseapp.com/member/account/",
			want:        "",
		},
		{
			name:        "localhost",
			continueURL: "http://localhost:8080/member/account/",
			want:        "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := firebaseEmailLinkDomainForContinueURL(tc.continueURL); got != tc.want {
				t.Fatalf("firebaseEmailLinkDomainForContinueURL(%q) = %q, want %q", tc.continueURL, got, tc.want)
			}
		})
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
	if got := urlHost("https://stage.ipace-owners.org/member/account/?x=1"); got != "stage.ipace-owners.org" {
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
		"https://stage.ipace-owners.org/member/account/",
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
		"https://ipace-owners.org/member/account/",
		"ipace-owners.org",
	)

	if payload["linkDomain"] != "ipace-owners.org" {
		t.Fatalf("linkDomain = %v", payload["linkDomain"])
	}
	if payload["continueUrl"] != "https://ipace-owners.org/member/account/" {
		t.Fatalf("continueUrl = %v", payload["continueUrl"])
	}
	if payload["canHandleCodeInApp"] != true {
		t.Fatalf("canHandleCodeInApp = %v", payload["canHandleCodeInApp"])
	}
}

func TestFirebaseEmailLinkPayloadOmitsCustomDomainForPreview(t *testing.T) {
	payload := firebaseEmailLinkPayload(
		"driver@example.com",
		"https://ipace-owners-staging--pr-20-ef2wibc5.web.app/member/account/",
		"",
	)

	if _, present := payload["linkDomain"]; present {
		t.Fatal("linkDomain should be omitted for Firebase Hosting previews")
	}
}

func TestJoinRecordFromRequestCapturesOptionalAggregateConsent(t *testing.T) {
	now := time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC)
	record := joinRecordFromRequest(joinRequest{
		Country:      "GB",
		Relationship: "current-owner-one",
		Skills:       stringArray{"legal", "data"},
		ConsentData:  "yes",
	}, "Driver Example", "driver@example.com", now)

	if record.Contact.Name != "Driver Example" {
		t.Fatalf("name = %q", record.Contact.Name)
	}
	if record.Contact.Email != "driver@example.com" {
		t.Fatalf("email = %q", record.Contact.Email)
	}
	if record.Consents.Contact != true {
		t.Fatalf("contact consent = %v", record.Consents.Contact)
	}
	if record.Consents.NotLegalClaim != true {
		t.Fatalf("not legal consent = %v", record.Consents.NotLegalClaim)
	}
	if record.Consents.AnonymisedAnalysis != true {
		t.Fatalf("anonymised consent = %v", record.Consents.AnonymisedAnalysis)
	}
	if record.CreatedAt != now || record.UpdatedAt != now {
		t.Fatalf("timestamps = %v/%v", record.CreatedAt, record.UpdatedAt)
	}

	withoutConsent := joinRecordFromRequest(joinRequest{}, "Driver Example", "driver@example.com", now)
	if withoutConsent.Consents.AnonymisedAnalysis {
		t.Fatal("anonymised consent should default to false")
	}
}

func TestFirebaseEmailActionCodeSettings(t *testing.T) {
	settings := firebaseEmailActionCodeSettings("https://ipace-owners.org/member/account/", "ipace-owners.org")

	if settings.URL != "https://ipace-owners.org/member/account/" {
		t.Fatalf("URL = %q", settings.URL)
	}
	if settings.HandleCodeInApp != true {
		t.Fatalf("HandleCodeInApp = %v", settings.HandleCodeInApp)
	}
	if settings.LinkDomain != "ipace-owners.org" {
		t.Fatalf("LinkDomain = %q", settings.LinkDomain)
	}

	previewSettings := firebaseEmailActionCodeSettings("https://ipace-owners-staging--pr-20-ef2wibc5.web.app/member/account/", "")
	if previewSettings.LinkDomain != "" {
		t.Fatalf("preview LinkDomain = %q", previewSettings.LinkDomain)
	}
}

func TestResendMagicLinkPayloadUsesHeroImageAndReplyTo(t *testing.T) {
	t.Setenv("RESEND_FROM", "I-PACE Owners <members@ipace-owners.org>")
	t.Setenv("RESEND_REPLY_TO", "contact@ipace-owners.org")
	t.Setenv("RESEND_ASSET_BASE_URL", "https://ipace-owners.org")

	payload := resendMagicLinkPayload(
		"driver@example.com",
		"https://ipace-owners.org/__/auth/action?mode=signIn&oobCode=secret",
		"https://ipace-owners.org/member/account/",
	)

	if payload["from"] != "I-PACE Owners <members@ipace-owners.org>" {
		t.Fatalf("from = %v", payload["from"])
	}
	if payload["reply_to"] != "contact@ipace-owners.org" {
		t.Fatalf("reply_to = %v", payload["reply_to"])
	}
	htmlBody, ok := payload["html"].(string)
	if !ok {
		t.Fatal("html payload missing")
	}
	if !strings.Contains(htmlBody, "https://ipace-owners.org/images/ipace-hero.png") {
		t.Fatalf("html does not include hero image: %s", htmlBody)
	}
	if !strings.Contains(htmlBody, "Sign in securely") {
		t.Fatalf("html does not include CTA")
	}
	textBody, ok := payload["text"].(string)
	if !ok {
		t.Fatal("text payload missing")
	}
	if !strings.Contains(textBody, "https://ipace-owners.org/__/auth/action") {
		t.Fatalf("text does not include action link: %s", textBody)
	}
}

func TestEmailAssetBaseURLUsesPreviewOriginAndAvoidsGenericFirebaseHosts(t *testing.T) {
	cases := []struct {
		name        string
		continueURL string
		envBaseURL  string
		want        string
	}{
		{
			name:        "custom domain",
			continueURL: "https://stage.ipace-owners.org/member/account/",
			want:        "https://stage.ipace-owners.org",
		},
		{
			name:        "preview host overrides static staging asset base",
			continueURL: "https://ipace-owners-staging--pr-20-ef2wibc5.web.app/member/account/",
			envBaseURL:  "https://stage.ipace-owners.org",
			want:        "https://ipace-owners-staging--pr-20-ef2wibc5.web.app",
		},
		{
			name:        "generic Firebase host",
			continueURL: "https://ipace-owners-staging.web.app/member/account/",
			want:        "https://ipace-owners.org",
		},
		{
			name:        "local",
			continueURL: "http://localhost:8080/member/account/",
			want:        "https://ipace-owners.org",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envBaseURL != "" {
				t.Setenv("RESEND_ASSET_BASE_URL", tc.envBaseURL)
			}
			if got := emailAssetBaseURL(tc.continueURL); got != tc.want {
				t.Fatalf("emailAssetBaseURL(%q) = %q, want %q", tc.continueURL, got, tc.want)
			}
		})
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
