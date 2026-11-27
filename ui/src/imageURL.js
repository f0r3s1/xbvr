// remote images go through the server's resizing cache; local paths are used as they are
export function imageURL (url, size = '1200x') {
  if (!url || !url.startsWith('http')) return url
  try {
    return `/img/${size}/` + encodeURI(url.includes('%') ? decodeURI(url) : url)
  } catch {
    return `/img/${size}/` + url
  }
}
