package main

import (
	"archive/zip"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// ─── ODT XML structures ───────────────────────────────────────────────────────

type ODTDocument struct {
	Body ODTBody `xml:"body"`
}

type ODTBody struct {
	Text ODTText `xml:"text"`
}

type ODTText struct {
	Paragraphs []ODTNode `xml:",any"`
}

// ODTNode is a generic node that can represent any element in the ODT XML tree.
type ODTNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []ODTNode  `xml:",any"`
	CharData string     `xml:",chardata"`
}

// ─── Conversion state ─────────────────────────────────────────────────────────

type Converter struct {
	imageFiles    map[string][]byte // "Pictures/foo.jpg" -> raw bytes
	writtenImages map[string]string // ODT internal path -> output filename
	imageCounter  int
	dateStr       string // YYYY-MM-DD prefix for image filenames
	imgURLBase    string // URL prefix in markdown, e.g. /img/post
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	var (
		odtPath     = flag.String("f", "", "Path to the .odt file (required)")
		title       = flag.String("title", "", "Post title (required)")
		description = flag.String("desc", "", "Short description")
		tagsStr     = flag.String("tags", "", `Comma-separated tags, e.g. "Game,Life"`)
		dateStr     = flag.String("date", "", "Post date YYYY-MM-DD (defaults to today)")
		author      = flag.String("author", "honemik", "Author name")
		draft       = flag.Bool("draft", false, "Mark as draft")
		blogRoot    = flag.String("root", "..", "Path to the blog root directory")
	)
	flag.Parse()

	if *odtPath == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "Usage: odt2post -f <file.odt> -title \"My Title\" [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *dateStr == "" {
		*dateStr = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", *dateStr); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid date format %q, expected YYYY-MM-DD\n", *dateStr)
		os.Exit(1)
	}

	blogRoot_ := filepath.Clean(*blogRoot)
	contentDir := filepath.Join(blogRoot_, "src", "content", "posts")
	imgDir := filepath.Join(blogRoot_, "public", "img", "post")

	conv := &Converter{
		imageFiles:    make(map[string][]byte),
		writtenImages: make(map[string]string),
		dateStr:       *dateStr,
		imgURLBase:    "/img/post",
	}

	// ── Open & read ODT (zip archive) ───────────────────────────────────────
	zr, err := zip.OpenReader(*odtPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open ODT file: %v\n", err)
		os.Exit(1)
	}
	defer zr.Close()

	var contentXML []byte
	for _, f := range zr.File {
		rc, openErr := f.Open()
		if openErr != nil {
			continue
		}
		data, _ := io.ReadAll(rc)
		rc.Close()

		if f.Name == "content.xml" {
			contentXML = data
		} else if isImagePath(f.Name) {
			conv.imageFiles[f.Name] = data
		}
	}

	if contentXML == nil {
		fmt.Fprintln(os.Stderr, "content.xml not found inside the ODT file")
		os.Exit(1)
	}

	// ── Parse XML ───────────────────────────────────────────────────────────
	var doc ODTDocument
	if err := xml.Unmarshal(contentXML, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "XML parse error: %v\n", err)
		os.Exit(1)
	}

	// ── Convert body to Markdown ─────────────────────────────────────────────
	var bodySB strings.Builder
	for _, node := range doc.Body.Text.Paragraphs {
		bodySB.WriteString(conv.nodeToMarkdown(node))
	}
	body := strings.TrimSpace(bodySB.String())

	// ── Assemble file ────────────────────────────────────────────────────────
	tags := parseTags(*tagsStr)
	fm := buildFrontmatter(*title, *description, *author, *dateStr, tags, *draft)
	markdown := fm + "\n" + body + "\n"

	// ── Write markdown ───────────────────────────────────────────────────────
	slug := slugify(*title)
	mdFilename := fmt.Sprintf("%s-%s.md", *dateStr, slug)
	mdPath := filepath.Join(contentDir, mdFilename)

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create content dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write markdown: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Markdown → %s\n", mdPath)

	// ── Write extracted images ───────────────────────────────────────────────
	if len(conv.writtenImages) > 0 {
		if err := os.MkdirAll(imgDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot create image dir: %v\n", err)
			os.Exit(1)
		}
		for src, outName := range conv.writtenImages {
			dest := filepath.Join(imgDir, outName)
			imgData, exists := conv.imageFiles[src]
			if !exists {
				fmt.Fprintf(os.Stderr, "⚠ Image not found in ODT: %s\n", src)
				continue
			}
			if err := os.WriteFile(dest, imgData, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Cannot write image %s: %v\n", dest, err)
			} else {
				fmt.Printf("✓ Image    → %s\n", dest)
			}
		}
	}
}

// ─── Node → Markdown ─────────────────────────────────────────────────────────

