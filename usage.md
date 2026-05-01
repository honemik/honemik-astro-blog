# How to Use Your Astro Editorial Blog

Welcome to your newly migrated, beautifully handcrafted Astro blog! Here is your quick reference guide for writing posts and updating the site.

---

## 1. How to Write a New Post

All your blog posts live in the **`src/content/posts/`** directory. To create a new post, simply create a new Markdown file (e.g., `2026-05-01-my-new-post.md`) in that folder.

### The Frontmatter
Every post must start with a block of "frontmatter" (metadata) surrounded by `---` lines. Here is the template you should use:

```yaml
---
title: "Your Post Title Here"
description: "A short, beautiful caption that appears on the homepage cards."
date: 2026-05-01
tags:
  - Life
  - Photography
categories:
  - Journal
---
```
*   **title:** The main headline of your post.
*   **description:** This is important! It shows up as the handwritten micro-caption on the homepage and editorial cards.
*   **date:** Used to sort your posts chronologically.
*   **tags / categories:** Used to group your posts in the "Collections" section.

### Adding Images
1. **Where to put the image file:** Save your new images inside the **`public/img/`** folder (you can create subfolders like `public/img/2026/` to keep it organized).
2. **How to show it in the post:** Use standard Markdown image syntax, starting the path with a forward slash `/` (which represents the `public` folder).

```markdown
![A beautiful sunset over the city](/img/2026/sunset.jpg)
```

---

## 2. How to Update the Blog (Deploying)

Because we connected your Cloudflare Pages account directly to your GitHub repository, **you never have to manually upload to Cloudflare.** Cloudflare will automatically detect when you push new code to GitHub and will rebuild your website for you.

When you finish writing a new post or making changes, simply open your terminal in your project folder (`D:\Code\honemik_blog`) and run these three Git commands:

```bash
git add .
git commit -m "Added a new post about my day"
git push origin main
```

Within 1-2 minutes, Cloudflare will automatically build and publish your new post to your live website!

---

## 3. Other Instructions

### Running the Local Preview
If you want to see how your new post looks before publishing it to the world, run the local development server:
```bash
npm run dev
```
Then, open your browser and go to `http://localhost:4321/`.

### Updating the Homepage Thumbnails
The homepage cover image is chosen dynamically from **`public/assets/img/Background/`**. If you want to change the pool of cover images, add or remove image files in that folder.
