# Post Tool Improvement Plan

## Goal

Turn the current `odt2post` converter into a modern terminal CLI for importing ODT notes into the Astro blog, managing existing posts, and keeping post images organized.

The tool should stay focused on this repository:

- Posts: `src/content/posts/*.md`
- Images: `public/img/post/*`
- Existing schema supports: `title`, `description`, `author`, `date`, `tags`, `categories`, `series`, `draft`, `ShowToc`, `ShowBreadCrumbs`

## Current State

The current Go tool:

- Reads `content.xml` from an `.odt` zip archive.
- Converts basic headings, paragraphs, lists, and embedded images to Markdown.
- Writes Markdown to `src/content/posts/`.
- Copies embedded images to `public/img/post/`.
- Requires `-title`.
- Defaults date to today.
- Does not read ODT metadata from `meta.xml`.
- Does not manage existing posts.
- Does not convert or compress images.

## Proposed CLI Shape

Rename the binary behavior from a single conversion command into subcommands:

```text
odt2post import <file.odt> [flags]
odt2post inspect <file.odt> [flags]
odt2post list [flags]
odt2post view <post-selector> [flags]
odt2post draft <post-selector> [flags]
odt2post publish <post-selector> [flags]
odt2post delete <post-selector> [flags]
odt2post clean-images [flags]
```

Use `import` as the main workflow. Keep the old flag style working as a compatibility alias:

```text
odt2post -f note.odt -title "Title"
```

The compatibility path should internally call `import`.

## Modern Terminal UX

Recommended Go libraries:

- `github.com/spf13/cobra` for subcommands and help output.
- `gopkg.in/yaml.v3` for frontmatter and ODT metadata block parsing.
- `github.com/charmbracelet/huh` for optional interactive prompts.
- `github.com/charmbracelet/lipgloss` for readable terminal styling.

Behavior:

- Non-interactive by default when all needed data is known.
- Interactive review when information is missing or when `--interactive` is used.
- Always support script-friendly flags: `--yes`, `--json`, `--no-color`, `--root`.
- Use clear previews before destructive actions.
- Auto-detect blog root by walking upward until `astro.config.mjs` and `src/content/config.ts` are found. Keep `--root` as an override.

## Metadata Extraction Rules

Import should build post metadata using this precedence:

1. CLI flags.
2. ODT post metadata block in the document body.
3. ODT package metadata from `meta.xml`.
4. Content heuristics.
5. Filename/defaults.

### Date

Default post date should be the ODT last edited date.

Implementation order:

1. Read `meta.xml`.
2. Use `office:document-meta/office:meta/dc:date` as the last edited datetime.
3. Convert to `YYYY-MM-DD` for frontmatter.
4. If `dc:date` is missing, fallback to the `.odt` file modified time.
5. If both fail, fallback to today's date and show a warning.

Allow `--date YYYY-MM-DD` as an intentional override.

### Title

Default title should come from the ODT document title.

Implementation order:

1. `title:` in the ODT metadata block.
2. `dc:title` from `meta.xml`.
3. First heading level 1 in the document.
4. Cleaned filename without extension.

The existing required `-title` flag should become optional.

### Description

Implementation order:

1. `description:` in the ODT metadata block.
2. CLI `--desc`.
3. First non-empty paragraph, trimmed to around 120 characters.
4. Empty string.

### Tags, Categories, Series, Draft

Use the ODT metadata block or CLI flags.

Add `categories` to generated frontmatter because the Astro content schema already supports it.

Recommended flags:

```text
--tags Blog,Note
--categories Diary,Tech
--series Site-Rebuild
--draft
```

## ODT Metadata Block Format

Use a simple YAML-like block at the top of the ODT body for information that is hard to infer.

Example text typed at the top of the ODT:

```yaml
---
title: My Post Title
description: Short summary for the blog list
tags: [Blog, Note]
categories: [Life]
series: []
draft: false
cover: first
---
```

Rules:

- The block must appear before normal content.
- The converter removes this block from the final Markdown.
- All fields are optional.
- `date` is allowed only as an override, but the default remains the ODT last edited date.
- `cover` can be `first`, an image number like `2`, or empty.

This keeps the ODT easy to write while avoiding extra sidecar files.

## ODT Parsing Improvements

Read these files from the ODT zip:

- `content.xml` for body content.
- `meta.xml` for title and last edited date.
- `Pictures/*` and any other image-looking paths for embedded images.

Improve conversion behavior:

- Preserve image order by collecting images in document traversal order.
- Support multiple images inside the same paragraph/frame.
- Keep alt text if the ODT frame has a title/name/caption that can be read.
- Strip the metadata block before Markdown conversion.
- Keep current simple Markdown output for unsupported formatting.

## Image Conversion And Compression

Default image mode should be:

```text
--image-format webp
--image-quality 82
--max-width 1600
```

Implementation strategy:

1. Try to convert raster images to `.webp`.
2. Use `cwebp` if installed.
3. Fallback to ImageMagick `magick` if installed.
4. If no converter is installed, copy the original image and show a warning.

Special cases:

- Do not convert SVG.
- Do not convert animated GIF unless animation support is explicitly handled.
- Preserve original extension when WebP conversion is unavailable.
- Update Markdown image links to match the final written image filename.

Recommended filename format:

```text
YYYY-MM-DD-slug-01.webp
YYYY-MM-DD-slug-02.webp
```

Add image deduplication by hashing image bytes before writing. If the same embedded image appears twice, write one file and reuse the link.

