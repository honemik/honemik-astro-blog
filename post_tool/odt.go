package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ODTDocument struct {
	Body ODTBody `xml:"body"`
}

type ODTBody struct {
	Text ODTText `xml:"text"`
}

type ODTText struct {
	Nodes []ODTNode `xml:",any"`
}

type ODTNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []ODTNode  `xml:",any"`
	CharData string     `xml:",chardata"`
}

type ODTFile struct {
	Path       string
	Content    ODTDocument
	Meta       ODTMeta
	Images     map[string][]byte
	FileMod    string
	RawContent []byte
	RawMeta    []byte
}

type ODTMeta struct {
	Title       string `json:"title,omitempty"`
	ModifiedRaw string `json:"modifiedRaw,omitempty"`
	Date        string `json:"date,omitempty"`
}

type ODTImageRef struct {
	Token string `json:"token"`
	Src   string `json:"src"`
	Alt   string `json:"alt,omitempty"`
}

type ODTMarkdown struct {
	Body           string        `json:"body,omitempty"`
	Images         []ODTImageRef `json:"images"`
	FirstHeading   string        `json:"firstHeading,omitempty"`
	FirstParagraph string        `json:"firstParagraph,omitempty"`
}

type ODTBlockMeta struct {
	Title       string
	Description string
	Date        string
	Author      string
	Tags        []string
	Categories  []string
	Series      []string
	Draft       *bool
	Cover       string
}

func readODT(path string) (*ODTFile, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(abs)
	if err != nil {
		return nil, fmt.Errorf("cannot open ODT file: %w", err)
	}
	defer zr.Close()

	out := &ODTFile{
		Path:   abs,
		Images: make(map[string][]byte),
	}

	for _, f := range zr.File {
		rc, openErr := f.Open()
		if openErr != nil {
			continue
		}
		data, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			return nil, readErr
		}

		switch {
		case f.Name == "content.xml":
			out.RawContent = data
		case f.Name == "meta.xml":
			out.RawMeta = data
		case isImagePath(f.Name):
			out.Images[normalizeODTPath(f.Name)] = data
		}
	}

	if len(out.RawContent) == 0 {
		return nil, fmt.Errorf("content.xml not found inside %s", abs)
	}
	if err := xml.Unmarshal(out.RawContent, &out.Content); err != nil {
		return nil, fmt.Errorf("content.xml parse error: %w", err)
	}
	if len(out.RawMeta) > 0 {
		out.Meta = parseODTMeta(out.RawMeta)
	}
	if info, statErr := os.Stat(abs); statErr == nil {
		out.FileMod = dateFromTime(info.ModTime())
	}
	if out.Meta.Date == "" {
		out.Meta.Date = out.FileMod
	}
	return out, nil
}

func parseODTMeta(data []byte) ODTMeta {
	var node ODTNode
	if err := xml.Unmarshal(data, &node); err != nil {
		return ODTMeta{}
	}

	meta := ODTMeta{
		Title:       strings.TrimSpace(firstLocalText(node, "title")),
		ModifiedRaw: strings.TrimSpace(firstLocalText(node, "date")),
	}
	if t, ok := parseODTDate(meta.ModifiedRaw); ok {
		meta.Date = dateFromTime(t)
	}
	return meta
}

func firstLocalText(n ODTNode, local string) string {
	if n.XMLName.Local == local {
		return collectText(n)
	}
	for _, child := range n.Children {
		if found := firstLocalText(child, local); strings.TrimSpace(found) != "" {
			return found
		}
	}
	return ""
}

type markdownConverter struct {
	images      []ODTImageRef
	imageTokens map[string]string
}

func convertODTMarkdown(doc ODTDocument) ODTMarkdown {
	conv := &markdownConverter{
		imageTokens: make(map[string]string),
	}
	var sb strings.Builder
	for _, node := range doc.Body.Text.Nodes {
		sb.WriteString(conv.nodeToMarkdown(node))
	}
	body := strings.TrimSpace(sb.String())
	return ODTMarkdown{
		Body:           body,
		Images:         conv.images,
		FirstHeading:   firstMarkdownHeading(body),
		FirstParagraph: firstMarkdownParagraph(body),
	}
}

func (c *markdownConverter) nodeToMarkdown(n ODTNode) string {
	switch n.XMLName.Local {
	case "h":
		level := attrInt(n.Attrs, "outline-level", 2)
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		text := strings.TrimSpace(collectText(n))
		if text == "" {
			return ""
		}
		return strings.Repeat("#", level) + " " + text + "\n\n"
	case "p":
		return c.paragraphToMarkdown(n)
	case "list":
		return c.listToMarkdown(n, false, 0) + "\n"
	case "frame":
		return c.imagesMarkdown(n)
	}
	return ""
}

