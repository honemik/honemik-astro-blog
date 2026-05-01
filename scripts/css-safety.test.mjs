import { readFile } from 'node:fs/promises';
import assert from 'node:assert/strict';
import test from 'node:test';

const cssPath = new URL('../src/styles/global.css', import.meta.url);

async function readCss() {
  return readFile(cssPath, 'utf8');
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
