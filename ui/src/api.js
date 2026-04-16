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