func (c *Converter) nodeToMarkdown(n ODTNode) string {
	switch n.XMLName.Local {
	case "h": // text:h
		level := attrInt(n.Attrs, "outline-level", 2)
		prefix := strings.Repeat("#", level)
		return prefix + " " + collectText(n) + "\n\n"

	case "p": // text:p
		if img := c.extractImage(n); img != "" {
			return img + "\n\n"
		}
		text := collectText(n)
		if strings.TrimSpace(text) == "" {
			return "\n"
		}
		return text + "\n\n"

	case "list": // text:list
		return c.listToMarkdown(n, false) + "\n"

	case "frame": // draw:frame at top level
		if img := c.extractImage(n); img != "" {
			return img + "\n\n"
		}
	}
	return ""
}

func (c *Converter) listToMarkdown(n ODTNode, ordered bool) string {
	var sb strings.Builder
	for _, child := range n.Children {
		if child.XMLName.Local != "list-item" {
			continue
		}
		for _, item := range child.Children {
			switch item.XMLName.Local {
			case "p":
				bullet := "- "
				if ordered {
					bullet = "1. "
				}
				sb.WriteString(bullet + collectText(item) + "\n")
			case "list":
				inner := c.listToMarkdown(item, ordered)
				for _, line := range strings.Split(strings.TrimRight(inner, "\n"), "\n") {
					if line != "" {
						sb.WriteString("  " + line + "\n")
					}
				}
			}
		}
	}
	return sb.String()
}

// extractImage searches a subtree for a draw:image and returns the markdown string.
func (c *Converter) extractImage(n ODTNode) string {
	for _, child := range n.Children {
		switch child.XMLName.Local {
		case "frame":
			if img := c.extractImage(child); img != "" {
				return img
			}
		case "image":
			href := attrVal(child.Attrs, "href")
			if href == "" {
				continue
			}
			href = strings.TrimPrefix(href, "./")

			outName, seen := c.writtenImages[href]
			if !seen {
				c.imageCounter++
				ext := filepath.Ext(href)
				if ext == "" {
					ext = ".jpg"
				}
				outName = fmt.Sprintf("%s-%02d%s", c.dateStr, c.imageCounter, ext)
				c.writtenImages[href] = outName
			}
			return fmt.Sprintf("![](%s/%s)", c.imgURLBase, outName)
		default:
			if img := c.extractImage(child); img != "" {
				return img
			}
		}
	}
	return ""
}

// ─── Text extraction ──────────────────────────────────────────────────────────

func collectText(n ODTNode) string {
	var sb strings.Builder
	collectTextInto(n, &sb)
	return sb.String()
}

func collectTextInto(n ODTNode, sb *strings.Builder) {
	sb.WriteString(n.CharData)
	for _, child := range n.Children {
		switch child.XMLName.Local {
		case "line-break":
			sb.WriteString("  \n")
		case "tab":
			sb.WriteString("\t")
		case "s": // text:s repeated spaces
			count := attrInt(child.Attrs, "c", 1)
			sb.WriteString(strings.Repeat(" ", count))
		default:
			collectTextInto(child, sb)
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// isImagePath returns true for any zip entry that looks like an image,
// regardless of which subdirectory it lives in (Pictures/, media/, etc.).
func isImagePath(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif":
		return true
	}
	return false
}

func attrVal(attrs []xml.Attr, localName string) string {
	for _, a := range attrs {
		if a.Name.Local == localName {
			return a.Value
		}
	}
	return ""
}

func attrInt(attrs []xml.Attr, localName string, def int) int {
	v := attrVal(attrs, localName)
	if v == "" {
		return def
	}
	var n int
	fmt.Sscan(v, &n)
	if n == 0 {
		return def
	}
	return n
}

func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func buildFrontmatter(title, desc, author, date string, tags []string, draft bool) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %q\n", title))
	if desc != "" {
		sb.WriteString(fmt.Sprintf("description: %q\n", desc))
	}
	sb.WriteString(fmt.Sprintf("author: %q\n", author))
	sb.WriteString(fmt.Sprintf("date: %q\n", date))
	if len(tags) > 0 {
		sb.WriteString("tags:\n")
		for _, t := range tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	if draft {
		sb.WriteString("draft: true\n")
	} else {
		sb.WriteString("draft: false\n")
	}
	sb.WriteString("ShowToc: false\n")
	sb.WriteString("ShowBreadCrumbs: true\n")
	sb.WriteString("---\n")
	return sb.String()
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	prevSep := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevSep = false
		} else if !prevSep {
			sb.WriteRune('-')
			prevSep = true
		}
	}
	result := strings.Trim(sb.String(), "-")
	if result == "" {
		result = "post"
	}
	runes := []rune(result)
	if len(runes) > 50 {
		runes = runes[:50]
	}
	return string(runes)
}
