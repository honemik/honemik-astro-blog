# odt2post

`odt2post` imports LibreOffice/OpenOffice `.odt` notes into this Astro blog and manages existing Markdown posts.

It reads ODT text and metadata, writes posts under `src/content/posts/`, stores post images under `public/img/post/`, can convert extracted images to WebP, and includes safe post/image management commands.

## Build

```bash
cd post_tool
go build -buildvcs=false -o odt2post.exe .
```

## Commands

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

Short aliases are available for daily use:

```text
odt2post add note.odt      # import
odt2post ls                # list
odt2post show 2026-05-01   # view
odt2post rm 2026-05-01     # safe delete preview
odt2post orphans           # clean-images preview
```

Legacy import still works:

```bash
./odt2post.exe -f note.odt -title "My Title"
```

## Import

Basic import:

```bash
./odt2post.exe import ~/Documents/my-note.odt
```

Preview without writing:

```bash
./odt2post.exe inspect ~/Documents/my-note.odt
./odt2post.exe import ~/Documents/my-note.odt -dry-run
```

Override detected metadata:

```bash
./odt2post.exe import ./my-note.odt \
  -title "My Post Title" \
  -desc "Short post summary" \
  -tags "Blog,Note" \
  -categories "Life" \
  -date 2026-05-04
```

The default date is the ODT last edited date from `meta.xml` (`dc:date`). If that is missing, the tool falls back to the `.odt` file modified time.

The default title comes from ODT metadata (`dc:title`), then the first H1 heading, then the filename.

## ODT Metadata Block

For fields that are hard to infer, type this block at the very top of the ODT document:

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

The block is removed from the final Markdown. All fields are optional. A CLI flag overrides the block when both are present.

## Image Handling

Default image settings:

```text
-image-format webp
-image-quality 82
-max-width 1600
```

Images are named like:

```text
YYYY-MM-DD-post-slug-01.webp
YYYY-MM-DD-post-slug-02.webp
```

WebP conversion uses the first available tool:

1. `cwebp`
2. ImageMagick `magick`

If no converter is available, the original image is copied and the Markdown link uses the original extension. SVG files are copied as-is.

## Post Inventory

List posts:

```bash
./odt2post.exe ls
./odt2post.exe ls -images
./odt2post.exe ls -missing-images
./odt2post.exe ls -json
```

Write an index file:

```bash
./odt2post.exe ls -json -write-index post_index.json
```

The list view is terminal-width aware and accounts for Chinese/Japanese full-width characters, so mixed-language titles should stay aligned.

## View Notes

Selectors can be an exact filename, date prefix, slug substring, or title substring.

```bash
./odt2post.exe show 2026-05-01
./odt2post.exe show "Tiddly" -images
./odt2post.exe show 2026-05-01 -full
./odt2post.exe show 2026-05-01.md -raw
```

The normal view formats Markdown for terminal reading: it wraps long paragraphs, keeps short note lines readable, and shows image links as compact `[image]` lines. Use `-raw` when you want the exact Markdown body.

## Draft And Publish

```bash
./odt2post.exe draft 2026-05-01
./odt2post.exe publish 2026-05-01
```

These commands update only the `draft` field in frontmatter.

## Safe Delete

Delete defaults to preview mode:

```bash
./odt2post.exe delete 2026-05-01
```

Apply the move to trash:

```bash
./odt2post.exe delete 2026-05-01 -yes
```

Also move post-owned images:

```bash
./odt2post.exe delete 2026-05-01 -with-images -yes
```

Deleted files are moved to:

```text
post_tool/.trash/<timestamp>/
```

Images are moved only when they are referenced by the selected post and not referenced by another post.

## Clean Orphan Images

Preview unreferenced images:

```bash
./odt2post.exe clean-images
```

Move orphan images to trash:

```bash
./odt2post.exe clean-images -yes
```

## Common Flags

| Flag | Description |
|------|-------------|
| `-root <path>` | Blog root. Auto-detected when omitted. |
| `-json` | Print JSON output where supported. |
| `-yes` | Confirm write/delete actions. |
| `-no-color` | Reserved for script-friendly output. |
| `-interactive` | Reserved for prompt-based flows. |

## Converted ODT Elements

| ODT element | Markdown output |
|-------------|-----------------|
| Heading level 1-6 | `#` through `######` |
| Paragraph | Plain paragraph text |
| Embedded image | Markdown image link |
| Bullet list | `- item` |
| Nested list | Indented list |
| Line break | Markdown hard break |
| Tab | Tab character |

Complex ODT formatting such as tables, text boxes, footnotes, colors, and custom styles is intentionally simplified.
