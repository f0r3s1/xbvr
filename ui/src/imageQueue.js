// Card images load at most 4 at a time and only near the screen, leaving the browser's ~6 http connections free for API calls.
const MAX_ACTIVE = 4
const SLOW_MS = 12000
let active = 0
const waiting = []
const jobs = new WeakMap()

function finish (job) {
  if (!job.inFlight) return
  job.inFlight = false
  clearTimeout(job.timer)
  active--
  pump()
}

// a hanging request holds one of the browser's few connections: cancel it, let the card load the original ("queue-slow"), retry once
function slow (job) {
  abort(job)
  job.el.dispatchEvent(new CustomEvent('queue-slow'))
  if (!job.retried) {
    job.retried = true
    enqueue(job)
  }
}

// an image that scrolled away gives its slot back and is queued again when visible
function abort (job) {
  if (!job.inFlight) return
  job.el.removeAttribute('src')
  finish(job)
}

function pump () {
  while (active < MAX_ACTIVE && waiting.length) {
    const job = waiting.shift()
    job.queued = false
    if (!job.el.isConnected) continue
    job.inFlight = true
    active++
    job.timer = setTimeout(() => slow(job), SLOW_MS)
    job.el.src = job.src
  }
}

function enqueue (job) {
  if (job.queued || job.inFlight || job.loadedSrc === job.src) return
  job.queued = true
  waiting.push(job)
  pump()
}

function dequeue (job) {
  if (!job.queued) return
  job.queued = false
  const i = waiting.indexOf(job)
  if (i !== -1) waiting.splice(i, 1)
}

const observer = typeof IntersectionObserver === 'undefined' ? null : new IntersectionObserver(entries => {
  for (const entry of entries) {
    const job = jobs.get(entry.target)
    if (!job) continue
    job.visible = entry.isIntersecting
    if (job.visible) {
      enqueue(job)
    } else {
      dequeue(job)
      abort(job)
    }
  }
}, { rootMargin: '400px 0px' })

// images on other servers do not compete for our connections and load directly
const isOwnServer = src => typeof src === 'string' && src.startsWith('/')

function setSource (el, src) {
  const job = jobs.get(el)
  if (job.src === src) return
  dequeue(job)
  abort(job)
  job.src = src
  job.retried = false
  if (!src) {
    el.removeAttribute('src')
  } else if (!isOwnServer(src) || !observer) {
    el.src = src
    job.loadedSrc = src
  } else if (job.visible) {
    enqueue(job)
  }
}

export default {
  mounted (el, binding) {
    const job = { el, src: null, queued: false, inFlight: false, visible: false, loadedSrc: null }
    el.addEventListener('load', () => {
      if (el.getAttribute('src') === job.src) job.loadedSrc = job.src
      finish(job)
    })
    el.addEventListener('error', () => finish(job))
    jobs.set(el, job)
    setSource(el, binding.value)
    observer?.observe(el)
  },
  updated (el, binding) {
    setSource(el, binding.value)
  },
  unmounted (el) {
    const job = jobs.get(el)
    if (!job) return
    observer?.unobserve(el)
    dequeue(job)
    abort(job)
  }
}
