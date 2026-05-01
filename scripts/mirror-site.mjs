import fs from 'node:fs/promises';
import path from 'node:path';

const ORIGIN = 'https://futami-akashi-lib.jp';
const ROOT = new URL(`${ORIGIN}/`);
const PUBLIC_DIR = path.resolve('public');
const OUTPUT_DIR = path.resolve('output');
const OUTPUT_JSON = path.join(OUTPUT_DIR, 'mirror-pages.json');
const FONT_CSS_PATH = path.join(PUBLIC_DIR, 'assets/fonts/zen-maru-gothic.css');
const FONT_URL = 'https://fonts.googleapis.com/css2?family=Zen+Maru+Gothic:wght@400;500;700&display=swap';
const LOCAL_SCHEMA_AVATAR = '/wp/wp-content/uploads/2025/03/logo.png';

const htmlPages = new Map();
const pageQueue = [];
const queuedPages = new Set();
const assetQueue = [];
const queuedAssets = new Set();
const downloadedAssets = new Set();

const assetExtensions = new Set([
  '.css', '.js', '.jpg', '.jpeg', '.png', '.webp', '.svg', '.gif', '.ico',
  '.json', '.webmanifest', '.woff', '.woff2', '.ttf', '.eot', '.mp4', '.webm',
  '.pdf', '.txt'
]);

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isInternal(url) {
  return url.hostname === ROOT.hostname;
}

function normalizePageUrl(input, base = ROOT.href) {
  let url;
  try {
    url = new URL(input, base);
  } catch {
    return null;
  }

  if (!isInternal(url)) return null;
  url.hash = '';
  url.search = '';

  if (url.pathname.startsWith('/wp/wp-admin')) return null;
  if (url.pathname.startsWith('/wp-json')) return null;
  if (url.pathname.includes('/feed/')) return null;
  if (url.pathname.endsWith('/feed')) return null;
  if (url.pathname.endsWith('/trackback/')) return null;

  const ext = path.posix.extname(url.pathname).toLowerCase();
  if (assetExtensions.has(ext)) return null;

  if (!url.pathname.endsWith('/')) {
    url.pathname += '/';
  }

  return url.href;
}

function normalizeAssetUrl(input, base = ROOT.href) {
  let url;
  try {
    url = new URL(input, base);
  } catch {
    return null;
  }

  if (!isInternal(url)) return null;
  url.hash = '';

  const ext = path.posix.extname(url.pathname).toLowerCase();
  if (!assetExtensions.has(ext)) return null;

  return url.href;
}

function localPathForAsset(urlString) {
  const url = new URL(urlString);
  const decodedPath = decodeURIComponent(url.pathname);
  return path.join(PUBLIC_DIR, decodedPath);
}

function enqueuePage(url) {
  const normalized = normalizePageUrl(url);
  if (!normalized || queuedPages.has(normalized)) return;
  queuedPages.add(normalized);
  pageQueue.push(normalized);
}

function enqueueAsset(url, base = ROOT.href) {
  const normalized = normalizeAssetUrl(url, base);
  if (!normalized || queuedAssets.has(normalized)) return;
  queuedAssets.add(normalized);
  assetQueue.push(normalized);
}

