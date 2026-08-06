package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runList(args []string) error {
	var opts commonOptions
	var draftsOnly, liveOnly, all, showImages, showMissing bool
	var writeIndex string
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.BoolVar(&draftsOnly, "drafts", false, "Show draft posts only.")
	fs.BoolVar(&liveOnly, "live", false, "Show live posts only.")
	fs.BoolVar(&all, "all", false, "Show all posts.")
	fs.BoolVar(&showImages, "images", false, "Show image paths under each post.")
	fs.BoolVar(&showMissing, "missing-images", false, "Show posts with missing images only.")
	fs.StringVar(&writeIndex, "write-index", "", "Write JSON index to a file.")
	addCommonFlags(fs, &opts)
	if _, err := parseFlagSet(fs, args); err != nil {
		return err
	}
	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	scan, err := scanPosts(root)
	if err != nil {
		return err
	}

	filtered := make([]PostInfo, 0, len(scan.Posts))
	for _, post := range scan.Posts {
		if draftsOnly && !post.Draft {
			continue
		}
		if liveOnly && post.Draft {
			continue
		}
		if showMissing && len(post.MissingImages) == 0 {
			continue
		}
		filtered = append(filtered, post)
	}
	if all || (!draftsOnly && !liveOnly && !showMissing) {
		filtered = scan.Posts
	}
	scan.Posts = filtered

	if writeIndex != "" {
		data, err := jsonMarshalIndent(scan)
		if err != nil {
			return err
		}
		if err := os.WriteFile(writeIndex, data, 0644); err != nil {
			return err
		}
	}
	if opts.JSON {
		return writeJSON(scan)
	}

	width := terminalWidth()
	titleWidth := width - 10 - 8 - 8 - 4 - 24
	if titleWidth < 24 {
		titleWidth = 24
	}
	if titleWidth > 44 {
		titleWidth = 44
	}
	categoryWidth := width - 10 - 8 - titleWidth - 8 - 10
	if categoryWidth < 12 {
		categoryWidth = 12
	}
	if categoryWidth > 30 {
		categoryWidth = 30
	}
	drafts := 0
	totalImages := 0
	totalMissing := 0
	rows := make([][]string, 0, len(filtered))
	for _, post := range filtered {
		status := "live"
		if post.Draft {
			status = "draft"
			drafts++
		}
		totalImages += len(post.Images)
		totalMissing += len(post.MissingImages)
		rows = append(rows, []string{
			post.Date,
			status,
			shortDisplayText(post.Title, titleWidth),
			fmt.Sprint(len(post.Images)),
			shortDisplayText(strings.Join(post.Categories, ", "), categoryWidth),
		})
	}
	fmt.Printf("Posts: %d", len(filtered))
	if drafts > 0 {
		fmt.Printf("  Drafts: %d", drafts)
	}
	fmt.Printf("  Images: %d", totalImages)
	if totalMissing > 0 {
		fmt.Printf("  Missing images: %d", totalMissing)
	}
	fmt.Println()
	fmt.Println()
	if len(rows) == 0 {
		fmt.Println("No posts matched.")
		return nil
	}
	printTable([]string{"DATE", "STATUS", "TITLE", "IMAGES", "CATEGORIES"}, rows)
	if showImages {
		for _, post := range filtered {
			if len(post.Images) == 0 {
				continue
			}
			fmt.Println()
			fmt.Println(shortDisplayText(post.Title, width))
			for _, image := range post.Images {
				marker := ""
				for _, missing := range post.MissingImages {
					if missing == image {
						marker = " (missing)"
						break
					}
				}
				fmt.Println("  ", image+marker)
			}
		}
	}
	if writeIndex != "" {
		fmt.Println()
		fmt.Println("Wrote index:", writeIndex)
	}
	return nil
}

