package ipace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"firebase.google.com/go/v4/auth"
)

var (
	generateFirebaseEmailSignInLink = generateFirebaseEmailSignInLinkRequest
	sendResendMagicLinkEmail        = sendResendMagicLinkEmailRequest
)

func resendEmailConfigured() bool {
	return strings.TrimSpace(os.Getenv("RESEND_API_KEY")) != "" && strings.TrimSpace(os.Getenv("RESEND_FROM")) != ""
}

func firebaseEmailActionCodeSettings(continueURL string, linkDomain string) *auth.ActionCodeSettings {
	settings := &auth.ActionCodeSettings{
		URL:             continueURL,
		HandleCodeInApp: true,
	}
	if linkDomain != "" {
		settings.LinkDomain = linkDomain
	}
	return settings
}

func generateFirebaseEmailSignInLinkRequest(ctx context.Context, email string, continueURL string, linkDomain string) (string, error) {
	client, err := firebaseAuth(ctx)
	if err != nil {
		return "", err
	}
	return client.EmailSignInLink(ctx, email, firebaseEmailActionCodeSettings(continueURL, linkDomain))
}

func sendResendMagicLinkEmailRequest(ctx context.Context, email string, actionLink string, continueURL string) error {
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}
	payload := resendMagicLinkPayload(email, actionLink, continueURL)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(res.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("resend response read failed: %w", readErr)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("resend returned %d with %d response bytes", res.StatusCode, len(responseBody))
	}
	return nil
}

func resendMagicLinkPayload(email string, actionLink string, continueURL string) map[string]any {
	payload := map[string]any{
		"from":    strings.TrimSpace(os.Getenv("RESEND_FROM")),
		"to":      []string{email},
		"subject": "Your secure sign-in link for I-PACE Owners",
		"html":    magicLinkEmailHTML(actionLink, emailAssetBaseURL(continueURL)),
		"text":    magicLinkEmailText(actionLink),
		"tags": []map[string]string{
			{"name": "category", "value": "magic-link"},
		},
	}
	if replyTo := strings.TrimSpace(os.Getenv("RESEND_REPLY_TO")); replyTo != "" {
		payload["reply_to"] = replyTo
	}
	return payload
}

func emailAssetBaseURL(continueURL string) string {
	parsed, err := url.Parse(continueURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return "https://ipace-owners.org"
	}
	host := parsed.Hostname()
	if strings.HasSuffix(host, ".web.app") && strings.Contains(host, "--pr-") {
		return parsed.Scheme + "://" + parsed.Host
	}
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("RESEND_ASSET_BASE_URL")), "/"); value != "" {
		return value
	}
	if host == "localhost" || strings.HasSuffix(host, ".web.app") || strings.HasSuffix(host, ".firebaseapp.com") {
		return "https://ipace-owners.org"
	}
	return parsed.Scheme + "://" + parsed.Host
}

func magicLinkEmailHTML(actionLink string, assetBaseURL string) string {
	body := strings.Join([]string{
		`<p style="margin:0 0 18px;font-size:16px;line-height:1.6;color:#374151;">Use this link to complete registration or sign in to your I-PACE Owners member area.</p>`,
		`<p style="margin:0 0 26px;font-size:16px;line-height:1.6;color:#374151;">The link is time-limited and should only be used by you.</p>`,
	}, "")
	return brandedEmailHTML(brandedEmailMessage{
		DocumentTitle:      "Your secure sign-in link for I-PACE Owners",
		Preheader:          "Use this secure link to sign in to I-PACE Owners.",
		Heading:            "Your secure sign-in link",
		BodyHTML:           body,
		PrimaryActionLabel: "Sign in securely",
		PrimaryActionURL:   actionLink,
		FallbackURL:        actionLink,
		FallbackLabel:      "If the button does not work, copy and paste this URL into your browser:",
		FooterNote:         "If you did not request this email, you can safely ignore it.",
		AssetBaseURL:       assetBaseURL,
	})
}

type brandedEmailMessage struct {
	DocumentTitle      string
	Preheader          string
	Heading            string
	BodyHTML           string
	PrimaryActionLabel string
	PrimaryActionURL   string
	FallbackURL        string
	FallbackLabel      string
	FooterNote         string
	AssetBaseURL       string
	SupplementHTML     string
}