function extractAttributeUrls(html, base) {
  const attrPattern = /\b(?:href|src|data-src|poster|action)=["']([^"']+)["']/gi;
  for (const match of html.matchAll(attrPattern)) {
    const value = match[1].trim();
    enqueuePage(value, base);
    enqueueAsset(value, base);
  }

  const srcsetPattern = /\b(?:srcset|data-srcset|imagesrcset)=["']([^"']+)["']/gi;
  for (const match of html.matchAll(srcsetPattern)) {
    for (const part of match[1].split(',')) {
      const candidate = part.trim().split(/\s+/)[0];
      enqueueAsset(candidate, base);
    }
  }

  const absolutePattern = /https?:\/\/futami-akashi-lib\.jp\/[^\s"'<>),]+/gi;
  for (const match of html.matchAll(absolutePattern)) {
    enqueuePage(match[0], base);
    enqueueAsset(match[0], base);
  }

  const rootAssetPattern = /["'(](\/(?:assets|wp)\/[^"')\s<>]+)["')]/gi;
  for (const match of html.matchAll(rootAssetPattern)) {
    enqueueAsset(match[1], base);
  }

  const cssUrlPattern = /url\((['"]?)([^'")]+)\1\)/gi;
  for (const match of html.matchAll(cssUrlPattern)) {
    enqueueAsset(match[2], base);
  }
}

function rewriteSrcset(value) {
  return value.split(',').map((part) => {
    const trimmed = part.trim();
    const [candidate, ...rest] = trimmed.split(/\s+/);
    return [rewriteUrl(candidate), ...rest].join(' ');
  }).join(', ');
}

function rewriteUrl(value) {
  if (!value || value.startsWith('data:') || value.startsWith('mailto:') || value.startsWith('tel:')) {
    return value;
  }

  try {
    const url = new URL(value, ROOT.href);
    if (url.hostname === ROOT.hostname) {
      return `${url.pathname}${url.search}${url.hash}`;
    }
  } catch {
    return value;
  }

  return value;
}

function stripRemoteTracking(html) {
  return html
    .replace(/<script\b[^>]*googletagmanager\.com\/gtag\/js[^>]*><\/script>/gi, '')
    .replace(/<script>\s*window\.dataLayer[\s\S]*?gtag\('config', 'G-9CK850B4C1'\);\s*<\/script>/i, '')
    .replace(/<script[^>]*cdn\.gtranslate\.net[^>]*><\/script>/gi, '');
}

function rewriteHtml(html) {
  let output = stripRemoteTracking(html);

  output = output.replace(/<link[^>]+href=["']https:\/\/fonts\.(?:googleapis|gstatic)\.com[^"']*["'][^>]*>\s*/gi, '');
  output = output.replace(
    /<link[^>]+href=["']https:\/\/fonts\.googleapis\.com\/css2\?family=Zen\+Maru\+Gothic[^"']*["'][^>]*>/i,
    '<link href="/assets/fonts/zen-maru-gothic.css" rel="stylesheet">'
  );
  output = output.replace('</head>', '<link href="/assets/fonts/zen-maru-gothic.css" rel="stylesheet">\n</head>');

  output = output.replace(/https?:\/\/futami-akashi-lib\.jp/gi, '');
  output = output.replace(/\/\/futami-akashi-lib\.jp/gi, '');
  output = output.replace(/https?:\/\/(assets|wp)\//gi, '/$1/');
  output = output.replace(/\/\/(assets|wp)\//gi, '/$1/');
  output = output.replace(/https:\/\/secure\.gravatar\.com\/avatar\/[^"\\<>\s]+/gi, LOCAL_SCHEMA_AVATAR);
  output = output.replace(
    /<div class="gtranslate_wrapper"([^>]*)><\/div>/gi,
    '<div class="gtranslate_wrapper"$1><a href="#" class="glink" data-gt-lang="ja">日本語</a><a href="#" class="glink" data-gt-lang="en">English</a></div>'
  );

  output = output.replace(/\b(srcset|data-srcset|imagesrcset)=["']([^"']+)["']/gi, (_full, attr, value) => {
    return `${attr}="${rewriteSrcset(value)}"`;
  });

  output = output.replace(/\b(href|src|data-src|poster|action)=["']([^"']+)["']/gi, (_full, attr, value) => {
    return `${attr}="${rewriteUrl(value)}"`;
  });

  output = output.replace(/<html lang="ja"/i, '<html lang="ja"');
  return output;
}

async function downloadGoogleFonts() {
  await fs.mkdir(path.dirname(FONT_CSS_PATH), { recursive: true });
  const css = await fetchText(FONT_URL);
  const fontUrls = [...css.matchAll(/url\((https:\/\/fonts\.gstatic\.com\/[^)]+)\)/g)].map((match) => match[1]);
  let rewritten = css;

  for (const [index, fontUrl] of fontUrls.entries()) {
    const weight = index === 0 ? '400' : index === 1 ? '500' : '700';
    const fileName = `zen-maru-gothic-${weight}.ttf`;
    const localUrl = `/assets/fonts/${fileName}`;
    const response = await fetch(fontUrl);
    if (!response.ok) throw new Error(`${response.status} ${fontUrl}`);
    await fs.writeFile(path.join(PUBLIC_DIR, 'assets/fonts', fileName), Buffer.from(await response.arrayBuffer()));
    rewritten = rewritten.replace(fontUrl, localUrl);
  }

  await fs.writeFile(FONT_CSS_PATH, rewritten);
  console.log(`wrote local fonts to ${FONT_CSS_PATH}`);
}

function rewriteAssetText(text) {
  return text
    .replace(/https?:\/\/futami-akashi-lib\.jp/gi, '')
    .replace(/\/\/futami-akashi-lib\.jp/gi, '')
    .replace(/https?:\/\/(assets|wp)\//gi, '/$1/')
    .replace(/\/\/(assets|wp)\//gi, '/$1/');
}

async function fetchText(url) {
  const response = await fetch(url, {
    headers: {
      'user-agent': 'Mozilla/5.0 local Astro visual recreation crawler'
    }
  });
  if (!response.ok) throw new Error(`${response.status} ${url}`);
  return response.text();
}

async function seedFromSitemap() {
  enqueuePage(ROOT.href);
  const index = await fetchText(`${ORIGIN}/sitemap_index.xml`);
  const sitemapUrls = [...index.matchAll(/<loc>(.*?)<\/loc>/g)].map((match) => match[1]);

  for (const sitemapUrl of sitemapUrls) {
    const xml = await fetchText(sitemapUrl);
    for (const match of xml.matchAll(/<loc>(.*?)<\/loc>/g)) {
      const loc = match[1];
      enqueuePage(loc);
      enqueueAsset(loc);
    }
    for (const match of xml.matchAll(/<image:loc>(.*?)<\/image:loc>/g)) {
      enqueueAsset(match[1]);
    }
  }
}

async function crawlPages() {
  while (pageQueue.length) {
    const url = pageQueue.shift();
    if (htmlPages.has(url)) continue;

    try {
      const response = await fetch(url, {
        headers: {
          'user-agent': 'Mozilla/5.0 local Astro visual recreation crawler'
        }
      });

      if (!response.ok) {
        console.warn(`page failed ${response.status}: ${url}`);
        continue;
      }

      const contentType = response.headers.get('content-type') || '';
      if (!contentType.includes('text/html')) continue;

      const html = await response.text();
      htmlPages.set(url, rewriteHtml(html));
      extractAttributeUrls(html, url);
      console.log(`page ${htmlPages.size}: ${url}`);
      await sleep(80);
    } catch (error) {
      console.warn(`page failed: ${url}\n${error.message}`);
    }
  }
}

async function downloadAssets() {
  while (assetQueue.length) {
    const url = assetQueue.shift();
    if (downloadedAssets.has(url)) continue;
    downloadedAssets.add(url);

    const localPath = localPathForAsset(url);
    try {
      await fs.mkdir(path.dirname(localPath), { recursive: true });
      const response = await fetch(url, {
        headers: {
          'user-agent': 'Mozilla/5.0 local Astro visual recreation crawler'
        }
      });

      if (!response.ok) {
        console.warn(`asset failed ${response.status}: ${url}`);
        continue;
      }

      const contentType = response.headers.get('content-type') || '';
      const buffer = Buffer.from(await response.arrayBuffer());

      if (
        contentType.includes('text/css') ||
        contentType.includes('javascript') ||
        contentType.includes('json') ||
        url.endsWith('.css') ||
        url.endsWith('.js') ||
        url.endsWith('.webmanifest')
      ) {
        const text = rewriteAssetText(buffer.toString('utf8'));
        await fs.writeFile(localPath, text);
        extractAttributeUrls(text, url);
      } else {
        await fs.writeFile(localPath, buffer);
      }

      console.log(`asset ${downloadedAssets.size}: ${url}`);
      await sleep(40);
    } catch (error) {
      console.warn(`asset failed: ${url}\n${error.message}`);
    }
  }
}

function pageRecord(url, html) {
  const parsed = new URL(url);
  let route = decodeURIComponent(parsed.pathname);
  if (route !== '/' && route.endsWith('/')) route = route.slice(0, -1);
  const slug = route === '/' ? '' : route.slice(1);
  return { route, slug, html };
}

async function writeData() {
  await fs.mkdir(OUTPUT_DIR, { recursive: true });
  const records = [...htmlPages.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([url, html]) => pageRecord(url, html));
  await fs.writeFile(OUTPUT_JSON, JSON.stringify(records, null, 2));
  console.log(`wrote ${records.length} pages to ${OUTPUT_JSON}`);
}

await seedFromSitemap();
await crawlPages();
await downloadAssets();
await downloadGoogleFonts();
await writeData();

console.log(`done: ${htmlPages.size} pages, ${downloadedAssets.size} assets`);
