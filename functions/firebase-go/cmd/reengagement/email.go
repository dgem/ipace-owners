package main

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"strings"
	"time"
)

func reengagementPayload(person recipient, actionLink string, vehicleURL string, memberCount int, eligibleCount int) map[string]any {
	payload := map[string]any{
		"from": strings.TrimSpace(os.Getenv("RESEND_FROM")), "to": []string{person.Email},
		"subject": "Complete your I-PACE Owners registration",
		"html":    reengagementHTML(person, actionLink, vehicleURL, memberCount, eligibleCount), "text": reengagementText(person, actionLink, vehicleURL, memberCount, eligibleCount),
		"tags": []map[string]string{{"name": "category", "value": "join-reengagement"}},
	}
	if replyTo := strings.TrimSpace(os.Getenv("RESEND_REPLY_TO")); replyTo != "" {
		payload["reply_to"] = replyTo
	}
	return payload
}

func reengagementHTML(person recipient, actionLink string, vehicleURL string, memberCount int, eligibleCount int) string {
	name := html.EscapeString(firstName(person.Name))
	link := html.EscapeString(actionLink)
	vehicle := html.EscapeString(vehicleURL)
	image := html.EscapeString(strings.TrimRight(assetBaseURL(vehicleURL), "/") + "/images/ipace-hero.png")
	date := submittedDate(person.SubmittedAt)
	progress := html.EscapeString(registrationProgress(eligibleCount))
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Complete your I-PACE Owners registration</title></head>
<body style="margin:0;background:#f7f8fb;color:#111827;font-family:Arial,Helvetica,sans-serif;"><div style="display:none;max-height:0;overflow:hidden;">One quick step completes your I-PACE Owners registration.</div>
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f7f8fb;padding:24px 12px;"><tr><td align="center"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:640px;background:#fff;border-radius:20px;overflow:hidden;border:1px solid #dbe3ea;">
<tr><td style="background:#12324a;padding:24px 28px;color:#fff;"><div style="font-size:22px;font-weight:700;line-height:1.2;">I-PACE Owners</div><div style="font-size:14px;color:#c9d7e3;margin-top:4px;">Advocacy Group</div></td></tr>
<tr><td><img src="` + image + `" width="640" alt="Jaguar I-PACE" style="display:block;width:100%;height:auto;border:0;"></td></tr>
<tr><td style="padding:32px 28px;"><p style="margin:0 0 10px;font-size:16px;line-height:1.6;color:#374151;">Hello ` + name + `,</p><h1 style="margin:0 0 14px;color:#12324a;font-size:28px;line-height:1.2;">Please complete your registration</h1>
<p style="margin:0 0 18px;font-size:16px;line-height:1.6;color:#374151;">You asked to join the I-PACE Owners Advocacy Group on ` + html.EscapeString(date) + `. Your details were saved, but your secure sign-in was not completed.</p>
<div style="margin:0 0 22px;padding:18px 20px;background:#eef6f5;border-left:4px solid #0f766e;border-radius:10px;"><p style="margin:0 0 6px;font-size:16px;line-height:1.5;color:#12324a;font-weight:700;">` + fmt.Sprintf("%d", memberCount) + ` owners have joined us so far.</p><p style="margin:0;font-size:14px;line-height:1.55;color:#374151;">` + progress + ` That is a meaningful step toward our ambition of bringing together 1,000 I-PACE owners.</p></div>
<p style="margin:0 0 26px;font-size:16px;line-height:1.6;color:#374151;">Use this fresh, time-limited link to verify your account details and open your private member area.</p>
<p style="margin:0 0 24px;"><a href="` + link + `" style="display:inline-block;background:#0f766e;color:#fff;text-decoration:none;font-weight:700;border-radius:999px;padding:14px 22px;font-size:16px;">Verify my account details</a></p>
<p style="margin:0 0 8px;font-size:14px;line-height:1.5;color:#4b5563;">After signing in, you can strengthen the shared evidence by adding each I-PACE you own and any battery-health or service information you wish to contribute.</p>
<p style="margin:0 0 26px;"><a href="` + vehicle + `" style="display:inline-block;color:#0f766e;text-decoration:underline;font-weight:700;font-size:14px;">Add my I-PACE data</a></p>
<p style="margin:0 0 8px;font-size:12px;line-height:1.5;color:#6b7280;">If the main button does not work, copy this link into your browser:</p><p style="margin:0;word-break:break-all;font-size:12px;line-height:1.5;"><a href="` + link + `" style="color:#0f766e;">` + link + `</a></p></td></tr>
<tr><td style="background:#eef6f5;padding:20px 28px;color:#374151;font-size:13px;line-height:1.5;">You are receiving this because you submitted the Join form and agreed that we could contact you. If you no longer wish to hear from us, reply to this email and let us know.</td></tr>
</table></td></tr></table></body></html>`
}

func reengagementText(person recipient, actionLink string, vehicleURL string, memberCount int, eligibleCount int) string {
	return "Hello " + firstName(person.Name) + ",\n\n" +
		"Please complete your I-PACE Owners registration\n\n" +
		"You asked to join the I-PACE Owners Advocacy Group on " + submittedDate(person.SubmittedAt) + ". Your details were saved, but your secure sign-in was not completed.\n\n" +
		fmt.Sprintf("%d owners have joined us so far. %s That is a meaningful step toward our ambition of bringing together 1,000 I-PACE owners.\n\n", memberCount, registrationProgress(eligibleCount)) +
		"Verify your account details and sign in using this fresh, time-limited link:\n" + actionLink + "\n\n" +
		"After signing in, you can add each I-PACE you own and any battery-health or service information you wish to contribute:\n" + vehicleURL + "\n\n" +
		"You are receiving this because you submitted the Join form and agreed that we could contact you. If you no longer wish to hear from us, reply to this email and let us know.\n"
}

func registrationProgress(eligibleCount int) string {
	if eligibleCount > 125 {
		return "If everyone receiving this reminder completes verification, more than 125 additional members will have registered accounts."
	}
	return fmt.Sprintf("If everyone receiving this reminder completes verification, %d additional members will have registered accounts.", eligibleCount)
}

func firstName(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
}

func submittedDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2 January 2006")
}

func assetBaseURL(fallbackURL string) string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("RESEND_ASSET_BASE_URL")), "/"); value != "" {
		return value
	}
	if parsed, err := url.Parse(fallbackURL); err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" {
		return parsed.Scheme + "://" + parsed.Host
	}
	return "https://ipace-owners.org"
}
