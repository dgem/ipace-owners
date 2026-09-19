package ipace

import (
	"fmt"
	"regexp"
	"strings"
)

var surveyReminderCounterRE = regexp.MustCompile(`(?m)^\[\[(responses|members|vehicles|soh|service):([0-9]+)\]\]$`)
var surveyReminderCounterLabels = map[string]string{
	"responses": "Survey responses", "members": "Owners joined", "vehicles": "Cars recorded",
	"soh": "Battery-health readings", "service": "Service & fault records",
}

func surveyReminderPlainCounters(markdown string) string {
	return surveyReminderCounterRE.ReplaceAllStringFunc(markdown, func(line string) string {
		match := surveyReminderCounterRE.FindStringSubmatch(line)
		return match[2] + " " + strings.ToLower(surveyReminderCounterLabels[match[1]])
	})
}

func surveyReminderEmailHTML(markdown string) string {
	matches := surveyReminderCounterRE.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return markdownToEmailHTML(markdown)
	}
	first := surveyReminderCounterRE.FindStringIndex(markdown)
	before := markdownToEmailHTML(markdown[:first[0]])
	after := markdownToEmailHTML(surveyReminderCounterRE.ReplaceAllString(markdown[first[0]:], ""))
	// Raster background works in major email clients. Values and labels remain
	// ordinary text, visible even when an email client blocks background images.
	counters := `<div style="background:#12324a;border-radius:12px;padding:16px 4px;text-align:center;margin:0 0 24px;">`
	for _, match := range matches {
		counters += fmt.Sprintf(`<div style="display:inline-block;vertical-align:top;width:170px;max-width:48%%;margin:0 0 12px;"><table role="presentation" cellpadding="0" cellspacing="0" style="width:132px;margin:0 auto;"><tr><td align="center" valign="middle" height="132" background="https://ipace-owners.org/images/racing-laurel-email.png" style="height:132px;background-image:url('https://ipace-owners.org/images/racing-laurel-email.png');background-size:132px 132px;background-repeat:no-repeat;background-position:center;color:#f2ca6b;font-size:27px;font-weight:bold;">%s</td></tr></table><p style="margin:3px 5px;color:#ffffff;font-size:12px;line-height:1.4;font-weight:bold;">%s</p></div>`, match[2], surveyReminderCounterLabels[match[1]])
	}
	return before + counters + "</div>" + after
}
