import { readFile } from 'node:fs/promises';
import assert from 'node:assert/strict';
import test from 'node:test';

const cssPath = new URL('../src/styles/global.css', import.meta.url);
const expectedImportManifest = [
  './partials/base.css',
  './partials/quiet-reference.css',
  './partials/editorial.css',
  './partials/final-overrides.css',
  './partials/motion.css',
  './partials/tag-refinement.css',
  './partials/micro-interactions.css'
];

async function readCssFile(fileUrl, seen = new Set()) {
  assert.equal(seen.has(fileUrl.href), false, `CSS import cycle detected for ${fileUrl.href}`);
  seen.add(fileUrl.href);

  const css = await readFile(fileUrl, 'utf8');
  const importPattern = /^@import\s+['"](.+?)['"];\s*$/gm;
  let resolvedCss = css;

  for (const match of [...css.matchAll(importPattern)]) {
    const importPath = match[1];
    assert.equal(importPath.startsWith('.'), true, `CSS imports should stay local: ${importPath}`);
    const importedCss = await readCssFile(new URL(importPath, fileUrl), seen);
    resolvedCss = resolvedCss.replace(match[0], importedCss);
  }

  seen.delete(fileUrl.href);
  return resolvedCss;
}

async function readCss() {
  return readCssFile(cssPath);
}

function mediaBlocks(css, query) {
  const blocks = [];
  const mediaStart = `@media ${query}`;
  let searchIndex = 0;

  while (searchIndex < css.length) {
    const mediaIndex = css.indexOf(mediaStart, searchIndex);
    if (mediaIndex === -1) break;

    const openIndex = css.indexOf('{', mediaIndex);
    assert.notEqual(openIndex, -1, `${query} media query should open a block`);

    let depth = 0;
    let closeIndex = openIndex;
    for (; closeIndex < css.length; closeIndex += 1) {
      if (css[closeIndex] === '{') depth += 1;
      if (css[closeIndex] === '}') depth -= 1;
      if (depth === 0) break;
    }

    assert.equal(depth, 0, `${query} media query should close its block`);
    blocks.push(css.slice(openIndex + 1, closeIndex));
    searchIndex = closeIndex + 1;
  }

  return blocks;
}

test('global CSS has balanced braces and no merge conflict markers', async () => {
  const css = await readCss();
  let balance = 0;

  for (const char of css) {
    if (char === '{') balance += 1;
    if (char === '}') balance -= 1;
    assert.ok(balance >= 0, 'CSS closes more braces than it opens');
  }

  assert.equal(balance, 0, 'CSS should have balanced braces');
  assert.equal(css.includes('<<<<<<<'), false, 'CSS should not contain merge conflict markers');
  assert.equal(css.includes('>>>>>>>'), false, 'CSS should not contain merge conflict markers');
});

test('global CSS remains a conservative import manifest', async () => {
  const css = await readFile(cssPath, 'utf8');
  const imports = [...css.matchAll(/^@import\s+['"](.+?)['"];\s*$/gm)].map((match) => match[1]);
  const nonImportCss = css.replace(/^@import\s+['"].+?['"];\s*$/gm, '').trim();

  assert.deepEqual(imports, expectedImportManifest);
  assert.equal(nonImportCss, '', 'global.css should only coordinate ordered partial imports');
});

test('footer social responsive column rule is not duplicated', async () => {
  const css = await readCss();
  const duplicateRulePattern = /\.footer-social\s*\{\s*grid-template-columns:\s*repeat\(2,\s*max-content\);\s*\}/g;
  const matches = css.match(duplicateRulePattern) ?? [];

  assert.equal(matches.length, 1);
});

test('quiet reference color tokens have a single source of truth', async () => {
  const css = await readCss();
  const quietColorTokens = [
    '--color-ink',
    '--color-muted',
    '--color-green',
    '--color-blue',
    '--color-yellow',
    '--color-pink',
    '--color-wood'
  ];

  for (const token of quietColorTokens) {
    const definitionPattern = new RegExp(`${token}\\s*:`, 'g');
    const definitions = css.match(definitionPattern) ?? [];
    assert.equal(definitions.length, 1, `${token} should be defined once`);
  }
});

test('media queries stay conservatively consolidated', async () => {
  const css = await readCss();

  assert.equal(mediaBlocks(css, '(max-width: 920px)').length, 4);
  assert.equal(mediaBlocks(css, '(prefers-reduced-motion: reduce)').length, 5);
});

test('consolidated reduced-motion protections remain present', async () => {
  const css = await readCss();
  const reducedMotionBlocks = mediaBlocks(css, '(prefers-reduced-motion: reduce)');

  assert.ok(
    reducedMotionBlocks.some((block) => block.includes('.quiet-doorway__media') && block.includes('transition: none')),
    'quiet doorway parallax media should still disable transitions for reduced motion'
  );
  assert.ok(
    reducedMotionBlocks.some((block) => block.includes('.post-topic-chip::after') && block.includes('.tag-back-link::after') && block.includes('transition: none')),
    'tag underline motion should still disable transitions for reduced motion'
  );
});

test('shared PR3 design tokens replace repeated literal values', async () => {
  const css = await readCss();

  assert.match(css, /--border-hairline:\s*1px solid var\(--line-hairline\);/);
  assert.match(css, /--motion-line:\s*320ms var\(--ease-editorial\);/);

  const oldHairlineBorders = css.match(/border(?:-(?:top|right|bottom|left|block|inline))?:\s*1px solid var\(--line-hairline\);/g) ?? [];
  assert.deepEqual(oldHairlineBorders, [], 'hairline borders should use --border-hairline');

  const oldLineMotion = css.match(/transition:\s*transform 320ms var\(--ease-editorial\);/g) ?? [];
  assert.deepEqual(oldLineMotion, [], 'line transform transitions should use --motion-line');
});

test('confirmed PR5 dead selectors stay removed', async () => {
  const css = await readCss();
  const cssWithoutComments = css.replace(/\/\*[\s\S]*?\*\//g, '');
  const removedSelectors = [
    'hero-grid',
    'hero-copy',
    'hero-actions',
    'btn',
    'btn-primary',
    'btn-secondary',
    'hero-panel',
    'today-card',
    'latest-note',
    'hero-stats',
    'term-cloud',
    'post-grid',
    'resource-card',
    'resource-grid',
    'archive-list',
    'about-grid',
    'article-hero',
    'article-title-card',
    'article-layout',
    'search-box',
    'search-result__terms',
    'issue-strip',
    'latest-stack',
    'editorial-index-grid',
    'field-error-note',
    'form-annotation'
  ];

  for (const selector of removedSelectors) {
    assert.equal(
      new RegExp(`\\.${selector.replaceAll('-', '\\-')}(?![\\w-])`).test(cssWithoutComments),
      false,
      `${selector} should remain removed`
    );
  }
});
