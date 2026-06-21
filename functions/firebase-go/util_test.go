package ipace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOriginAllowed(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://staging.example.com")

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://ipace-owners.org", true},
		{"https://staging.example.com", true},
		{"https://example.web.app", true},
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

func TestProjectIDFallbacks(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCP_PROJECT", "")
	t.Setenv("FIREBASE_PROJECT_ID", "firebase-project")
	t.Setenv("PROJECT_ID", "generic-project")

	if got := projectID(); got != "firebase-project" {
		t.Fatalf("projectID() = %q, want firebase-project", got)
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
