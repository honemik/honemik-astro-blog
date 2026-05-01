import { formatDate, getAllPosts, getExcerpt, getTags, termSlug } from './posts';

export async function getSearchItems() {
  const posts = await getAllPosts();
  return posts.map((post) => ({
    title: post.data.title,
    date: formatDate(post.data.date),
    description: getExcerpt(post, 140),
    url: `/posts/${post.slug}/`,
    terms: [...post.data.categories, ...post.data.tags].slice(0, 4),
    haystack: `${post.data.title} ${post.data.description} ${post.data.tags.join(' ')} ${post.data.categories.join(' ')} ${post.body}`.toLowerCase()
  }));
}

export async function getSearchSuggestions() {
  const posts = await getAllPosts();
  return getTags(posts).slice(0, 4).map(([tag, count]) => ({
    label: tag,
    count,
    url: `/tags/${termSlug(tag)}/`
  }));
}
