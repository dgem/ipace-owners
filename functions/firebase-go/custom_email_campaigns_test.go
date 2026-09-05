package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
)

type recordingCampaignDocumentWriter struct {
	data        any
	optionCount int
}

func (writer *recordingCampaignDocumentWriter) Set(_ context.Context, data any, options ...firestore.SetOption) (*firestore.WriteResult, error) {
	writer.data = data
	writer.optionCount = len(options)
	return nil, nil
}

func TestSaveCustomCampaignRecordReplacesStructWithoutMergeAll(t *testing.T) {
	record := customCampaignRecord{
		CampaignID: "campaign-1",
		Kind:       "all-members-drive",
		Name:       "Help us reach 1,000 members",
		Sent:       10,
		Remaining:  419,
	}
	writer := &recordingCampaignDocumentWriter{}
	if err := saveCustomCampaignRecord(context.Background(), writer, record); err != nil {
		t.Fatal(err)
	}
	saved, ok := writer.data.(customCampaignRecord)
	if !ok || saved != record {
		t.Fatalf("saved record = %#v", writer.data)
	}
	if writer.optionCount != 0 {
		t.Fatalf("Set received %d options; Firestore MergeAll cannot be used with struct data", writer.optionCount)
	}
}

func TestCustomCampaignSubstitutionsRenderEveryDocumentedValue(t *testing.T) {
	audience := customCampaignAudience{
		MembersJoined:            412,
		MembersVerified:          399,
		VehiclesRegisteredCount:  287,
		VehiclesSoHReadingsCount: 634,
		ServiceFaultRecordsCount: 91,
	}
	person := customCampaignRecipient{
		Name:       "Dr Jane Driver",
		JoinedAt:   time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC),
		VerifiedAt: time.Date(2026, time.July, 18, 9, 0, 0, 0, time.UTC),
		Vehicles: []customCampaignVehicle{{
			ID: "vehicle-1", Registration: "IPACE", ModelYear: "2021", SoHReadingsCount: 3,
		}},
	}
	data := customCampaignSubstitutionData(audience, person)
	template := strings.Join([]string{
		"{{membersJoined}}",
		"{{membersVerified}}",
		"{{memberFirstName}}",
		"{{memberLastName}}",
		"{{memberTittle}}",
		"{{memberTitle}}",
		"{{memberJoined}}",
		"{{memberVerified}}",
		"{{memberVehicles}}",
		"{{vehiclesRegisteredCount}}",
		"{{vehiclesSoHReadingsCount}}",
		"{{serviceFaultRecordsCount}}",
	}, "|")
	got, err := applyCustomCampaignSubstitutions(template, data)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"412", "399", "Jane", "Driver", "Dr", "17 July 2026", "18 July 2026",
		`"registration":"IPACE"`, `"sohReadingsCount":3`, "287", "634", "91",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("rendered substitutions missing %q: %s", expected, got)
		}
	}
	if strings.Contains(got, "{{") {
		t.Fatalf("rendered substitutions were left unresolved: %s", got)
	}
}

func TestValidateCustomCampaignDraftRejectsUnknownActionsAndUnsafeLinks(t *testing.T) {
	valid := customCampaignDraftRequest{
		Name:     "Member update",
		Subject:  "Hello {{memberFirstName}}",
		Markdown: "We have {{membersJoined}} members. [Visit us](https://ipace-owners.org/).",
	}
	if err := validateCustomCampaignDraft(valid); err != nil {
		t.Fatalf("valid draft rejected: %v", err)
	}
	cases := []customCampaignDraftRequest{
		{Name: "Test", Subject: "Hello", Markdown: "{{unknownValue}}"},
		{Name: "Test", Subject: "Hello", Markdown: "{{memberFirstName | printf \"%s\"}}"},
		{Name: "Test", Subject: "Hello", Markdown: "[Unsafe](javascript:alert(1))"},
		{Name: "Test", Subject: "Hello\nBcc: someone@example.com", Markdown: "Body"},
		{CampaignID: "nested/path", Name: "Test", Subject: "Hello", Markdown: "Body"},
		{SourceCampaignID: "../other", Name: "Test", Subject: "Hello", Markdown: "Body"},
	}
	for _, input := range cases {
		if err := validateCustomCampaignDraft(input); err == nil {
			t.Fatalf("unsafe draft accepted: %#v", input)
		}
	}
}