func brandedEmailHTML(message brandedEmailMessage) string {
	title := html.EscapeString(strings.TrimSpace(message.DocumentTitle))
	if title == "" {
		title = "I-PACE Owners"
	}
	preheader := html.EscapeString(strings.TrimSpace(message.Preheader))
	heading := html.EscapeString(strings.TrimSpace(message.Heading))
	imageURL := html.EscapeString(strings.TrimRight(strings.TrimSpace(message.AssetBaseURL), "/") + "/images/ipace-hero.png")
	body := strings.TrimSpace(message.BodyHTML)
	if body == "" {
		body = `<p style="margin:0;font-size:16px;line-height:1.6;color:#374151;">Thank you for being part of I-PACE Owners.</p>`
	}

	primaryAction := ""
	if strings.TrimSpace(message.PrimaryActionURL) != "" && strings.TrimSpace(message.PrimaryActionLabel) != "" {
		primaryAction = `<p style="margin:0 0 28px;"><a href="` + html.EscapeString(strings.TrimSpace(message.PrimaryActionURL)) + `" style="display:inline-block;background:#0f766e;color:#ffffff;text-decoration:none;font-weight:700;border-radius:999px;padding:14px 22px;font-size:16px;">` + html.EscapeString(strings.TrimSpace(message.PrimaryActionLabel)) + `</a></p>`
	}

	fallback := ""
	if strings.TrimSpace(message.FallbackURL) != "" {
		label := strings.TrimSpace(message.FallbackLabel)
		if label == "" {
			label = "If the button does not work, copy and paste this URL into your browser:"
		}
		escapedURL := html.EscapeString(strings.TrimSpace(message.FallbackURL))
		fallback = `<p style="margin:0 0 12px;font-size:13px;line-height:1.5;color:#4b5563;">` + html.EscapeString(label) + `</p><p style="margin:0;word-break:break-all;font-size:13px;line-height:1.5;color:#4b5563;"><a href="` + escapedURL + `" style="color:#0f766e;">` + escapedURL + `</a></p>`
	}

	supplement := strings.TrimSpace(message.SupplementHTML)
	if supplement != "" {
		supplement = `<div style="margin:22px 0 0;">` + supplement + `</div>`
	}

	footer := html.EscapeString(strings.TrimSpace(message.FooterNote))
	if footer == "" {
		footer = "If you did not request this email, you can safely ignore it."
	}

	return `<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width,initial-scale=1">
		<title>` + title + `</title>
	</head>
	<body style="margin:0;background:#f7f8fb;color:#111827;font-family:Arial,Helvetica,sans-serif;">
		<div style="display:none;max-height:0;overflow:hidden;">` + preheader + `</div>
		<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f7f8fb;padding:24px 12px;">
			<tr>
				<td align="center">
					<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:640px;background:#ffffff;border-radius:20px;overflow:hidden;border:1px solid #dbe3ea;">
						<tr>
							<td style="background:#12324a;padding:24px 28px;color:#ffffff;">
								<div style="font-size:22px;font-weight:700;line-height:1.2;">I-PACE Owners</div>
								<div style="font-size:14px;color:#c9d7e3;margin-top:4px;">Advocacy Group</div>
							</td>
						</tr>
						<tr>
							<td>
								<img src="` + imageURL + `" width="640" alt="Jaguar I-PACE" style="display:block;width:100%;height:auto;border:0;">
							</td>
						</tr>
						<tr>
							<td style="padding:32px 28px;">
								<h1 style="margin:0 0 14px;color:#12324a;font-size:28px;line-height:1.2;">` + heading + `</h1>
								` + body + primaryAction + supplement + fallback + `
							</td>
						</tr>
						<tr>
							<td style="background:#eef6f5;padding:20px 28px;color:#374151;font-size:14px;line-height:1.5;">
								` + footer + `
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>
	</body>
</html>`
}

func magicLinkEmailText(actionLink string) string {
	return "Your secure sign-in link for I-PACE Owners\n\n" +
		"Use this link to complete registration or sign in to your I-PACE Owners member area:\n\n" +
		actionLink + "\n\n" +
		"The link is time-limited and should only be used by you.\n\n" +
		"If you did not request this email, you can safely ignore it.\n"
}
