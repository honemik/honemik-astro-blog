import { access, readFile, readdir } from 'node:fs/promises';
import assert from 'node:assert/strict';
import test from 'node:test';

const dist = new URL('../dist/', import.meta.url);

async function readOutput(path) {
  return readFile(new URL(path, dist), 'utf8');
}

test('page-level archive indexes use h1 headings', async () => {
  const pages = [
    ['posts/index.html', 'Posts'],
    ['archive/index.html', 'Archive'],
    ['tags/index.html', 'Tags'],
    ['categories/index.html', 'Categories']
  ];

  for (const [path, title] of pages) {
    const html = await readOutput(path);
    assert.match(html, new RegExp(`<h1>${title}</h1>`));
  }
});

test('homepage heading and image switch control have separate semantics', async () => {
  const html = await readOutput('index.html');
  assert.match(html, /<h1[^>]*hero-word[^>]*>[\s\S]*?honemik[\s\S]*?<\/h1>/);
  assert.match(html, /<\/h1>\s*<button class="hero-word-button"[^>]*aria-label="Switch cover image"/);
});

test('post display flags control breadcrumbs and the table of contents', async () => {
  const tocDisabled = await readOutput('posts/2021-11-06/index.html');
  const breadcrumbsDisabled = await readOutput('posts/2023-12-26/index.html');

  assert.doesNotMatch(tocDisabled, /class="article-toc"/);
  assert.doesNotMatch(breadcrumbsDisabled, /Back to all posts/);
});

test('the first Markdown image is rendered once in the article body', async () => {
  const html = await readOutput('posts/2023-12-26/index.html');
  const matches = html.match(/\/img\/post\/2023-12-25-01\.jpg/g) ?? [];

  assert.equal(matches.length, 1);
  assert.doesNotMatch(html, /class="article-cover"/);
});

test('Pagefind replaces the full-body JavaScript search payload', async () => {
  const search = await readOutput('search/index.html');

  await access(new URL('pagefind/pagefind.js', dist));
  assert.match(search, /import\('\/pagefind\/pagefind\.js'\)/);
  await assert.rejects(access(new URL('search-data.js', dist)));
});

test('generated font styles use WOFF2 files only', async () => {
  const fontCss = await readOutput('assets/fonts/zen-maru-gothic.css');
  const bundledCssFiles = (await readdir(new URL('_astro/', dist))).filter((file) => file.endsWith('.css'));
  const bundledCss = (await Promise.all(bundledCssFiles.map((file) => readOutput(`_astro/${file}`)))).join('\n');

  assert.match(fontCss, /zen-maru-gothic-400\.woff2/);
  assert.match(fontCss, /zen-maru-gothic-500\.woff2/);
  assert.match(fontCss, /zen-maru-gothic-700\.woff2/);
  assert.doesNotMatch(fontCss, /\.ttf/);
  assert.match(bundledCss, /ChenYuluoyan-2\.0-Thin\.woff2/);
  assert.doesNotMatch(bundledCss, /ChenYuluoyan-2\.0-Thin\.ttf/);
});
