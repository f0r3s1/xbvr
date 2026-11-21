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