func runView(args []string) error {
	var opts commonOptions
	var full, images, raw bool
	fs := flag.NewFlagSet("view", flag.ExitOnError)
	fs.BoolVar(&full, "full", false, "Show the full Markdown body.")
	fs.BoolVar(&images, "images", false, "Show image paths.")
	fs.BoolVar(&raw, "raw", false, "Show raw Markdown instead of the formatted terminal preview.")
	addCommonFlags(fs, &opts)
	if _, err := parseFlagSet(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: odt2post view <post-selector> [flags]")
	}
	selector := fs.Arg(0)
	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	scan, err := scanPosts(root)
	if err != nil {
		return err
	}
	post, err := requireOnePost(scan.Posts, selector)
	if err != nil {
		return err
	}
	if opts.JSON {
		return writeJSON(post)
	}
	printPostView(root, post, full, images, raw)
	return nil
}

func printPostView(root string, post PostInfo, full, showImages bool, raw bool) {
	width := terminalWidth()
	status := "live"
	if post.Draft {
		status = "draft"
	}
	title := shortDisplayText(post.Title, width)
	fmt.Println(title)
	fmt.Println(horizontalRule(displayWidth(title)))
	printDetailRow("File", relativeToRoot(root, post.Path), width)
	printDetailRow("Date", post.Date, width)
	printDetailRow("Status", status, width)
	if len(post.Tags) > 0 {
		printDetailRow("Tags", strings.Join(post.Tags, ", "), width)
	}
	if len(post.Categories) > 0 {
		printDetailRow("Categories", strings.Join(post.Categories, ", "), width)
	}
	if len(post.Series) > 0 {
		printDetailRow("Series", strings.Join(post.Series, ", "), width)
	}
	printDetailRow("Images", fmt.Sprint(len(post.Images)), width)
	if showImages || len(post.MissingImages) > 0 {
		for _, image := range post.Images {
			marker := ""
			for _, missing := range post.MissingImages {
				if missing == image {
					marker = " (missing)"
					break
				}
			}
			for _, line := range wrapDisplay(image+marker, width, "  - ") {
				fmt.Println(line)
			}
		}
	}
	fmt.Println()
	body := renderMarkdownBody(post.Body, full, raw, width)
	if body == "" {
		fmt.Println("(empty post)")
		return
	}
	fmt.Println(body)
}

func printDetailRow(key, value string, width int) {
	keyWidth := 12
	prefix := "  " + padRightDisplay(key, keyWidth) + " "
	for _, line := range wrapWithIndents(value, width, prefix, strings.Repeat(" ", displayWidth(prefix))) {
		fmt.Println(line)
	}
}

func relativeToRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func runDraftPublish(args []string, draft bool) error {
	var opts commonOptions
	fs := flag.NewFlagSet("draft", flag.ExitOnError)
	addCommonFlags(fs, &opts)
	if _, err := parseFlagSet(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		if draft {
			return fmt.Errorf("usage: odt2post draft <post-selector> [flags]")
		}
		return fmt.Errorf("usage: odt2post publish <post-selector> [flags]")
	}
	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	scan, err := scanPosts(root)
	if err != nil {
		return err
	}
	post, err := requireOnePost(scan.Posts, fs.Arg(0))
	if err != nil {
		return err
	}
	if err := setPostDraft(post.Path, draft); err != nil {
		return err
	}
	action := "Published"
	if draft {
		action = "Drafted"
	}
	if opts.JSON {
		return writeJSON(map[string]any{"path": post.Path, "draft": draft})
	}
	fmt.Println(action+":", post.Path)
	return nil
}

func runDelete(args []string) error {
	var opts commonOptions
	var withImages, keepImages, dryRun bool
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	fs.BoolVar(&withImages, "with-images", false, "Also move post-owned images to trash.")
	fs.BoolVar(&keepImages, "keep-images", false, "Keep all images.")
	fs.BoolVar(&dryRun, "dry-run", false, "Preview without moving files.")
	addCommonFlags(fs, &opts)
	if _, err := parseFlagSet(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: odt2post delete <post-selector> [flags]")
	}
	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	scan, err := scanPosts(root)
	if err != nil {
		return err
	}
	post, err := requireOnePost(scan.Posts, fs.Arg(0))
	if err != nil {
		return err
	}
	if keepImages {
		withImages = false
	}

	ownedImages := postOwnedImagePaths(root, scan.Posts, post)
	plan := deletePlan(root, post, ownedImages, withImages)
	execute := opts.Yes && !dryRun
	if opts.JSON {
		return writeJSON(map[string]any{"execute": execute, "plan": plan})
	}
	printDeletePlan(plan, execute)
	if !execute {
		fmt.Println("No files moved. Re-run with -yes to apply.")
		return nil
	}
	return applyMovePlan(plan)
}

