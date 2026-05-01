# CSS Refactor Plan

## Current State

- Source CSS: `src/styles/global.css`
- Size: about 98.3KB source, 80.7KB built CSS
- Built compression: about 14.1KB gzip, 12.2KB brotli
- Structure: one large global file with several layered redesign passes
- Main risk: cascade order is important; moving rules across sections may change UI

## 1. CSS Refactor Opportunities

### 1. Repeated redesign layers
- Issue: `:root`, `body`, `.nav-link`, `.quiet-cover`, `.quiet-cover__media` appear across multiple later override sections.
- Impact: global theme, homepage hero, navigation, motion.
- Risk: High.
- Estimated reduction: 150-350 lines, 3-8KB raw CSS.
- Note: Do not reorder yet. First consolidate only within the same visual area.

### 2. Repeated media queries
- Issue: `prefers-reduced-motion` appears 7 times, `max-width: 920px` 5 times, `max-width: 680px` 4 times.
- Impact: responsive layout and reduced-motion behavior.
- Risk: Medium to High.
- Estimated reduction: 80-180 lines, 1.5-4KB raw CSS.
- Note: Merge only when selector order and cascade result are identical.

### 3. Legacy or suspected unused CSS
- Issue: about 87 class selectors are not found in current Astro/TS/MD sources.
- Examples: `.hero-grid`, `.btn`, `.resource-card`, `.editorial-cover`, `.issue-strip`, `.latest-stack`, `.post-grid`, `.term-cloud`.
- Impact: old homepage/resource/editorial layouts.
- Risk: High.
- Estimated reduction: 800-1500 lines, 15-30KB raw CSS.
- Note: Do not delete directly. Confirm with route screenshots and class search first.

### 4. Repeated declaration patterns
- Issue: many repeated values: `display: grid`, `color: var(--color-muted)`, `font-family: var(--font-serif)`, `border-bottom: 1px solid var(--line-hairline)`, common gaps and font sizes.
- Impact: readability, design consistency.
- Risk: Medium.
- Estimated reduction: 50-150 lines if grouped carefully.
- Note: Tokens improve maintainability more than bundle size.

### 5. Selector complexity and specificity
- Issue: selectors like `html.motion-ready .magazine-settle.is-visible > :nth-child(2)` and `body.is-navigating #main-content` have high specificity.
- Impact: motion system and page transitions.
- Risk: High.
- Estimated reduction: small, mostly maintainability.
- Note: Avoid changing these until motion behavior has screenshot/browser checks.

### 6. Over-global base overrides
- Issue: broad global rules such as `a:hover`, `button, input`, repeated `body`, and repeated `:root` make later overrides harder to reason about.
- Impact: whole site.
- Risk: Medium to High.
- Estimated reduction: small initially.
- Note: Later phase should separate tokens/base/components/motion.

### 7. Utilities
- Issue: repeated small patterns could become utilities, but adding utility classes would require DOM changes.
- Impact: broad.
- Risk: High under current constraints.
- Estimated reduction: not recommended in phase 1.
- Note: Since DOM cannot change, prefer grouped selectors over new utility classes.

## 2. Recommended Refactor Order

1. Create baseline checks
   - Build CSS size before/after.
   - Screenshot key routes: home, posts index, article, archive, tags, search.
   - Check desktop and mobile.

2. PR 1: smallest safe cleanup
   - Remove only clearly redundant duplicate responsive declarations.
   - Do not move large sections.
   - Do not remove suspected dead CSS.

3. PR 2: media query consolidation
   - Merge reduced-motion blocks only when declarations are independent.
   - Merge repeated `max-width` blocks by breakpoint, preserving order.

4. PR 3: token cleanup
   - Normalize repeated spacing, font-size, transition, and border values.
   - Keep visual values unchanged.

5. PR 4: component-area consolidation
   - Refactor one area at a time:
     - Header/nav
     - Hero/cover
     - Article
     - Archive/tag
     - Search
     - Footer

6. PR 5: dead CSS removal
   - Remove only classes confirmed unused after source search and screenshot comparison.
   - Keep a removal list in the PR description.

## 3. First Minimal Executable PR

### Title
CSS cleanup pass 1: remove redundant duplicate responsive declarations

### Scope
- Only edit `src/styles/global.css`.
- Remove a tiny number of duplicate or subsumed declarations.
- Do not touch DOM, JS/TS, Astro components, or design tokens.
- Do not remove suspected dead CSS.

### Candidate
- Review duplicated `.footer-social { grid-template-columns: repeat(2, max-content); }` in responsive blocks.
- Remove only if final computed style is identical at affected breakpoints.

### Expected size reduction
- 5-20 lines
- About 0.1-0.5KB raw CSS

### Risk
Low to Medium.

### Verification
- `npm run build`
- Compare home/footer at desktop, tablet, and mobile widths.
- Check `/`, `/posts/`, `/search/`, `/archive/`, `/tags/`.