func TestRenderCustomCampaignEmailEscapesMemberDataAndUsesSharedBranding(t *testing.T) {
	record := customCampaignRecord{
		CampaignID: "campaign-1",
		Name:       "Test",
		Subject:    "Hello {{memberFirstName}}",
		Markdown:   "Hi {{memberTittle}} {{memberLastName}},\n\nVehicles: {{memberVehicles}}",
	}
	audience := customCampaignAudience{MembersJoined: 1, MembersVerified: 1}
	person := customCampaignRecipient{
		Name:     `Dr <Jane> Driver`,
		Vehicles: []customCampaignVehicle{{ID: `<vehicle>&`}},
	}
	subject, htmlBody, text, err := renderCustomCampaignEmail(record, audience, person)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Hello <Jane>" {
		t.Fatalf("subject = %q", subject)
	}
	for _, expected := range []string{"<!doctype html>", "/images/ipace-hero.png", `\u003cvehicle\u003e\u0026`} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("HTML missing %q: %s", expected, htmlBody)
		}
	}
	if strings.Contains(htmlBody, "<vehicle>") || !strings.Contains(text, `\u003cvehicle\u003e\u0026`) {
		t.Fatalf("member vehicle data was not correctly escaped by output context")
	}
}

func TestMemberValuesCannotInjectMarkdownLinks(t *testing.T) {
	record := customCampaignRecord{
		CampaignID: "campaign-1",
		Name:       "Test",
		Subject:    "Hello",
		Markdown:   "Hi {{memberFirstName}},",
	}
	person := customCampaignRecipient{Name: `[Jane](https://attacker.example/)`}
	_, htmlBody, text, err := renderCustomCampaignEmail(record, customCampaignAudience{}, person)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(htmlBody, `href="https://attacker.example/"`) {
		t.Fatalf("member value became an HTML link: %s", htmlBody)
	}
	if !strings.Contains(htmlBody, "[Jane](https://attacker.example/)") || !strings.Contains(text, "[Jane](https://attacker.example/)") {
		t.Fatalf("member value was not preserved as literal text")
	}
}

func TestCampaignMarkdownRendersHeadingsBoldItalicsListsAndLinks(t *testing.T) {
	markdown := "# Update\n\nWe are **growing** and *organised*.\n\n- One\n- [Two](https://ipace-owners.org/)"
	htmlBody := markdownToEmailHTML(markdown)
	for _, expected := range []string{
		"<h2", "<strong", "<em", "<ul", `href="https://ipace-owners.org/"`,
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("Markdown HTML missing %q: %s", expected, htmlBody)
		}
	}
	text := markdownToPlainText(markdown)
	if strings.Contains(text, "**") || strings.Contains(text, "*organised*") || strings.Contains(text, "# Update") {
		t.Fatalf("plain-text Markdown retained presentation delimiters: %q", text)
	}
}

func TestSplitCampaignMemberNameSupportsTitlesAndSingleNames(t *testing.T) {
	title, first, last := splitCampaignMemberName("Mrs. Jane Driver")
	if title != "Mrs." || first != "Jane" || last != "Driver" {
		t.Fatalf("unexpected titled name: %q %q %q", title, first, last)
	}
	title, first, last = splitCampaignMemberName("Prince")
	if title != "" || first != "Prince" || last != "" {
		t.Fatalf("unexpected single name: %q %q %q", title, first, last)
	}
}

