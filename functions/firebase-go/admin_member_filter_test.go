package ipace

import "testing"

func TestMemberFilters(t *testing.T) {
	member := adminMemberSearchResult{BetaTester: true, WillingToHelp: true, Country: "NO"}
	for _, tc := range []struct {
		preference, country string
		want                bool
	}{
		{"tester", "NO", true}, {"volunteer", "", true}, {"press", "", false}, {"", "unset", false}, {"", "FR", false},
	} {
		if got := memberMatchesFilters(member, tc.preference, tc.country); got != tc.want {
			t.Errorf("filters %q %q: got %v", tc.preference, tc.country, got)
		}
	}
	if !memberMatchesFilters(adminMemberSearchResult{}, "", "unset") {
		t.Fatal("unset country must match")
	}
}
