package ipace

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"regexp"
	"strings"
	texttemplate "text/template"
)

//go:embed email-templates/*.md.tmpl
var emailTemplateFiles embed.FS

var (
	emailTemplates     = texttemplate.Must(texttemplate.New("emails").ParseFS(emailTemplateFiles, "email-templates/*.md.tmpl"))
	markdownLinkRegexp = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func renderEmailMarkdownTemplate(name string, data any) (string, error) {
	var output bytes.Buffer
	if err := emailTemplates.ExecuteTemplate(&output, name, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
}

func markdownToEmailHTML(markdown string) string {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ""
	}

	lines := strings.Split(trimmed, "\n")
	paragraph := []string{}
	list := []string{}
	sections := make([]string, 0, 8)

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		text := renderInlineMarkdown(strings.Join(paragraph, " "))
		sections = append(sections, `<p style="margin:0 0 16px;font-size:16px;line-height:1.6;color:#374151;">`+text+`</p>`)
		paragraph = paragraph[:0]
	}
	flushList := func() {
		if len(list) == 0 {
			return
		}
		items := make([]string, 0, len(list))
		for _, item := range list {
			items = append(items, `<li style="margin:0 0 10px;">`+renderInlineMarkdown(item)+`</li>`)
		}
		sections = append(sections, `<ul style="margin:0 0 16px 20px;padding:0;font-size:16px;line-height:1.6;color:#374151;">`+strings.Join(items, "")+`</ul>`)
		list = list[:0]
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			flushParagraph()
			flushList()
			continue
		}
		if strings.HasPrefix(line, "- ") {
			flushParagraph()
			list = append(list, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			continue
		}
		flushList()
		paragraph = append(paragraph, line)
	}
	flushParagraph()
	flushList()

	return strings.Join(sections, "")
}

func markdownToPlainText(markdown string) string {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ""
	}
	converted := markdownLinkRegexp.ReplaceAllString(trimmed, "$1: $2")
	return converted + "\n"
}

func renderInlineMarkdown(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	matches := markdownLinkRegexp.FindAllStringSubmatchIndex(trimmed, -1)
	if len(matches) == 0 {
		return html.EscapeString(trimmed)
	}

	var builder strings.Builder
	cursor := 0
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}
		builder.WriteString(html.EscapeString(trimmed[cursor:match[0]]))
		label := strings.TrimSpace(trimmed[match[2]:match[3]])
		url := strings.TrimSpace(trimmed[match[4]:match[5]])
		builder.WriteString(`<a href="` + html.EscapeString(url) + `" style="color:#0f766e;">` + html.EscapeString(label) + `</a>`)
		cursor = match[1]
	}
	builder.WriteString(html.EscapeString(trimmed[cursor:]))
	return builder.String()
}

func renderCampaignTemplate(name string, data any) (text string, htmlBody string, err error) {
	markdown, err := renderEmailMarkdownTemplate(name, data)
	if err != nil {
		return "", "", fmt.Errorf("render template %s: %w", name, err)
	}
	return markdownToPlainText(markdown), markdownToEmailHTML(markdown), nil
}

func mustRenderCampaignTemplate(name string, data any) (text string, htmlBody string) {
	text, htmlBody, err := renderCampaignTemplate(name, data)
	if err != nil {
		panic(err)
	}
	return text, htmlBody
}
