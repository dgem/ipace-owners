package ipace

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstagramPreviewSavesNamedDraft(t *testing.T) {
	originalAuth, originalSave := instagramCampaignAuthorize, instagramCampaignSaveDraft
	t.Cleanup(func() {
		instagramCampaignAuthorize = originalAuth
		instagramCampaignSaveDraft = originalSave
	})
	instagramCampaignAuthorize = func(context.Context, *http.Request) error { return nil }
	instagramCampaignSaveDraft = func(_ context.Context, input instagramDraftRequest, preview instagramPreview) (instagramCampaignRecord, error) {
		if input.Name != "Launch follow-up" {
			t.Fatalf("name=%q", input.Name)
		}
		return instagramCampaignRecord{
			CampaignID: "instagram-aaaaaaaaaa", Name: preview.Name,
			MediaPath: preview.MediaPath, Caption: preview.Caption, Status: "draft",
		}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-preview", strings.NewReader(`{"name":"Launch follow-up","mediaPath":"/reel.mp4","caption":"Hello","mediaReviewed":true}`))
	res := httptest.NewRecorder()
	AdminInstagramPreview(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"campaignId":"instagram-aaaaaaaaaa"`) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

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

func TestInstagramPreviewPreservesInvalidTokenStatus(t *testing.T) {
	original := instagramCampaignAuthorize
	t.Cleanup(func() { instagramCampaignAuthorize = original })
	instagramCampaignAuthorize = func(context.Context, *http.Request) error {
		return authorizationFailure(http.StatusUnauthorized, "Sign in required", context.Canceled)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-preview", strings.NewReader(`{}`))
	res := httptest.NewRecorder()
	AdminInstagramPreview(res, req)
	if res.Code != http.StatusUnauthorized {
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
	originalLoad := instagramCampaignLoad
	t.Cleanup(func() {
		instagramCampaignAuthorize = originalAuth
		instagramCampaignPublish = originalPublish
		instagramCampaignReserve = originalReserve
		instagramCampaignComplete = originalComplete
		instagramCampaignLoad = originalLoad
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
	instagramCampaignLoad = func(context.Context, string) (instagramCampaignRecord, error) {
		return instagramCampaignRecord{
			CampaignID: preview.CampaignID, Name: preview.Name,
			MediaPath: preview.MediaPath, Caption: preview.Caption, Status: "draft",
		}, nil
	}
	body := `{"campaignId":"` + preview.CampaignID + `","mediaPath":"/reel.mp4","caption":"Hello","mediaReviewed":true,"confirmation":"` + preview.Confirmation + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-publish", strings.NewReader(body))
	res := httptest.NewRecorder()
	AdminInstagramPublish(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "17890000000000000") {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestFetchInstagramInsightsParsesSupportedMetrics(t *testing.T) {
	originalClient := instagramHTTPClient
	t.Cleanup(func() { instagramHTTPClient = originalClient })
	t.Setenv("INSTAGRAM_MEDIA_BASE_URL", "https://ipace-owners.org")
	t.Setenv("INSTAGRAM_GRAPH_API_VERSION", "v99.0")
	t.Setenv("INSTAGRAM_USER_ID", "123")
	t.Setenv("INSTAGRAM_ACCESS_TOKEN", "token")
	instagramHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer token" || req.URL.Query().Get("metric") == "" {
			t.Fatalf("request=%s auth=%q", req.URL, req.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{"data":[
				{"name":"views","values":[{"value":1200}]},
				{"name":"reach","values":[{"value":900}]},
				{"name":"total_interactions","total_value":{"value":73}}
			]}`)),
		}, nil
	})}
	result, err := fetchInstagramInsights(context.Background(), "17890000000000000")
	if err != nil || !result.Available || result.Views != 1200 || result.Reach != 900 || result.TotalInteractions != 73 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
