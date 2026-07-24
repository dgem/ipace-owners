package ipace

import (
	"strings"
	"testing"
)

func TestMarkdownToEmailHTMLRendersEmphasisAndEscapesContent(t *testing.T) {
	rendered := markdownToEmailHTML(
		`Before *important <unsafe>* and [*linked text*](https://example.com/?a=1&b=2).`,
	)

	for _, expected := range []string{
		`<em style="font-style:italic;">important &lt;unsafe&gt;</em>`,
		`href="https://example.com/?a=1&amp;b=2"`,
		`<em style="font-style:italic;">linked text</em>`,
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
			"*If you didn't ask to join, let us know at contact@example.com.*\n\n" +
			"[Help](https://example.com/help)",
	)

	expected := "Your account verification link will expire in 24 hours.\n\n" +
		"If you didn't ask to join, let us know at contact@example.com.\n\n" +
		"Help: https://example.com/help\n"
	if rendered != expected {
		t.Fatalf("markdownToPlainText() = %q, want %q", rendered, expected)
	}
}
