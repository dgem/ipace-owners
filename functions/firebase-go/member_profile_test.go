package ipace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemberProfileRepresentationRequiresUKAddress(t *testing.T) {
	valid := memberContactProfile{Name: "Alex Owner", AddressLine1: "1 Example Road", City: "London", PostalCode: "SW1A 1AA", Country: "United Kingdom", Represent: true}
	if got := validateMemberContactProfile(&valid); got != "" {
		t.Fatalf("valid profile rejected: %s", got)
	}
	for _, profile := range []memberContactProfile{
		{Name: valid.Name, AddressLine1: valid.AddressLine1, City: valid.City, PostalCode: valid.PostalCode, Country: "France", Represent: true},
		{Name: valid.Name, AddressLine1: valid.AddressLine1, City: valid.City, PostalCode: "bad", Country: "GB", Represent: true},
		{Name: valid.Name, AddressLine1: "", City: valid.City, PostalCode: valid.PostalCode, Country: "GB", Represent: true},
	} {
		if got := validateMemberContactProfile(&profile); got == "" {
			t.Fatalf("invalid representation accepted: %+v", profile)
		}
	}
}

func TestMemberProfileNormalisesUKConstituentCountries(t *testing.T) {
	for _, country := range []string{"GB", "GBR", "England", "Scotland", "Wales", "Northern Ireland"} {
		profile := memberContactProfile{Name: "Alex Owner", AddressLine1: "1 Example Road", City: "London", PostalCode: "SW1A 1AA", Country: country, Represent: true}
		if got := validateMemberContactProfile(&profile); got != "" {
			t.Errorf("%q rejected for representation: %s", country, got)
		}
		if profile.Country != "United Kingdom" {
			t.Errorf("%q normalised to %q, want United Kingdom", country, profile.Country)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "GBR", "England"); got != "GBR" {
		t.Fatalf("firstNonEmpty = %q, want GBR", got)
	}
}

func TestMemberProfileAllowsInternationalContactWithoutRepresentation(t *testing.T) {
	profile := memberContactProfile{Name: "Taylor Example", AddressLine1: "1 Rue Example", City: "Paris", PostalCode: "75001", Country: "France"}
	if got := validateMemberContactProfile(&profile); got != "" {
		t.Fatalf("international contact rejected: %s", got)
	}
}

func TestAddressSearchContext(t *testing.T) {
	for input, want := range map[string]string{"United Kingdom": "GBR", "US": "USA", "France": "FRA", "Australia": "AUS"} {
		if got := addressSearchContext(input); got != want {
			t.Errorf("context for %q = %q, want %q", input, got, want)
		}
	}
}

func TestRepresentationChoiceHistoryTracksMandateChanges(t *testing.T) {
	base := memberContactProfile{Name: "Alex Owner", AddressLine1: "1 Example Road", City: "London", PostalCode: "SW1A 1AA", Country: "GB", Represent: true, RepresentationVersion: "2026-10-02-v1"}
	if !representationChoiceChanged(memberContactProfile{}, base) {
		t.Fatal("initial opt-in must be recorded")
	}
	if representationChoiceChanged(base, base) {
		t.Fatal("unchanged profile must not produce another event")
	}
	changedAddress := base
	changedAddress.City = "Oxford"
	if !representationChoiceChanged(base, changedAddress) {
		t.Fatal("mandate address change must be recorded")
	}
	withdrawn := base
	withdrawn.Represent = false
	withdrawn.RepresentationVersion = ""
	if !representationChoiceChanged(base, withdrawn) {
		t.Fatal("withdrawal must be recorded")
	}
}

func TestAddressIDAcceptsInternationalSuggestionIDs(t *testing.T) {
	for _, id := range []string{"paf_23747771", "usps_V122200597|1600||17ND", "herewe_pap|uEXlTCwRpMmtSYizTlZnQ|en"} {
		if !addressID.MatchString(id) {
			t.Errorf("valid suggestion ID rejected: %q", id)
		}
	}
	for _, id := range []string{"", "../secret", "bad?key=value", "hello world"} {
		if addressID.MatchString(id) {
			t.Errorf("invalid suggestion ID accepted: %q", id)
		}
	}
}

func TestMemberProfileEndpointsRequireAuthentication(t *testing.T) {
	for _, tc := range []struct {
		method string
		url    string
		run    func(http.ResponseWriter, *http.Request)
	}{
		{http.MethodGet, "/api/member-profile", MemberProfile},
		{http.MethodPost, "/api/member-profile", MemberProfile},
		{http.MethodGet, "/api/member-address-lookup?query=London", MemberAddressLookup},
		{http.MethodPost, "/api/admin/member-verification", AdminMemberVerification},
	} {
		res := httptest.NewRecorder()
		tc.run(res, httptest.NewRequest(tc.method, tc.url, strings.NewReader(`{}`)))
		if res.Code != http.StatusUnauthorized {
			t.Errorf("%s %s returned %d: %s", tc.method, tc.url, res.Code, res.Body.String())
		}
		if res.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("%s did not set private no-store", tc.url)
		}
	}
}
