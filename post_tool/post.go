package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type PostMeta struct {
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Author          string   `json:"author"`
	Date            string   `json:"date"`
	Tags            []string `json:"tags,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Series          []string `json:"series,omitempty"`
	Draft           bool     `json:"draft"`
	ShowToc         bool     `json:"ShowToc"`
	ShowBreadCrumbs bool     `json:"ShowBreadCrumbs"`
}

type PostInfo struct {
	Path          string   `json:"path"`
	Filename      string   `json:"filename"`
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	Author        string   `json:"author,omitempty"`
	Date          string   `json:"date,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Series        []string `json:"series,omitempty"`
	Draft         bool     `json:"draft"`
	Images        []string `json:"images"`
	MissingImages []string `json:"missingImages,omitempty"`
	Body          string   `json:"-"`
}

type PostScan struct {
	Root          string     `json:"root"`
	Posts         []PostInfo `json:"posts"`
	Referenced    []string   `json:"referencedImages"`
	MissingImages []string   `json:"missingImages"`
}

var markdownImageRE = regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
var dateSelectorRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func buildFrontmatter(meta PostMeta) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("title: " + quoteYAMLString(meta.Title) + "\n")
	if meta.Description != "" {
		sb.WriteString("description: " + quoteYAMLString(meta.Description) + "\n")
	}
	if meta.Author == "" {
		meta.Author = "honemik"
	}
	sb.WriteString("author: " + quoteYAMLString(meta.Author) + "\n")
	sb.WriteString("date: " + quoteYAMLString(meta.Date) + "\n")
	writeStringListYAML(&sb, "tags", meta.Tags)
	writeStringListYAML(&sb, "categories", meta.Categories)
	writeStringListYAML(&sb, "series", meta.Series)
	sb.WriteString(fmt.Sprintf("draft: %t\n", meta.Draft))
	sb.WriteString(fmt.Sprintf("ShowToc: %t\n", meta.ShowToc))
	sb.WriteString(fmt.Sprintf("ShowBreadCrumbs: %t\n", meta.ShowBreadCrumbs))
	sb.WriteString("---\n")
	return sb.String()
}

func writeStringListYAML(sb *strings.Builder, key string, values []string) {
	values = compactStrings(values)
	if len(values) == 0 {
		return
	}
	sb.WriteString(key + ":\n")
	for _, value := range values {
		sb.WriteString("  - " + quoteYAMLString(value) + "\n")
	}
}

func scanPosts(root string) (PostScan, error) {
	dir := postsDir(root)
	var posts []PostInfo
	referencedSet := map[string]bool{}
	missingSet := map[string]bool{}

	if _, err := os.Stat(dir); err != nil {
		return PostScan{Root: root}, nil
	}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		post, readErr := readPostInfo(root, path)
		if readErr != nil {
			return readErr
		}
		for _, img := range post.Images {
			referencedSet[img] = true
		}
		for _, img := range post.MissingImages {
			missingSet[img] = true
		}
		posts = append(posts, post)
		return nil
	})
	if err != nil {
		return PostScan{}, err
	}

	sort.Slice(posts, func(i, j int) bool {
		if posts[i].Date == posts[j].Date {
			return posts[i].Title < posts[j].Title
		}
		return posts[i].Date > posts[j].Date
	})

	return PostScan{
		Root:          root,
		Posts:         posts,
		Referenced:    sortedBoolKeys(referencedSet),
		MissingImages: sortedBoolKeys(missingSet),
	}, nil
}

func readPostInfo(root, path string) (PostInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PostInfo{}, err
	}
	fm, body := splitFrontmatter(string(data))
	meta := map[string]any{}
	if fm != "" {
		if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
			return PostInfo{}, fmt.Errorf("%s frontmatter parse error: %w", path, err)
		}
	}

	images := extractMarkdownImages(body)
	missing := missingImagePaths(root, images)
	filename := filepath.Base(path)
	slug := strings.TrimSuffix(filename, filepath.Ext(filename))

	return PostInfo{
		Path:          path,
		Filename:      filename,
		Slug:          slug,
		Title:         stringFromAny(meta["title"]),
		Description:   stringFromAny(meta["description"]),
		Author:        stringFromAny(meta["author"]),
		Date:          dateFromAny(meta["date"]),
		Tags:          stringListFromAny(meta["tags"]),
		Categories:    stringListFromAny(meta["categories"]),
		Series:        stringListFromAny(meta["series"]),
		Draft:         boolValue(meta["draft"]),
		Images:        images,
		MissingImages: missing,
		Body:          body,
	}, nil
}

