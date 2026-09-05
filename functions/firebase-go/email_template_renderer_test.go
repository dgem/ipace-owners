package ipace

import (
	"strings"
	"testing"
)

func TestMarkdownToEmailHTMLRendersEmphasisAndEscapesContent(t *testing.T) {
	rendered := markdownToEmailHTML(
		`Before *important <unsafe>* and [*linked text*](https://example.com/?a=1&b=2). [Email us](mailto:contact@example.com).`,
	)

	for _, expected := range []string{
		`<em style="font-style:italic;">important &lt;unsafe&gt;</em>`,
		`href="https://example.com/?a=1&amp;b=2"`,
		`<em style="font-style:italic;">linked text</em>`,
		`href="mailto:contact@example.com"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("markdownToEmailHTML() = %q, want fragment %q", rendered, expected)
		}
	}
	if strings.Contains(rendered, "<unsafe>") {
		t.Fatalf("markdownToEmailHTML() returned unescaped HTML: %q", rendered)
	}
}

func TestMarkdownToPlainTextRemovesEmphasisMarkers(t *testing.T) {
	rendered := markdownToPlainText(
		"*Your account verification link will expire in 24 hours.*\n\n" +
			"*If you didn't ask to join, please* [let us know](mailto:contact@example.com).\n\n" +
			"[Help](https://example.com/help)",
	)

	expected := "Your account verification link will expire in 24 hours.\n\n" +
		"If you didn't ask to join, please let us know: mailto:contact@example.com.\n\n" +
		"Help: https://example.com/help\n"
	if rendered != expected {
		t.Fatalf("markdownToPlainText() = %q, want %q", rendered, expected)
	}
}

func TestMarkdownActionLinkRendersAnEmailSafePillAndPlainTextFallback(t *testing.T) {
	markdown := "[Have your say — tell us what you need](https://ipace-owners.org/member/survey-response/?id=survey_example){.button}"
	html := markdownToEmailHTML(markdown)
	for _, expected := range []string{
		`href="https://ipace-owners.org/member/survey-response/?id=survey_example"`,
		`border-radius:999px`,
		`background:#0f766e`,
		`Have your say — tell us what you need`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("markdownToEmailHTML() = %q, want fragment %q", html, expected)
		}
	}

	plain := markdownToPlainText(markdown)
	if strings.Contains(plain, "{.button}") || !strings.Contains(plain, "Have your say — tell us what you need: https://ipace-owners.org/member/survey-response/?id=survey_example") {
		t.Fatalf("markdownToPlainText() = %q, want a clean action-link fallback", plain)
	}
}
