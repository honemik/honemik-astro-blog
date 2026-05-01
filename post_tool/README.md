# odt2post

A zero-dependency Go tool that converts an `.odt` file (LibreOffice / OpenOffice Writer) into a blog post for this Astro blog. It extracts text as Markdown, copies embedded images to the correct `public/img/post/` directory, and generates a properly-formatted `.md` file with YAML frontmatter under `src/content/posts/`.

## Build

```bash
cd post_tool
go build -buildvcs=false -o odt2post.exe .
```

## Usage

```
odt2post.exe -f <file.odt> -title "My Post Title" [options]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-f` | *(required)* | Path to the `.odt` file |
| `-title` | *(required)* | Post title |
| `-desc` | `""` | Short description (optional) |
| `-tags` | `""` | Comma-separated tags, e.g. `"Life,Travel"` |
| `-date` | today | Post date in `YYYY-MM-DD` format |
| `-author` | `honemik` | Author name |
| `-draft` | `false` | Set to `true` to mark as draft |
| `-root` | `..` | Path to the blog root (default assumes you run from `post_tool/`) |

### Examples

**Basic conversion (date defaults to today):**
```bash
./odt2post.exe -f ~/Documents/my-trip.odt -title "My Trip to Kyoto" -tags "Travel,Life"
```

**Full options, running from the blog root:**
```bash
./post_tool/odt2post.exe \
  -f ./my-trip.odt \
  -title "My Trip to Kyoto" \
  -desc "A weekend in Kyoto during autumn" \
  -tags "Travel,Life" \
  -date 2025-11-15 \
  -root .
```

## Output

Given a title `"My Trip to Kyoto"` and date `2025-11-15`, the tool produces:

- **`src/content/posts/2025-11-15-my-trip-to-kyoto.md`** — Markdown post with frontmatter
- **`public/img/post/2025-11-15-01.jpg`**, `2025-11-15-02.png`, … — Extracted images, numbered in document order

Images are referenced in the markdown as:
```markdown
![](/img/post/2025-11-15-01.jpg)
```

## What it converts

| ODT element | Markdown output |
|-------------|-----------------|
| Heading (level 1–6) | `# …` to `###### …` |
| Paragraph | Plain paragraph text |
| Embedded image (`draw:frame`) | `![](/img/post/…)` |
| Bullet list | `- item` |
| Nested list | Indented `  - item` |
| Line break | `  ` (two trailing spaces) |
| Tab character | `\t` |

> **Note:** Complex ODT formatting (tables, text boxes, footnotes, colours, custom styles) is intentionally simplified or dropped — this tool focuses on clean readable prose with inline images.
