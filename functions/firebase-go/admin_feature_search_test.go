package ipace

import (
	"strings"
	"testing"
)

func TestFeatureTesterSearchTermsMatchQueueQuery(t *testing.T) {
	if !strings.Contains(featureTesterSearchTerms, "feature") {
		t.Fatal("feature tester queue query must match opted-in members")
	}
}
