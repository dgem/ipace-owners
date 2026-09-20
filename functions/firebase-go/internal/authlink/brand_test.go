package authlink

import (
	"net/url"
	"strings"
	"testing"
)

func TestBrandPreservesSignInQueryAcrossEnvironments(t *testing.T) {
	for _, tc := range []struct{ project, origin string }{
		{"ipace-owners-production", "https://ipace-owners.org"},
		{"ipace-owners-staging", "https://ipace-owners-staging--pr-123-example.web.app"},
	} {
		continuation := tc.origin + "/member/account/?authTrace=example&next=survey"
		query := "apiKey=public-test-key&mode=signIn&oobCode=dummy%2Bcode%2Ftest&continueUrl=" + url.QueryEscape(continuation) + "&lang=en"
		for _, host := range []string{tc.project + ".firebaseapp.com", tc.project + ".web.app", strings.TrimPrefix(tc.origin, "https://")} {
			got, err := Brand("https://"+host+"/__/auth/action?"+query, continuation, tc.project)
			if err != nil || got != tc.origin+"/auth/action?"+query {
				t.Fatalf("brand %s: %q %v", host, got, err)
			}
		}
	}
}

func TestBrandRejectsUnexpectedOrMalformedActionLinksWithoutLeakingCode(t *testing.T) {
	base := "https://ipace-owners-production.firebaseapp.com/__/auth/action"
	valid := "?apiKey=public-test-key&mode=signIn&oobCode=private-code"
	for _, link := range []string{
		"https://foreign.firebaseapp.com/__/auth/action" + valid,
		"http://ipace-owners-production.firebaseapp.com/__/auth/action" + valid,
		"https://user@ipace-owners-production.firebaseapp.com/__/auth/action" + valid,
		base + valid + "#fragment",
		base + valid + "&mode=resetPassword",
		base + "?apiKey=public-test-key&mode=resetPassword&oobCode=private-code",
		base + "?apiKey=public-test-key&mode=signIn",
		base + valid + "&oobCode=another",
		base + valid + "&bad=%ZZ",
	} {
		got, err := Brand(link, "https://ipace-owners.org/member/account/", "ipace-owners-production")
		if err == nil || got != "" || strings.Contains(err.Error(), "private-code") {
			t.Fatalf("unsafe link result: %q %v", got, err)
		}
	}
}

func TestBrandLeavesAlternateFirebaseFormatsAndLocalDevelopmentIntact(t *testing.T) {
	for _, tc := range []struct{ link, continuation string }{
		{"https://ipace-owners-production.firebaseapp.com/__/auth/links?link=example", "https://ipace-owners.org/member/account/"},
		{"https://ipace-owners-production.firebaseapp.com/__/auth/action?mode=signIn", "http://localhost:8080/member/account/"},
	} {
		got, err := Brand(tc.link, tc.continuation, "ipace-owners-production")
		if err != nil || got != tc.link {
			t.Fatalf("changed supported alternative: %q %v", got, err)
		}
	}
}