## Post And Image Indexing

Implement a scanner that reads every post under `src/content/posts/` and extracts:

- Markdown path.
- Slug/filename.
- Title.
- Date.
- Draft status.
- Tags.
- Categories.
- Series.
- Image links in the body.
- Missing image files.
- Orphan image candidates.

The scanner should be the shared backend for `list`, `view`, `draft`, `publish`, `delete`, and `clean-images`.

### `list`

Show a table:

```text
DATE        STATUS   TITLE                    IMAGES  CATEGORIES
2026-05-01  live     This Post                1       Blog
2024-08-05  live     Tiddly wiki note         0       Blog, Note
```

Useful flags:

```text
--drafts
--live
--all
--images
--missing-images
--json
```

When `--images` is used, include image paths under each post.

### Optional Manifest

Add:

```text
odt2post list --json --write-index post_tool/post_index.json
```

This writes a generated inventory of posts and images. Do not rely on the manifest for correctness; scanning Markdown remains the source of truth.

## Viewing Notes And Posts

Interpret "notes" as imported posts/notes in `src/content/posts/`.

### `view <post-selector>`

Selectors should support:

- Exact filename.
- Date prefix, such as `2026-05-01`.
- Slug substring.
- Title substring.

Output:

- Frontmatter summary.
- Markdown path.
- Image list.
- Missing image warnings.
- First 30 to 60 lines of Markdown body.

Flags:

```text
--full
--images
--json
```

If multiple posts match, show a numbered list and ask the user to choose in interactive mode. In non-interactive mode, fail with a clear message.

## Draft, Publish, And Delete

### `draft <post-selector>`

Set `draft: true` in frontmatter.

Rules:

- Preserve existing frontmatter fields.
- Do not rewrite the body.
- Confirm when multiple posts match.

### `publish <post-selector>`

Set `draft: false`.

### `delete <post-selector>`

Use safe delete by default:

- Move the Markdown file to `post_tool/.trash/<timestamp>/posts/`.
- Move post-owned images to `post_tool/.trash/<timestamp>/images/` only when `--with-images` is passed.
- Default to dry-run preview unless `--yes` is passed or the user confirms interactively.

Flags:

```text
--with-images
--keep-images
--yes
--dry-run
```

Image ownership rule:

- Only delete images referenced by the selected post and not referenced by any other post.
- Never delete images outside `public/img/post/`.

## Clean Orphan Images

### `clean-images`

Find images in `public/img/post/` that are not referenced by any Markdown post.

Default behavior:

- Print orphan candidates.
- Do not remove anything unless `--yes` is used.
- Move removed images to `post_tool/.trash/<timestamp>/images/`.

Flags:

```text
--dry-run
--yes
--json
```

## Suggested Project Structure

Refactor from one large `main.go` into:

```text
post_tool/
  main.go
  go.mod
  README.md
  tool_plan.md
  internal/
    cli/
      root.go
      import.go
      inspect.go
      list.go
      view.go
      draft.go
      delete.go
    odt/
      reader.go
      metadata.go
      markdown.go
    post/
      frontmatter.go
      scanner.go
      selector.go
      writer.go
    image/
      extract.go
      convert.go
      hash.go
    ui/
      table.go
      prompts.go
```

Keep the first implementation small. The important part is separating ODT import logic from post management logic.

## Implementation Phases

### Phase 1: Metadata Auto-Detection

- Parse `meta.xml`.
- Make `--title` optional.
- Use ODT last edited date as default date.
- Add categories and series frontmatter output.
- Add `inspect <file.odt>` to preview detected metadata and images without writing.

### Phase 2: ODT Metadata Block

- Detect the top YAML-like block.
- Merge block fields with detected metadata.
- Remove block from Markdown body.
- Add validation and helpful error messages for invalid YAML.

### Phase 3: Image WebP Pipeline

- Add image naming based on date and slug.
- Add image hashing and deduplication.
- Add optional WebP conversion through `cwebp` or `magick`.
- Update Markdown image links to final output names.
- Warn clearly when WebP conversion is unavailable.

### Phase 4: Post Inventory

- Add frontmatter parser.
- Add Markdown image-link scanner.
- Add `list` command with table and JSON output.
- Add missing-image detection.

### Phase 5: View And Status Commands

- Add post selectors.
- Add `view`, `draft`, and `publish`.
- Preserve frontmatter structure as much as practical.

### Phase 6: Safe Delete And Cleanup

- Add `.trash` move behavior.
- Add `delete`.
- Add `clean-images`.
- Make all destructive commands preview first.

### Phase 7: README Refresh

- Update examples.
- Document the ODT metadata block.
- Document optional image tool dependencies:
  - `cwebp`
  - ImageMagick `magick`
- Document safe delete behavior.

## Acceptance Checklist

- `odt2post import note.odt` works without manually passing title or date.
- Date defaults to ODT last edited date.
- Title defaults to ODT title, then first heading, then filename.
- Categories can be set from the ODT metadata block.
- Images are converted to WebP when a converter is available.
- Markdown image links point to the actual generated files.
- `odt2post list --images` shows posts and their images.
- `odt2post view <selector>` shows a readable post preview.
- `odt2post draft <selector>` sets `draft: true`.
- `odt2post publish <selector>` sets `draft: false`.
- `odt2post delete <selector>` safely moves posts to trash.
- `odt2post clean-images` finds image files not referenced by any post.
- Existing one-shot conversion usage still works or prints a clear migration message.
