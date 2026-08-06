package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var orderedListRE = regexp.MustCompile(`^\d+\.\s+`)

func renderMarkdownBody(body string, full bool, raw bool, width int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if raw {
		return rawBodyPreview(body, full)
	}

	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	truncated := false
	if !full && len(lines) > 90 {
		lines = lines[:90]
		truncated = true
	}

	var out strings.Builder
	var paragraph []string
	inCode := false

	writeLine := func(line string) {
		out.WriteString(line)
		out.WriteString("\n")
	}
	writeBlank := func() {
		text := out.String()
		if text == "" || strings.HasSuffix(text, "\n\n") {
			return
		}
		out.WriteString("\n")
	}
	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		if shouldPreserveShortLines(paragraph) {
			for _, source := range paragraph {
				for _, line := range wrapDisplay(source, width, "") {
					writeLine(line)
				}
			}
		} else {
			for _, line := range wrapDisplay(joinParagraphLines(paragraph), width, "") {
				writeLine(line)
			}
		}
		writeBlank()
		paragraph = nil
	}

	for _, sourceLine := range lines {
		line := strings.TrimRight(sourceLine, "\r")
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			flushParagraph()
			writeLine(truncateDisplay(trimmed, width))
			inCode = !inCode
			continue
		}
		if inCode {
			writeLine(truncateDisplay(strings.TrimRight(line, " "), width))
			continue
		}
		if trimmed == "" {
			flushParagraph()
			writeBlank()
			continue
		}
		if isStandaloneMarkdownLine(trimmed) {
			flushParagraph()
			for _, rendered := range renderMarkdownLine(trimmed, width) {
				writeLine(rendered)
			}
			writeBlank()
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	flushParagraph()
	if truncated {
		writeLine("...")
	}
	return strings.TrimRight(out.String(), "\n")
}

func rawBodyPreview(body string, full bool) string {
	if full {
		return body
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= 90 {
		return body
	}
	return strings.Join(lines[:90], "\n") + "\n..."
}

func isStandaloneMarkdownLine(line string) bool {
	return strings.HasPrefix(line, "#") ||
		strings.HasPrefix(line, "![") ||
		strings.HasPrefix(line, "- ") ||
		strings.HasPrefix(line, "* ") ||
		orderedListRE.MatchString(line) ||
		strings.HasPrefix(line, ">") ||
		strings.HasPrefix(line, "|")
}

func renderMarkdownLine(line string, width int) []string {
	switch {
	case strings.HasPrefix(line, "#"):
		level := 0
		for _, r := range line {
			if r != '#' {
				break
			}
			level++
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if title == "" {
			return []string{line}
		}
		prefix := strings.Repeat("#", level) + " "
		return wrapDisplay(prefix+title, width, "")
	case strings.HasPrefix(line, "!["):
		return renderImageLine(line, width)
	case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
		text := strings.TrimSpace(line[2:])
		return wrapWithIndents(text, width, "- ", "  ")
	case orderedListRE.MatchString(line):
		idx := strings.Index(line, ".")
		prefix := line[:idx+2]
		text := strings.TrimSpace(line[idx+2:])
		return wrapWithIndents(text, width, prefix, strings.Repeat(" ", displayWidth(prefix)))
	case strings.HasPrefix(line, ">"):
		text := strings.TrimSpace(strings.TrimLeft(line, ">"))
		return wrapDisplay(text, width, "> ")
	default:
		return wrapDisplay(line, width, "")
	}
}

func wrapWithIndents(text string, width int, firstIndent string, restIndent string) []string {
	lines := wrapDisplay(text, width, firstIndent)
	if len(lines) <= 1 {
		return lines
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, strings.TrimSpace(firstIndent)) {
			line = strings.TrimSpace(strings.TrimPrefix(line, strings.TrimSpace(firstIndent)))
		}
		lines[i] = restIndent + line
	}
	return lines
}

func renderImageLine(line string, width int) []string {
	matches := markdownImageRE.FindStringSubmatch(line)
	if len(matches) < 2 {
		return wrapDisplay(line, width, "")
	}
	alt := ""
	if end := strings.Index(line, "]"); strings.HasPrefix(line, "![") && end > 2 {
		alt = strings.TrimSpace(line[2:end])
	}
	label := "[image]"
	if alt != "" {
		label = fmt.Sprintf("[image: %s]", alt)
	}
	return wrapDisplay(label+" "+matches[1], width, "")
}

func joinParagraphLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i > 0 && needsParagraphSpace(sb.String(), line) {
			sb.WriteString(" ")
		}
		sb.WriteString(line)
	}
	return sb.String()
}

func shouldPreserveShortLines(lines []string) bool {
	if len(lines) < 2 || len(lines) > 8 {
		return false
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || displayWidth(line) > 24 || endsLikeSentenceFragment(line) {
			return false
		}
	}
	return true
}

func endsLikeSentenceFragment(line string) bool {
	switch lastRune(line) {
	case ',', '.', ';', ':', '、', '。', '，', '；', '：':
		return true
	default:
		return false
	}
}

func needsParagraphSpace(prev, next string) bool {
	prev = strings.TrimSpace(prev)
	next = strings.TrimSpace(next)
	if prev == "" || next == "" {
		return false
	}
	last := lastRune(prev)
	first := firstRune(next)
	if isWideRune(last) || isWideRune(first) {
		return false
	}
	if unicode.IsSpace(last) || unicode.IsSpace(first) {
		return false
	}
	return true
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func lastRune(s string) rune {
	var last rune
	for _, r := range s {
		last = r
	}
	return last
}
