// Package authlink gives generated Firebase sign-in links a first-party entry URL.
package authlink

import (
	"fmt"
	"net/url"
)

// Brand preserves Firebase's query byte-for-byte. Hosting redirects /auth/action
// to its own reserved Firebase handler; no application server sees the action code.
// continueURL must be the server-selected, authorized email continuation URL.
func Brand(actionLink, continueURL, projectID string) (string, error) {
	action, err := url.Parse(actionLink)
	if err != nil || action.Scheme != "https" || action.Host == "" || action.User != nil || action.Fragment != "" {
		return "", fmt.Errorf("invalid Firebase action link")
	}
	continuation, err := url.Parse(continueURL)
	if err != nil || continuation.Host == "" || continuation.User != nil {
		return "", fmt.Errorf("invalid sign-in continuation URL")
	}
	// Keep local development and Firebase's alternate mobile-link formats intact.
	if continuation.Scheme != "https" || action.Path != "/__/auth/action" {
		return actionLink, nil
	}
	if projectID == "" || (action.Host != projectID+".firebaseapp.com" && action.Host != projectID+".web.app" && action.Host != continuation.Host) {
		return "", fmt.Errorf("unexpected Firebase action host")
	}
	query, err := url.ParseQuery(action.RawQuery)
	if err != nil || query.Get("mode") != "signIn" || len(query["mode"]) != 1 || query.Get("oobCode") == "" || len(query["oobCode"]) != 1 || query.Get("apiKey") == "" || len(query["apiKey"]) != 1 {
		return "", fmt.Errorf("invalid Firebase sign-in parameters")
	}
	action.Host = continuation.Host
	action.Path = "/auth/action"
	action.RawPath = ""
	return action.String(), nil
}
