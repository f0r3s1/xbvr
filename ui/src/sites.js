import ky from 'ky'

let request = null

export function loadSites () {
  request = request || ky.get('/api/scene/sites').json().catch(() => { request = null; return [] })
  return request
}

export function siteLogoURL (u) {
  if (!u) return ''
  return u.startsWith('http') ? '/img/200x/' + u.replace('://', ':/') : u
}

export function siteInitials (name) {
  return name.replace(/\(.*?\)/g, '').split(/\s+/).filter(Boolean).slice(0, 2).map(w => w[0]).join('').toUpperCase()
}

export function siteHue (name) {
  let h = 0
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) % 360
  return h
}

export function siteFilterQuery (store, name) {
  const filters = { ...store.state.sceneList.filters, sites: [name], sort: 'release_desc', dlState: 'any', isAvailable: null, isAccessible: null, isHidden: false }
  return { name: 'scenes', query: { q: btoa(unescape(encodeURIComponent(JSON.stringify(filters)))) } }
}

const colorCache = {}

// Dominant saturated colour of a logo as [hue, saturation], read from the same-origin image proxy.
export function logoColor (url) {
  if (!url) return Promise.resolve(null)
  colorCache[url] = colorCache[url] || new Promise(resolve => {
    const img = new Image()
    img.onload = () => {
      const size = 24
      const canvas = document.createElement('canvas')
      canvas.width = canvas.height = size
      const ctx = canvas.getContext('2d', { willReadFrequently: true })
      ctx.drawImage(img, 0, 0, size, size)
      const data = ctx.getImageData(0, 0, size, size).data
      const buckets = {}
      for (let i = 0; i < data.length; i += 4) {
        const [r, g, b, a] = [data[i] / 255, data[i + 1] / 255, data[i + 2] / 255, data[i + 3]]
        const max = Math.max(r, g, b)
        const min = Math.min(r, g, b)
        const sat = max === 0 ? 0 : (max - min) / max
        if (a < 128 || sat < 0.35 || max < 0.25) continue
        const d = max - min
        let h = max === r ? ((g - b) / d) % 6 : max === g ? (b - r) / d + 2 : (r - g) / d + 4
        h = Math.round(((h * 60) + 360) % 360 / 10) * 10
        buckets[h] = buckets[h] || { n: 0, sat: 0 }
        buckets[h].n++
        buckets[h].sat += sat
      }
      const best = Object.entries(buckets).sort((a, b) => b[1].n - a[1].n)[0]
      resolve(best && best[1].n > 8 ? [Number(best[0]), Math.round(Math.min(90, (best[1].sat / best[1].n) * 100))] : null)
    }
    img.onerror = () => resolve(null)
    img.src = siteLogoURL(url)
  })
  return colorCache[url]
}