func splitFrontmatter(content string) (string, string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}
	lines := strings.Split(content, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			fm := strings.Join(lines[1:i], "\n")
			body := strings.Join(lines[i+1:], "\n")
			return fm, strings.TrimLeft(body, "\n")
		}
	}
	return "", content
}

func extractMarkdownImages(body string) []string {
	matches := markdownImageRE.FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	out := []string{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		url := strings.TrimSpace(match[1])
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		out = append(out, url)
	}
	return out
}

func missingImagePaths(root string, images []string) []string {
	missing := []string{}
	for _, image := range images {
		path, ok := imageURLToPath(root, image)
		if !ok {
			continue
		}
		if !fileExists(path) {
			missing = append(missing, image)
		}
	}
	return missing
}

func imageURLToPath(root, url string) (string, bool) {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "/img/post/") {
		return "", false
	}
	name := strings.TrimPrefix(url, "/img/post/")
	if name == "" || strings.Contains(name, "..") {
		return "", false
	}
	return filepath.Join(postImagesDir(root), filepath.FromSlash(name)), true
}

func imagePathToURL(root, path string) (string, bool) {
	rel, err := filepath.Rel(postImagesDir(root), path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", false
	}
	return "/img/post/" + filepath.ToSlash(rel), true
}

func findPostMatches(posts []PostInfo, selector string) []PostInfo {
	selector = strings.TrimSpace(selector)
	lower := strings.ToLower(selector)
	if selector == "" {
		return nil
	}

	matchers := []func(PostInfo) bool{
		func(post PostInfo) bool { return strings.ToLower(post.Filename) == lower },
	}
	if dateSelectorRE.MatchString(selector) {
		matchers = append(matchers, func(post PostInfo) bool { return post.Date == selector })
	}
	matchers = append(matchers,
		func(post PostInfo) bool { return strings.ToLower(post.Slug) == lower },
		func(post PostInfo) bool { return strings.ToLower(post.Title) == lower },
		func(post PostInfo) bool { return strings.Contains(strings.ToLower(post.Filename), lower) },
		func(post PostInfo) bool { return strings.Contains(strings.ToLower(post.Slug), lower) },
		func(post PostInfo) bool { return strings.Contains(strings.ToLower(post.Title), lower) },
	)

	for _, matcher := range matchers {
		var matches []PostInfo
		for _, post := range posts {
			if matcher(post) {
				matches = append(matches, post)
			}
		}
		if len(matches) > 0 {
			return dedupePosts(matches)
		}
	}
	return nil
}

func requireOnePost(posts []PostInfo, selector string) (PostInfo, error) {
	matches := findPostMatches(posts, selector)
	if len(matches) == 0 {
		return PostInfo{}, fmt.Errorf("no post matched %q", selector)
	}
	if len(matches) > 1 {
		return PostInfo{}, fmt.Errorf("%q matched %d posts:\n%s\nUse a longer title, slug, or exact filename", selector, len(matches), formatPostMatches(matches))
	}
	return matches[0], nil
}

func dedupePosts(posts []PostInfo) []PostInfo {
	seen := map[string]bool{}
	var out []PostInfo
	for _, post := range posts {
		if seen[post.Path] {
			continue
		}
		seen[post.Path] = true
		out = append(out, post)
	}
	return out
}

func formatPostMatches(posts []PostInfo) string {
	rows := make([][]string, 0, len(posts))
	for _, post := range posts {
		rows = append(rows, []string{
			post.Date,
			shortDisplayText(post.Title, 34),
			post.Filename,
		})
	}
	return formatTable([]string{"DATE", "TITLE", "FILENAME"}, rows)
}

func setPostDraft(path string, draft bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	fm, body := splitFrontmatter(content)
	if fm == "" {
		return fmt.Errorf("%s has no YAML frontmatter", path)
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(fm), &node); err != nil {
		return err
	}
	if len(node.Content) == 0 || node.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%s frontmatter is not a mapping", path)
	}
	setYAMLScalar(node.Content[0], "draft", fmt.Sprintf("%t", draft), "!!bool")

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(node.Content[0]); err != nil {
		return err
	}
	enc.Close()

	next := "---\n" + strings.TrimSpace(buf.String()) + "\n---\n"
	if strings.TrimSpace(body) != "" {
		next += "\n" + strings.TrimLeft(body, "\n")
	}
	return os.WriteFile(path, []byte(next), 0644)
}

func setYAMLScalar(mapping *yaml.Node, key, value, tag string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value},
	)
}

func dateFromAny(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case time.Time:
		return dateFromTime(value)
	case string:
		if t, ok := parseODTDate(value); ok {
			return dateFromTime(t)
		}
		return value
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func boolValue(v any) bool {
	value, ok := boolFromAny(v)
	return ok && value
}

func sortedBoolKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