func TestCustomCampaignVerifiedAtPreservesFirstObservedValueAndBackfillsLegacyUsers(t *testing.T) {
	stored := time.Date(2026, time.July, 18, 9, 0, 0, 0, time.UTC)
	got, inferred := customCampaignVerifiedAt(stored, &auth.UserMetadata{LastLogInTimestamp: time.Now().UnixMilli()})
	if !got.Equal(stored) || inferred {
		t.Fatalf("stored verification time was replaced: %v inferred=%v", got, inferred)
	}
	lastLogin := time.Date(2026, time.July, 20, 10, 0, 0, 0, time.UTC)
	got, inferred = customCampaignVerifiedAt(time.Time{}, &auth.UserMetadata{LastLogInTimestamp: lastLogin.UnixMilli()})
	if !got.Equal(lastLogin) || !inferred {
		t.Fatalf("legacy verification time was not inferred from Auth metadata: %v inferred=%v", got, inferred)
	}
}

func TestSurveyCampaignTargetsRegisteredMembersAndUsesHundredMessageBatches(t *testing.T) {
	if got := customCampaignAudienceScopeForKind(surveyCampaignKind); got != customCampaignRegisteredAudience {
		t.Fatalf("survey audience scope = %q", got)
	}
	if got := customCampaignAudienceScopeForKind(jlrContactCampaignKind); got != customCampaignConsentedAudience {
		t.Fatalf("JLR audience scope = %q", got)
	}
	if emailCampaignBatchSize != 100 {
		t.Fatalf("email batch size = %d, want 100", emailCampaignBatchSize)
	}
}

func TestCustomCampaignJoinedAtUsesAuthCreationTime(t *testing.T) {
	created := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	if got := customCampaignJoinedAt(&auth.UserMetadata{CreationTimestamp: created.UnixMilli()}); !got.Equal(created) {
		t.Fatalf("joined time = %v, want %v", got, created)
	}
	if got := customCampaignJoinedAt(nil); !got.IsZero() {
		t.Fatalf("nil metadata joined time = %v", got)
	}
}

func TestCustomCampaignStatusIsAuditableAcrossRunStates(t *testing.T) {
	if got := customCampaignStatus(10, 0); got != "draft" {
		t.Fatalf("draft status = %q", got)
	}
	if got := customCampaignStatus(10, 4); got != "sending" {
		t.Fatalf("sending status = %q", got)
	}
	if got := customCampaignStatus(10, 10); got != "complete" {
		t.Fatalf("complete status = %q", got)
	}
	legacy := inferredLegacyCampaign("member-referral-production-2026-07-22")
	if legacy.Kind != "member-referral" || legacy.Subject == "" || !strings.Contains(legacy.Markdown, "{{memberFirstName}}") {
		t.Fatalf("legacy campaign cannot be used as a rerun starting point: %#v", legacy)
	}
	reminder := inferredLegacyCampaign("join-account-verification-production-2026-07-22")
	if reminder.Kind != "join-reengagement" || reminder.Subject != "" || reminder.Markdown != "" {
		t.Fatalf("legacy registration reminder must stay on its specialised audience path: %#v", reminder)
	}
}

func TestSentCustomCampaignCanOnlyBePreviewedUnchanged(t *testing.T) {
	record := customCampaignRecord{
		Name: "Update", Subject: "Hello", Markdown: "Body", SourceCampaignID: "source-1",
	}
	same := customCampaignDraftRequest{
		Name: "Update", Subject: "Hello", Markdown: "Body", SourceCampaignID: "source-1",
	}
	if !sameCustomCampaignDraft(record, same) {
		t.Fatal("unchanged saved campaign was not recognised")
	}
	same.Markdown = "Edited body"
	if sameCustomCampaignDraft(record, same) {
		t.Fatal("edited saved campaign was treated as unchanged")
	}
}