func (c *markdownConverter) paragraphToMarkdown(n ODTNode) string {
	text := strings.TrimSpace(collectText(n))
	images := c.imagesMarkdown(n)
	switch {
	case text == "" && images == "":
		return "\n"
	case text == "":
		return images
	case images == "":
		return text + "\n\n"
	default:
		return text + "\n\n" + images
	}
}

func (c *markdownConverter) listToMarkdown(n ODTNode, ordered bool, depth int) string {
	var sb strings.Builder
	index := 1
	for _, child := range n.Children {
		if child.XMLName.Local != "list-item" {
			continue
		}
		for _, item := range child.Children {
			switch item.XMLName.Local {
			case "p":
				bullet := "- "
				if ordered {
					bullet = fmt.Sprintf("%d. ", index)
				}
				text := strings.TrimSpace(collectText(item))
				if text != "" {
					sb.WriteString(strings.Repeat("  ", depth) + bullet + text + "\n")
				}
				if images := c.imagesMarkdown(item); images != "" {
					for _, line := range strings.Split(strings.TrimSpace(images), "\n") {
						sb.WriteString(strings.Repeat("  ", depth+1) + line + "\n")
					}
				}
				index++
			case "list":
				sb.WriteString(c.listToMarkdown(item, ordered, depth+1))
			}
		}
	}
	return sb.String()
}

func (c *markdownConverter) imagesMarkdown(n ODTNode) string {
	images := c.collectImages(n, "")
	if len(images) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, img := range images {
		alt := strings.ReplaceAll(img.Alt, "]", "")
		sb.WriteString(fmt.Sprintf("![%s](%s)\n\n", alt, img.Token))
	}
	return sb.String()
}

func (c *markdownConverter) collectImages(n ODTNode, inheritedAlt string) []ODTImageRef {
	alt := inheritedAlt
	if n.XMLName.Local == "frame" {
		for _, name := range []string{"title", "name"} {
			if v := strings.TrimSpace(attrVal(n.Attrs, name)); v != "" {
				alt = v
				break
			}
		}
	}

	var out []ODTImageRef
	for _, child := range n.Children {
		switch child.XMLName.Local {
		case "image":
			href := normalizeODTPath(attrVal(child.Attrs, "href"))
			if href == "" {
				continue
			}
			token, seen := c.imageTokens[href]
			if !seen {
				token = fmt.Sprintf("{{ODT_IMAGE_%03d}}", len(c.images)+1)
				c.imageTokens[href] = token
				ref := ODTImageRef{Token: token, Src: href, Alt: alt}
				c.images = append(c.images, ref)
				out = append(out, ref)
				continue
			}
			out = append(out, ODTImageRef{Token: token, Src: href, Alt: alt})
		default:
			out = append(out, c.collectImages(child, alt)...)
		}
	}
	return out
}

func stripODTMetadataBlock(markdown string) (ODTBlockMeta, string, error) {
	lines := strings.Split(markdown, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.TrimSpace(line) != "---" {
			return ODTBlockMeta{}, markdown, nil
		}
		start = i
		break
	}
	if start == -1 {
		return ODTBlockMeta{}, markdown, nil
	}

	end := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return ODTBlockMeta{}, markdown, fmt.Errorf("ODT metadata block starts with --- but has no closing ---")
	}

	raw := strings.Join(lines[start+1:end], "\n")
	block, err := parseBlockMeta(raw)
	if err != nil {
		return ODTBlockMeta{}, markdown, err
	}

	next := append([]string{}, lines[:start]...)
	next = append(next, lines[end+1:]...)
	return block, strings.TrimSpace(strings.Join(next, "\n")), nil
}

func parseBlockMeta(raw string) (ODTBlockMeta, error) {
	if strings.TrimSpace(raw) == "" {
		return ODTBlockMeta{}, nil
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(raw), &data); err != nil {
		return ODTBlockMeta{}, fmt.Errorf("invalid ODT metadata block YAML: %w", err)
	}
	var out ODTBlockMeta
	out.Title = stringFromAny(data["title"])
	out.Description = stringFromAny(data["description"])
	out.Date = stringFromAny(data["date"])
	out.Author = stringFromAny(data["author"])
	out.Tags = stringListFromAny(data["tags"])
	out.Categories = stringListFromAny(data["categories"])
	out.Series = stringListFromAny(data["series"])
	out.Cover = stringFromAny(data["cover"])
	if v, ok := boolFromAny(data["draft"]); ok {
		out.Draft = &v
	}
	return out, nil
}

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
		case "s":
			count := attrInt(child.Attrs, "c", 1)
			sb.WriteString(strings.Repeat(" ", count))
		default:
			collectTextInto(child, sb)
		}
	}
}

func firstMarkdownHeading(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func firstMarkdownParagraph(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "---" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "![") {
			continue
		}
		return line
	}
	return ""
}

func isImagePath(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tiff", ".tif":
		return true
	default:
		return false
	}
}

func normalizeODTPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.ReplaceAll(path, "\\", "/")
	return path
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
