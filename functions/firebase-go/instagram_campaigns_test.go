package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstagramPreviewRequiresAdmin(t *testing.T) {
	original := instagramCampaignAuthorize
	t.Cleanup(func() { instagramCampaignAuthorize = original })
	instagramCampaignAuthorize = func(context.Context, *http.Request) error { return context.Canceled }
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-preview", strings.NewReader(`{"mediaPath":"/reel.mp4","caption":"Hello","mediaReviewed":true}`))
	res := httptest.NewRecorder()
	AdminInstagramPreview(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestInstagramPreviewRejectsUnreviewedOrExternalMedia(t *testing.T) {
	if _, err := previewInstagramCampaign(instagramDraftRequest{MediaPath: "/reel.mp4", Caption: "Hello"}); err == nil {
		t.Fatal("expected unreviewed media to be rejected")
	}
	if _, err := previewInstagramCampaign(instagramDraftRequest{MediaPath: "https://attacker.example/reel.mp4", Caption: "Hello", MediaReviewed: true}); err == nil {
		t.Fatal("expected external media URL to be rejected")
	}
}

func TestInstagramPreviewIsDeterministicAndDoesNotExposeToken(t *testing.T) {
	t.Setenv("INSTAGRAM_MEDIA_BASE_URL", "https://ipace-owners.org")
	t.Setenv("INSTAGRAM_GRAPH_API_VERSION", "v99.0")
	t.Setenv("INSTAGRAM_USER_ID", "123")
	t.Setenv("INSTAGRAM_ACCESS_TOKEN", "never-return-this")
	input := instagramDraftRequest{MediaPath: "/reel.mp4", Caption: " Hello owners ", MediaReviewed: true}
	first, err := previewInstagramCampaign(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := previewInstagramCampaign(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.CampaignID != second.CampaignID || first.Confirmation != second.Confirmation {
		t.Fatalf("previews differ: %#v %#v", first, second)
	}
	if !first.Configured || first.MediaURL != "https://ipace-owners.org/reel.mp4" || strings.Contains(first.Confirmation, "never-return-this") {
		t.Fatalf("unexpected preview: %#v", first)
	}
}

func TestInstagramPublishRequiresExactPreviewConfirmation(t *testing.T) {
	originalAuth, originalPublish, originalReserve, originalComplete := instagramCampaignAuthorize, instagramCampaignPublish, instagramCampaignReserve, instagramCampaignComplete
	t.Cleanup(func() {
		instagramCampaignAuthorize = originalAuth
		instagramCampaignPublish = originalPublish
		instagramCampaignReserve = originalReserve
		instagramCampaignComplete = originalComplete
	})
	instagramCampaignAuthorize = func(context.Context, *http.Request) error { return nil }
	called := false
	instagramCampaignPublish = func(context.Context, instagramPreview) (string, error) { called = true; return "published", nil }
	instagramCampaignReserve = func(context.Context, instagramPreview) (string, error) { return "", nil }
	instagramCampaignComplete = func(context.Context, string, string, error) error { return nil }
	t.Setenv("INSTAGRAM_MEDIA_BASE_URL", "https://ipace-owners.org")
	t.Setenv("INSTAGRAM_GRAPH_API_VERSION", "v99.0")
	t.Setenv("INSTAGRAM_USER_ID", "123")
	t.Setenv("INSTAGRAM_ACCESS_TOKEN", "token")
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-publish", strings.NewReader(`{"campaignId":"wrong","mediaPath":"/reel.mp4","caption":"Hello","mediaReviewed":true,"confirmation":"wrong"}`))
	res := httptest.NewRecorder()
	AdminInstagramPublish(res, req)
	if res.Code != http.StatusConflict || called {
		t.Fatalf("status=%d called=%v body=%s", res.Code, called, res.Body.String())
	}
}

func TestInstagramPublishCallsProviderAfterExactConfirmation(t *testing.T) {
	originalAuth, originalPublish, originalReserve, originalComplete := instagramCampaignAuthorize, instagramCampaignPublish, instagramCampaignReserve, instagramCampaignComplete
	t.Cleanup(func() {
		instagramCampaignAuthorize = originalAuth
		instagramCampaignPublish = originalPublish
		instagramCampaignReserve = originalReserve
		instagramCampaignComplete = originalComplete
	})
	instagramCampaignAuthorize = func(context.Context, *http.Request) error { return nil }
	instagramCampaignPublish = func(_ context.Context, preview instagramPreview) (string, error) {
		if preview.MediaURL != "https://ipace-owners.org/reel.mp4" {
			t.Fatalf("media=%q", preview.MediaURL)
		}
		return "17890000000000000", nil
	}
	instagramCampaignReserve = func(context.Context, instagramPreview) (string, error) { return "", nil }
	instagramCampaignComplete = func(context.Context, string, string, error) error { return nil }
	t.Setenv("INSTAGRAM_MEDIA_BASE_URL", "https://ipace-owners.org")
	t.Setenv("INSTAGRAM_GRAPH_API_VERSION", "v99.0")
	t.Setenv("INSTAGRAM_USER_ID", "123")
	t.Setenv("INSTAGRAM_ACCESS_TOKEN", "token")
	preview, err := previewInstagramCampaign(instagramDraftRequest{MediaPath: "/reel.mp4", Caption: "Hello", MediaReviewed: true})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"campaignId":"` + preview.CampaignID + `","mediaPath":"/reel.mp4","caption":"Hello","mediaReviewed":true,"confirmation":"` + preview.Confirmation + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-publish", strings.NewReader(body))
	res := httptest.NewRecorder()
	AdminInstagramPublish(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "17890000000000000") {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