func TestStartedStaticCampaignRetainsSavedCopyWhenTemplateChanges(t *testing.T) {
	for _, kind := range []string{jlrContactCampaignKind, surveyCampaignKind} {
		record := customCampaignRecord{
			Kind:         kind,
			Name:         "Saved campaign",
			Subject:      "Saved subject",
			Markdown:     "Saved body",
			HeroImage:    "/images/saved-hero.png",
			HeroImageAlt: "Saved hero",
			Sent:         2,
		}
		changed := customCampaignDraftRequest{
			Kind:         kind,
			CreateWithID: true,
			Name:         "Changed campaign",
			Subject:      "Changed subject",
			Markdown:     "Changed body",
			HeroImage:    "/images/changed-hero.png",
			HeroImageAlt: "Changed hero",
		}

		got, reused := staticCampaignInputForStartedRecord(changed, record)
		if !reused {
			t.Fatalf("started %s campaign did not reuse its saved copy", kind)
		}
		if !sameCustomCampaignDraft(record, got) {
			t.Fatalf("started %s campaign preview did not retain saved copy: %#v", kind, got)
		}
	}
}

func TestStaticCampaignKindsAreValidCampaignRecordsForSending(t *testing.T) {
	for _, kind := range []string{jlrContactCampaignKind, surveyCampaignKind} {
		if !isStaticCampaignKind(kind) {
			t.Fatalf("static campaign kind %q was not accepted for sending", kind)
		}
	}
}

func TestStartedNonStaticCampaignCannotReuseSavedCopy(t *testing.T) {
	record := customCampaignRecord{Kind: customCampaignKind, Name: "Saved", Sent: 1}
	input := customCampaignDraftRequest{Kind: customCampaignKind, Name: "Changed"}
	got, reused := staticCampaignInputForStartedRecord(input, record)
	if reused || got.Name != input.Name {
		t.Fatalf("non-static campaign unexpectedly reused saved copy: %#v", got)
	}
}

func TestAdminCustomCampaignPreviewRequiresAdmin(t *testing.T) {
	original := campaignAuthorize
	t.Cleanup(func() { campaignAuthorize = original })
	campaignAuthorize = func(context.Context, *http.Request) error { return context.Canceled }
	req := httptest.NewRequest(http.MethodPost, "/api/admin/custom-campaign-preview", strings.NewReader(`{}`))
	res := httptest.NewRecorder()
	AdminCustomCampaignPreview(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminCustomCampaignPreviewRejectsInvalidBodyBeforePreview(t *testing.T) {
	originalAuth, originalPreview := campaignAuthorize, customCampaignPreview
	t.Cleanup(func() {
		campaignAuthorize = originalAuth
		customCampaignPreview = originalPreview
	})
	campaignAuthorize = func(context.Context, *http.Request) error { return nil }
	called := false
	customCampaignPreview = func(context.Context, customCampaignDraftRequest) (customCampaignPreviewResponse, error) {
		called = true
		return customCampaignPreviewResponse{}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/custom-campaign-preview", strings.NewReader(`{"name":`))
	res := httptest.NewRecorder()
	AdminCustomCampaignPreview(res, req)
	if res.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v", res.Code, called)
	}
}

func TestAdminEmailCampaignHistoryRequiresAdmin(t *testing.T) {
	original := campaignAuthorize
	t.Cleanup(func() { campaignAuthorize = original })
	campaignAuthorize = func(context.Context, *http.Request) error { return context.Canceled }
	req := httptest.NewRequest(http.MethodPost, "/api/admin/email-campaign-history", strings.NewReader(`{}`))
	res := httptest.NewRecorder()
	AdminEmailCampaignHistory(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminCustomCampaignSendRejectsInvalidBodyBeforeSending(t *testing.T) {
	originalAuth, originalSend := campaignAuthorize, customCampaignSend
	t.Cleanup(func() {
		campaignAuthorize = originalAuth
		customCampaignSend = originalSend
	})
	campaignAuthorize = func(context.Context, *http.Request) error { return nil }
	called := false
	customCampaignSend = func(context.Context, customCampaignSendRequest) (customCampaignPreviewResponse, error) {
		called = true
		return customCampaignPreviewResponse{}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/custom-campaign-send", strings.NewReader(`{"campaignId":`))
	res := httptest.NewRecorder()
	AdminCustomCampaignSend(res, req)
	if res.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v", res.Code, called)
	}
}
