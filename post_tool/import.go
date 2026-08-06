package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type importOptions struct {
	commonOptions
	File         string
	Title        string
	Description  string
	Tags         string
	Categories   string
	Series       string
	Date         string
	Author       string
	Draft        bool
	ImageFormat  string
	ImageQuality int
	MaxWidth     int
	DryRun       bool
}

type importPreview struct {
	ODT         string             `json:"odt"`
	Root        string             `json:"root,omitempty"`
	Meta        PostMeta           `json:"meta"`
	Output      string             `json:"output,omitempty"`
	ODTMeta     ODTMeta            `json:"odtMeta"`
	BlockMeta   ODTBlockMetaJSON   `json:"blockMeta,omitempty"`
	Images      []ImageWriteResult `json:"images"`
	Warnings    []string           `json:"warnings,omitempty"`
	BodyPreview string             `json:"bodyPreview,omitempty"`
}

type ODTBlockMetaJSON struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Date        string   `json:"date,omitempty"`
	Author      string   `json:"author,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	Series      []string `json:"series,omitempty"`
	Draft       *bool    `json:"draft,omitempty"`
	Cover       string   `json:"cover,omitempty"`
}

func runImport(args []string, legacy bool) error {
	opts := importOptions{
		Author:       "honemik",
		ImageFormat:  "webp",
		ImageQuality: 82,
		MaxWidth:     1600,
	}
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	addImportFlags(fs, &opts)
	seen, err := parseFlagSet(fs, args)
	if err != nil {
		return err
	}
	if opts.File == "" && fs.NArg() > 0 {
		opts.File = fs.Arg(0)
	}
	if opts.File == "" {
		if legacy {
			return fmt.Errorf("usage: odt2post -f <file.odt> [flags]")
		}
		return fmt.Errorf("usage: odt2post import <file.odt> [flags]")
	}

	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	opts.Root = root

	result, markdown, err := prepareImport(opts, seen)
	if err != nil {
		return err
	}
	if opts.DryRun {
		if opts.JSON {
			return writeJSON(result)
		}
		printImportPreview(result)
		return nil
	}

	if err := os.MkdirAll(postsDir(root), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(result.Output, []byte(markdown), 0644); err != nil {
		return err
	}
	if opts.JSON {
		return writeJSON(result)
	}

	fmt.Println("Imported post")
	fmt.Println("  Markdown:", result.Output)
	for _, image := range result.Images {
		if image.URL == "" {
			fmt.Println("  Image:", image.Source, "-", image.Warning)
			continue
		}
		if image.Reused {
			fmt.Println("  Image:", image.Source, "->", image.URL, "(reused)")
		} else {
			fmt.Println("  Image:", image.Source, "->", image.URL)
		}
		if image.Warning != "" {
			fmt.Println("    warning:", image.Warning)
		}
	}
	for _, warning := range result.Warnings {
		fmt.Println("  warning:", warning)
	}
	return nil
}

func runInspect(args []string) error {
	opts := importOptions{
		Author:       "honemik",
		ImageFormat:  "webp",
		ImageQuality: 82,
		MaxWidth:     1600,
		DryRun:       true,
	}
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	addImportFlags(fs, &opts)
	seen, err := parseFlagSet(fs, args)
	if err != nil {
		return err
	}
	if opts.File == "" && fs.NArg() > 0 {
		opts.File = fs.Arg(0)
	}
	if opts.File == "" {
		return fmt.Errorf("usage: odt2post inspect <file.odt> [flags]")
	}
	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	opts.Root = root

	result, _, err := prepareImport(opts, seen)
	if err != nil {
		return err
	}
	if opts.JSON {
		return writeJSON(result)
	}
	printImportPreview(result)
	return nil
}

func addImportFlags(fs *flag.FlagSet, opts *importOptions) {
	fs.StringVar(&opts.File, "f", "", "Path to the .odt file.")
	fs.StringVar(&opts.Title, "title", "", "Post title override.")
	fs.StringVar(&opts.Description, "desc", "", "Short description override.")
	fs.StringVar(&opts.Tags, "tags", "", "Comma-separated tags.")
	fs.StringVar(&opts.Categories, "categories", "", "Comma-separated categories.")
	fs.StringVar(&opts.Series, "series", "", "Comma-separated series names.")
	fs.StringVar(&opts.Date, "date", "", "Post date YYYY-MM-DD override.")
	fs.StringVar(&opts.Author, "author", "honemik", "Author name.")
	fs.BoolVar(&opts.Draft, "draft", false, "Mark as draft.")
	fs.StringVar(&opts.ImageFormat, "image-format", "webp", "Image format: webp or original.")
	fs.IntVar(&opts.ImageQuality, "image-quality", 82, "WebP quality.")
	fs.IntVar(&opts.MaxWidth, "max-width", 1600, "Maximum WebP width/height when ImageMagick is used.")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Preview without writing files.")
	addCommonFlags(fs, &opts.commonOptions)
}

func prepareImport(opts importOptions, seen map[string]bool) (importPreview, string, error) {
	odt, err := readODT(opts.File)
	if err != nil {
		return importPreview{}, "", err
	}
	converted := convertODTMarkdown(odt.Content)
	block, body, err := stripODTMetadataBlock(converted.Body)
	if err != nil {
		return importPreview{}, "", err
	}
	converted.Body = body
	converted.FirstHeading = firstMarkdownHeading(body)
	converted.FirstParagraph = firstMarkdownParagraph(body)

	meta, warnings, err := resolveImportMeta(opts, seen, odt, converted, block)
	if err != nil {
		return importPreview{}, "", err
	}
	slug := slugify(meta.Title)

	imageTokens, imageResults, err := writeImportedImages(odt, converted.Images, ImageWriteOptions{
		Root:     opts.Root,
		Date:     meta.Date,
		Slug:     slug,
		Format:   opts.ImageFormat,
		Quality:  opts.ImageQuality,
		MaxWidth: opts.MaxWidth,
		DryRun:   opts.DryRun,
	})
	if err != nil {
		return importPreview{}, "", err
	}
	body = replaceImageTokens(converted.Body, imageTokens)
	body = strings.TrimSpace(body)
	markdown := buildFrontmatter(meta) + "\n" + body + "\n"

	filename := fmt.Sprintf("%s-%s.md", meta.Date, slug)
	output := filepath.Join(postsDir(opts.Root), filename)

	result := importPreview{
		ODT:         odt.Path,
		Root:        opts.Root,
		Meta:        meta,
		Output:      output,
		ODTMeta:     odt.Meta,
		BlockMeta:   blockMetaJSON(block),
		Images:      imageResults,
		Warnings:    warnings,
		BodyPreview: shortText(body, 300),
	}
	return result, markdown, nil
}

func resolveImportMeta(opts importOptions, seen map[string]bool, odt *ODTFile, converted ODTMarkdown, block ODTBlockMeta) (PostMeta, []string, error) {
	var warnings []string

	title := firstNonEmpty(block.Title, odt.Meta.Title, converted.FirstHeading, filenameTitle(odt.Path))
	if seen["title"] {
		title = opts.Title
	}
	if strings.TrimSpace(title) == "" {
		return PostMeta{}, nil, fmt.Errorf("could not determine title")
	}

	description := firstNonEmpty(block.Description, shortText(converted.FirstParagraph, 120))
	if seen["desc"] {
		description = opts.Description
	}

	date := firstNonEmpty(block.Date, odt.Meta.Date, odt.FileMod)
	if seen["date"] {
		date = opts.Date
	}
	if date == "" {
		date = dateFromTime(time.Now())
		warnings = append(warnings, "date was missing from meta.xml and file stats; used today")
	}
	if err := validateDate(date); err != nil {
		return PostMeta{}, nil, err
	}

	author := firstNonEmpty(block.Author, "honemik")
	if seen["author"] {
		author = opts.Author
	}

	tags := block.Tags
	if seen["tags"] {
		tags = parseCSV(opts.Tags)
	}
	categories := block.Categories
	if seen["categories"] {
		categories = parseCSV(opts.Categories)
	}
	series := block.Series
	if seen["series"] {
		series = parseCSV(opts.Series)
	}

	draft := false
	if block.Draft != nil {
		draft = *block.Draft
	}
	if seen["draft"] {
		draft = opts.Draft
	}

	if odt.Meta.ModifiedRaw == "" {
		warnings = append(warnings, "meta.xml did not include dc:date; used ODT file modified date")
	}

	return PostMeta{
		Title:           title,
		Description:     description,
		Author:          author,
		Date:            date,
		Tags:            tags,
		Categories:      categories,
		Series:          series,
		Draft:           draft,
		ShowToc:         false,
		ShowBreadCrumbs: true,
	}, warnings, nil
}

func printImportPreview(result importPreview) {
	fmt.Println("Import preview")
	fmt.Println("  ODT:", result.ODT)
	fmt.Println("  Output:", result.Output)
	fmt.Println("  Title:", result.Meta.Title)
	fmt.Println("  Date:", result.Meta.Date)
	if result.Meta.Description != "" {
		fmt.Println("  Description:", result.Meta.Description)
	}
	if len(result.Meta.Tags) > 0 {
		fmt.Println("  Tags:", strings.Join(result.Meta.Tags, ", "))
	}
	if len(result.Meta.Categories) > 0 {
		fmt.Println("  Categories:", strings.Join(result.Meta.Categories, ", "))
	}
	fmt.Println("  Draft:", result.Meta.Draft)
	fmt.Println("  Images:", len(result.Images))
	for _, image := range result.Images {
		if image.URL != "" {
			fmt.Println("    ", image.Source, "->", image.URL)
		} else {
			fmt.Println("    ", image.Source, "-", image.Warning)
		}
	}
	for _, warning := range result.Warnings {
		fmt.Println("  Warning:", warning)
	}
}

func blockMetaJSON(block ODTBlockMeta) ODTBlockMetaJSON {
	return ODTBlockMetaJSON{
		Title:       block.Title,
		Description: block.Description,
		Date:        block.Date,
		Author:      block.Author,
		Tags:        block.Tags,
		Categories:  block.Categories,
		Series:      block.Series,
		Draft:       block.Draft,
		Cover:       block.Cover,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func filenameTitle(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	return strings.TrimSpace(base)
}
