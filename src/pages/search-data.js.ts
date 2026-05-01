import { getSearchItems } from '../utils/search';

export const prerender = true;

export async function GET() {
  const searchItems = await getSearchItems();
  return new Response(`window.__HONEMIK_SEARCH_DATA__ = ${JSON.stringify(searchItems)};\n`, {
    headers: {
      'Content-Type': 'application/javascript; charset=utf-8',
      'Cache-Control': 'public, max-age=3600'
    }
  });
}
