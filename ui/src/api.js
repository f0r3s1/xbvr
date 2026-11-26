import ky from 'ky'

// Shared client: retries POST (list endpoints are idempotent), network errors during boot, and 5xx/429
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

// every options store asks for /api/options/state on load; callers within a second share one request
let optionsStateInFlight = null
export function getOptionsState () {
  if (!optionsStateInFlight) {
    optionsStateInFlight = api.get('/api/options/state').json()
      .finally(() => setTimeout(() => { optionsStateInFlight = null }, 1000))
  }
  return optionsStateInFlight
}

// alternate source lookups from many cards within a short window are merged into one batched request
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
