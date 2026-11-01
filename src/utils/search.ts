import { getAllPosts, getTags, termSlug } from './posts';

export async function getSearchSuggestions() {
  const posts = await getAllPosts();
  return getTags(posts).slice(0, 4).map(([tag, count]) => ({
    label: tag,
    count,
    url: `/tags/${termSlug(tag)}/`
  }));
}