func runCleanImages(args []string) error {
	var opts commonOptions
	var dryRun bool
	fs := flag.NewFlagSet("clean-images", flag.ExitOnError)
	fs.BoolVar(&dryRun, "dry-run", false, "Preview without moving files.")
	addCommonFlags(fs, &opts)
	if _, err := parseFlagSet(fs, args); err != nil {
		return err
	}
	root, err := resolveBlogRoot(opts.Root)
	if err != nil {
		return err
	}
	scan, err := scanPosts(root)
	if err != nil {
		return err
	}
	orphans, err := orphanImages(root, scan.Referenced)
	if err != nil {
		return err
	}
	plan := cleanImagesPlan(root, orphans)
	execute := opts.Yes && !dryRun
	if opts.JSON {
		return writeJSON(map[string]any{"execute": execute, "orphans": orphans, "plan": plan})
	}
	if len(orphans) == 0 {
		fmt.Println("No orphan images found.")
		return nil
	}
	printDeletePlan(plan, execute)
	if !execute {
		fmt.Println("No files moved. Re-run with -yes to apply.")
		return nil
	}
	return applyMovePlan(plan)
}

type moveAction struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func deletePlan(root string, post PostInfo, ownedImages []string, withImages bool) []moveAction {
	stamp := time.Now().Format("20060102-150405")
	base := filepath.Join(trashDir(root), stamp)
	actions := []moveAction{{
		From: post.Path,
		To:   filepath.Join(base, "posts", filepath.Base(post.Path)),
	}}
	if withImages {
		for _, image := range ownedImages {
			actions = append(actions, moveAction{
				From: image,
				To:   filepath.Join(base, "images", filepath.Base(image)),
			})
		}
	}
	return actions
}

func cleanImagesPlan(root string, orphans []string) []moveAction {
	stamp := time.Now().Format("20060102-150405")
	base := filepath.Join(trashDir(root), stamp, "images")
	var actions []moveAction
	for _, image := range orphans {
		actions = append(actions, moveAction{
			From: image,
			To:   filepath.Join(base, filepath.Base(image)),
		})
	}
	return actions
}

func printDeletePlan(plan []moveAction, execute bool) {
	if execute {
		fmt.Println("Moving files:")
	} else {
		fmt.Println("Move preview:")
	}
	for _, action := range plan {
		fmt.Println("  ", action.From)
		fmt.Println("    ->", action.To)
	}
}

func applyMovePlan(plan []moveAction) error {
	for _, action := range plan {
		if err := ensureInside(filepath.Dir(filepath.Dir(action.To)), action.To); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(action.To), 0755); err != nil {
			return err
		}
		if err := os.Rename(action.From, action.To); err != nil {
			return err
		}
	}
	return nil
}

func postOwnedImagePaths(root string, posts []PostInfo, target PostInfo) []string {
	usedByOthers := map[string]bool{}
	for _, post := range posts {
		if post.Path == target.Path {
			continue
		}
		for _, image := range post.Images {
			usedByOthers[image] = true
		}
	}
	var out []string
	for _, image := range target.Images {
		if usedByOthers[image] {
			continue
		}
		path, ok := imageURLToPath(root, image)
		if !ok || !fileExists(path) {
			continue
		}
		if err := ensureInside(postImagesDir(root), path); err != nil {
			continue
		}
		out = append(out, path)
	}
	return out
}

func orphanImages(root string, referenced []string) ([]string, error) {
	referencedSet := map[string]bool{}
	for _, url := range referenced {
		referencedSet[url] = true
	}
	var orphans []string
	dir := postImagesDir(root)
	if _, err := os.Stat(dir); err != nil {
		return nil, nil
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isImagePath(path) {
			return nil
		}
		url, ok := imagePathToURL(root, path)
		if !ok {
			return nil
		}
		if !referencedSet[url] {
			orphans = append(orphans, path)
		}
		return nil
	})
	return orphans, err
}
