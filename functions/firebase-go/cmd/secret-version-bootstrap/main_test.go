package main

import (
	"errors"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestMissingOrDestroyed(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"missing", &googleapi.Error{Code: 404}, true},
		{"destroyed latest", &googleapi.Error{Code: 400, Message: "Secret Version is in DESTROYED state."}, true},
		{"disabled latest", &googleapi.Error{Code: 400, Message: "Secret Version is disabled."}, false},
		{"permission denied", &googleapi.Error{Code: 403}, false},
		{"other error", errors.New("network unavailable"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := missingOrDestroyed(tc.err); got != tc.want {
				t.Fatalf("missingOrDestroyed() = %v, want %v", got, tc.want)
			}
		})
	}
}
