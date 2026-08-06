package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func stringFromAny(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func stringListFromAny(v any) []string {
	switch value := v.(type) {
	case nil:
		return nil
	case []string:
		return compactStrings(value)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s := stringFromAny(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		return parseCSV(value)
	default:
		if s := stringFromAny(value); s != "" {
			return []string{s}
		}
		return nil
	}
}

func boolFromAny(v any) (bool, bool) {
	switch value := v.(type) {
	case nil:
		return false, false
	case bool:
		return value, true
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "yes", "1":
			return true, true
		case "false", "no", "0":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func compactStrings(values []string) []string {
	var out []string
	for _, value := range values {
		if s := strings.TrimSpace(value); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	prevSep := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevSep = false
			continue
		}
		if !prevSep {
			sb.WriteRune('-')
			prevSep = true
		}
	}
	result := strings.Trim(sb.String(), "-")
	if result == "" {
		result = "post"
	}
	runes := []rune(result)
	if len(runes) > 60 {
		runes = runes[:60]
	}
	return string(runes)
}

func shortText(s string, max int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func shortDisplayText(s string, maxWidth int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	return truncateDisplay(s, maxWidth)
}

func validateDate(date string) error {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date %q, expected YYYY-MM-DD", date)
	}
	return nil
}

func dateFromTime(t time.Time) string {
	return t.Format("2006-01-02")
}

func parseODTDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func jsonMarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ensureInside(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return err
	}
	if rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && !filepath.IsAbs(rel)) {
		return nil
	}
	return fmt.Errorf("%s is outside %s", targetAbs, rootAbs)
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func quoteYAMLString(s string) string {
	return strconv.Quote(s)
}

func printTable(headers []string, rows [][]string) {
	fmt.Print(formatTable(headers, rows))
}

func formatTable(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = displayWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				continue
			}
			if w := displayWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var out strings.Builder
	writeRow := func(row []string) {
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			padding := widths[i] - displayWidth(cell)
			if i == len(headers)-1 {
				out.WriteString(cell)
			} else {
				out.WriteString(cell + strings.Repeat(" ", padding+2))
			}
		}
		out.WriteString("\n")
	}

	writeRow(headers)
	separators := make([]string, len(headers))
	for i, w := range widths {
		separators[i] = strings.Repeat("-", w)
	}
	writeRow(separators)
	for _, row := range rows {
		writeRow(row)
	}
	return out.String()
}

func terminalWidth() int {
	if value := strings.TrimSpace(os.Getenv("COLUMNS")); value != "" {
		if n, err := strconv.Atoi(value); err == nil && n >= 60 && n <= 240 {
			return n
		}
	}
	return 100
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		switch {
		case r == '\t':
			width += 4
		case r == '\n' || r == '\r':
		case unicode.IsControl(r):
		case isCombiningRune(r):
		case isWideRune(r):
			width += 2
		default:
			width++
		}
	}
	return width
}

func padRightDisplay(s string, width int) string {
	padding := width - displayWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func truncateDisplay(s string, maxWidth int) string {
	if maxWidth <= 0 || displayWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	limit := maxWidth - 3
	width := 0
	var sb strings.Builder
	for _, r := range s {
		rw := runeWidth(r)
		if width+rw > limit {
			break
		}
		sb.WriteRune(r)
		width += rw
	}
	return strings.TrimSpace(sb.String()) + "..."
}

func wrapDisplay(s string, width int, indent string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if width <= displayWidth(indent)+10 {
		width = displayWidth(indent) + 40
	}
	available := width - displayWidth(indent)
	var lines []string
	current := strings.Builder{}
	currentWidth := 0
	lastBreakByte := -1

	flush := func() {
		line := strings.TrimSpace(current.String())
		if line != "" {
			lines = append(lines, indent+line)
		}
		current.Reset()
		currentWidth = 0
		lastBreakByte = -1
	}

	for _, r := range s {
		if r == '\n' {
			flush()
			continue
		}
		rw := runeWidth(r)
		current.WriteRune(r)
		currentWidth += rw
		if unicode.IsSpace(r) || isBreakPunctuation(r) {
			lastBreakByte = current.Len()
		}
		if currentWidth <= available {
			continue
		}
		if lastBreakByte > 0 {
			full := current.String()
			head := strings.TrimSpace(full[:lastBreakByte])
			tail := strings.TrimSpace(full[lastBreakByte:])
			if head != "" {
				lines = append(lines, indent+head)
			}
			current.Reset()
			current.WriteString(tail)
			currentWidth = displayWidth(tail)
			lastBreakByte = -1
			continue
		}
		full := []rune(current.String())
		var head strings.Builder
		var tail strings.Builder
		w := 0
		split := 0
		for i, rr := range full {
			rrw := runeWidth(rr)
			if w+rrw > available {
				split = i
				break
			}
			head.WriteRune(rr)
			w += rrw
		}
		for _, rr := range full[split:] {
			tail.WriteRune(rr)
		}
		if strings.TrimSpace(head.String()) != "" {
			lines = append(lines, indent+strings.TrimSpace(head.String()))
		}
		current.Reset()
		current.WriteString(strings.TrimSpace(tail.String()))
		currentWidth = displayWidth(current.String())
	}
	flush()
	return lines
}

func runeWidth(r rune) int {
	switch {
	case r == '\t':
		return 4
	case r == '\n' || r == '\r' || unicode.IsControl(r) || isCombiningRune(r):
		return 0
	case isWideRune(r):
		return 2
	default:
		return 1
	}
}

func isCombiningRune(r rune) bool {
	return (r >= 0x0300 && r <= 0x036F) ||
		(r >= 0x1AB0 && r <= 0x1AFF) ||
		(r >= 0x1DC0 && r <= 0x1DFF) ||
		(r >= 0x20D0 && r <= 0x20FF) ||
		(r >= 0xFE20 && r <= 0xFE2F)
}

func isWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115F) ||
		(r >= 0x2329 && r <= 0x232A) ||
		(r >= 0x2E80 && r <= 0xA4CF) ||
		(r >= 0xAC00 && r <= 0xD7A3) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) ||
		(r >= 0xFFE0 && r <= 0xFFE6)
}

func isBreakPunctuation(r rune) bool {
	switch r {
	case '、', '。', '，', '；', '：', '！', '？', '）', '】', '」', '』':
		return true
	default:
		return false
	}
}

func horizontalRule(width int) string {
	if width < 20 {
		width = 20
	}
	if width > 100 {
		width = 100
	}
	return strings.Repeat("-", width)
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
