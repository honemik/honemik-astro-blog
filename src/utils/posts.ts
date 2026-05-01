import { getCollection, type CollectionEntry } from 'astro:content';

export type BlogPost = CollectionEntry<'posts'>;

let postsCache: BlogPost[] | undefined;

export async function getAllPosts() {
  if (import.meta.env.DEV || !postsCache) {
    postsCache = (await getCollection('posts', ({ data }) => !data.draft))
      .sort((a, b) => b.data.date.getTime() - a.data.date.getTime());
  }

  return [...postsCache];
}

export function formatDate(date: Date) {
  return new Intl.DateTimeFormat('en', {
    year: 'numeric',
    month: 'short',
    day: '2-digit'
  }).format(date);
}

export function readingMinutes(post: BlogPost) {
  const plain = markdownToPlainText(post.body, {
    linkPattern: /\[[^\]]*]\([^)]*\)/g,
    markPattern: /[>#*_`~\-]/g
  });
  const latinWords = plain.match(/[A-Za-z0-9]+/g)?.length ?? 0;
  const cjkChars = plain.match(/[\u3400-\u9fff]/g)?.length ?? 0;
  const estimatedWords = latinWords + Math.ceil(cjkChars / 2);
  return Math.max(1, Math.ceil(estimatedWords / 220));
}

export function getExcerpt(post: BlogPost, maxLength = 150) {
  const source = post.data.description || post.body;
  const text = markdownToPlainText(source, {
    linkPattern: /\[[^\]]*]\(([^)]*)\)/g,
    markPattern: /[#>*_`~\-]/g
  })
    .replace(/\s+/g, ' ')
    .trim();

  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength).trim()}...`;
}

export function getPostCover(post: BlogPost) {
  const match = post.body.match(/!\[[^\]]*]\((\/img\/[^)\s]+)[^)]*\)/);
  return match?.[1] ?? '';
}

export async function getPostStaticPaths() {
  const posts = await getAllPosts();
  return posts.map((post, index) => ({
    params: { slug: post.slug },
    props: {
      post,
      newer: posts[index - 1],
      older: posts[index + 1]
    }
  }));
}

export function getTags(posts: BlogPost[]) {
  return countTerms(posts.flatMap((post) => post.data.tags));
}

export function getCategories(posts: BlogPost[]) {
  return countTerms(posts.flatMap((post) => post.data.categories));
}

export function countTerms(terms: string[]) {
  const counts = new Map<string, number>();
  for (const term of terms.filter(Boolean)) {
    counts.set(term, (counts.get(term) ?? 0) + 1);
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
}

export function termSlug(term: string) {
  return term.trim().toLowerCase().replace(/\s+/g, '-');
}

export function groupPostsByYear(posts: BlogPost[]) {
  const groups = new Map<string, BlogPost[]>();
  for (const post of posts) {
    const year = String(post.data.date.getFullYear());
    groups.set(year, [...(groups.get(year) ?? []), post]);
  }
  return [...groups.entries()].sort((a, b) => Number(b[0]) - Number(a[0]));
}

function markdownToPlainText(
  value: string,
  {
    linkPattern,
    markPattern
  }: {
    linkPattern: RegExp;
    markPattern: RegExp;
  }
) {
  return value
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/!\[[^\]]*]\([^)]*\)/g, ' ')
    .replace(linkPattern, ' ')
    .replace(markPattern, ' ');
}
