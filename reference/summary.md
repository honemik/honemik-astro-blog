# Reference Design Summary

This summary condenses the three research files in `reference/` and the seven image references (`01.png` through `07.png`). I am treating the images as the moodboard/photo reference set.

## Source Materials

- `deep-research-report.md`: Broad research on Japanese fashion magazine-inspired web design.
- `deep-research-report_website.md`: Prior evaluation of the honemik blog against Japanese editorial aesthetics.
- `Gemini_search.txt`: Additional research on Japanese editorial web architecture, typography, density, and interaction.
- `01.png`: A honemik-specific target mockup with a stronger editorial homepage structure.
- `02.png` to `07.png`: Moodboard examples showing fashion, lifestyle, vertical text, full-bleed photography, product/editorial landing pages, and atmospheric Japanese web layouts.

## Core Reference Principles

1. **Magazine feeling comes from page architecture, not decoration.** The research repeatedly says the look depends on grid, line length, margins, hierarchy, and content departments. Japanese text, vertical labels, cream backgrounds, or handwritten accents are only useful when the underlying layout is disciplined.

2. **Use departments instead of a generic page stack.** Strong examples organize content like a publication: feature, latest notes, archive, tags, departments, rankings, interviews, lookbooks, special pages, or issue sections. The user should feel they are browsing an edited issue, not only a blog list.

3. **Build a strict grid, then break it deliberately.** The reference favors 12-column or spread-like structures with controlled offsets, side rails, vertical markers, image/text asymmetry, and bento modules. Random collage is not the goal; controlled imbalance is.

4. **Alternate compression and release.** Japanese editorial rhythm moves between quiet image-led moments and dense information blocks. A full image, a compact index strip, a feature block, a tag grid, and a small update list can coexist if each has a clear role.

5. **Typography must have more roles.** Current magazine references rely on a layered type system: display titles, section tags, dates, captions, credits, body text, side notes, ranking numbers, issue numbers, and micro labels. These layers create secondary reading paths.

6. **Vertical text is an accent.** Tategaki or vertical labels should be short: issue labels, side notes, season markers, dates, or navigation accents. It should not carry long body content or critical mobile UI.

7. **Japanese design can be dense and still clear.** The research warns against the "minimalism trap." Removing too much information can make a page feel empty or underdeveloped. The goal is scannable density, not clutter.

8. **Photography should establish atmosphere and evidence.** The moodboard uses images as the primary emotional anchor. Images are not merely thumbnails; they create season, material, lifestyle, and editorial pacing. Captions and credits help the images feel intentionally published.

9. **Micro-details should be functional.** Captions, metadata rows, section tags, numbered markers, hairline rules, archive dates, and small descriptions should clarify reading flow. Decorative stickers or handwritten text should be used sparingly.

10. **Mobile should preserve the rhythm, not copy the desktop layout literally.** Side rails become horizontal chips, vertical labels shorten or rotate back to horizontal, and each viewport should have one clear focal point.

## Moodboard Image Notes

- `01.png`: The strongest direct target for honemik. It uses a left metadata rail, editorial hero text, a large calm image, an issue strip, a bento-style latest-notes area, a departments list, compact archive/tag/update modules, and a final image block. It feels more like an issue homepage than the current site.
- `02.png`: Playful fashion campaign style. Large photography, seasonal masthead, sparse copy, soft background, and decorative sticker-like accents. Useful only if honemik wants a youthful playful layer.
- `03.png`: Narrow editorial spread with oversized type, image offset, red accent, and lots of negative space. Good reference for controlled asymmetry and strong type hierarchy.
- `04.png`: Minimal Japanese layout with vertical navigation, wide image strips, ink-like accents, and very calm spacing. Useful for side labels and image-as-landscape rhythm.
- `05.png`: Product/lifestyle landing page with huge image, sparse navigation, vertical Japanese type, and bold lower typography. Useful for dramatic image scale and whitespace.
- `06.png`: Full-bleed food/nature hero with logo, menu, and vertical copy over photography. Useful for atmospheric hero treatment.
- `07.png`: Soft wellness layout with abstract background shapes, vertical side rail, large English display type, and centered Japanese copy. Useful for calm brand storytelling and side navigation.

## Implications For honemik

The best direction is not to make honemik look like a fashion store. The stronger fit is a **personal editorial diary / small magazine archive**: quiet, warm, image-led, but with clearer departments and more intentional information density.

The site should keep its soft personal feeling, but strengthen:

- a clearer issue-like homepage structure;
- a better balance of large feature, compact index, and archive modules;
- less equal-card repetition;
- more caption, meta, and section-label roles;
- more deliberate asymmetry;
- better mobile handling for vertical labels;
- fewer oversized rounded cards and playful UI badges where the reference wants flatter editorial composition.

## Design Checklist

- Use a 12-column desktop grid and a simpler single-column mobile rhythm.
- Give the homepage an editorial cover structure: issue rail, feature text, large image, metadata, caption.
- Add an issue strip or department index directly after the hero.
- Turn latest posts into a bento module: one large feature, several smaller notes.
- Turn categories/tags into departments with descriptions, not only pills.
- Replace generic placeholders with better editorial fallbacks or text-first compact rows.
- Add captions to meaningful images.
- Use vertical text only for short desktop accents.
- Use CJK-readable line lengths and generous body line-height.
- Validate desktop and mobile screenshots before implementation is considered done.
