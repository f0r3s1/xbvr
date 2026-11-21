import ky from 'ky'

// Shared API client with retry + sensible defaults.
// - Retries POST (our list endpoints are idempotent).
// - Retries on network errors (backend not up yet during dev boot).
// - Retries on 5xx/429.
const api = ky.extend({
  timeout: 30000,
  retry: {
    limit: 8,
    methods: ['get', 'post', 'put', 'delete', 'patch', 'head'],
    statusCodes: [408, 413, 429, 500, 502, 503, 504],
    backoffLimit: 3000,
  },
})

export default api

// /api/options/state is requested by every options store and several views at
// once on page load. Share one in-flight request between concurrent callers
// (no caching beyond that, so pollers still get fresh data).
let optionsStateInFlight = null
export function getOptionsState () {
  if (!optionsStateInFlight) {
    optionsStateInFlight = api.get('/api/options/state').json()
      .finally(() => { optionsStateInFlight = null })
  }
  return optionsStateInFlight
}

// Scene cards each need their alternate sources. Requests made within the same
// short window are merged into one /api/scene/alternate_sources?ids=... call, so a
// grid of 80 cards costs one request instead of 80.
let altQueue = new Map()
let altTimer = null
function flushAlternateSources () {
  const batch = altQueue
  altQueue = new Map()
  altTimer = null
  const ids = [...batch.keys()]
  for (let i = 0; i < ids.length; i += 200) {
    const chunk = ids.slice(i, i + 200)
    api.get('/api/scene/alternate_sources', { searchParams: { ids: chunk.join(',') } }).json()
      .then(res => chunk.forEach(id => batch.get(id).forEach(p => p.resolve(res[id] || null))))
      .catch(err => chunk.forEach(id => batch.get(id).forEach(p => p.reject(err))))
  }
}
export function getAlternateSources (sceneId) {
  return new Promise((resolve, reject) => {
    const key = String(sceneId)
    if (!altQueue.has(key)) altQueue.set(key, [])
    altQueue.get(key).push({ resolve, reject })
    if (!altTimer) altTimer = setTimeout(flushAlternateSources, 20)
  })
}
