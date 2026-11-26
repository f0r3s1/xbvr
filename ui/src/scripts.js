
// AI scripts carry "ai" as a separate word in the file name ("clip_ai2", "clip (AI)"), not inside one ("Kaia")
const AI_SCRIPT_RE = /(^|[\s._\-()[\]])ai(\d+)?(?=$|[\s._\-()[\]])/i

export function isAiScript (filename) {
  if (!filename) return false
  return AI_SCRIPT_RE.test(filename.replace(/\.funscript$/i, ''))
}

// generators write "<Name> v<1-3 dotted numbers>" into metadata.description; anything looser is not a generator
const GENERATOR_RE = /(?:^|[\s([])([A-Za-z][A-Za-z0-9_-]*)\s+v(\d+(?:\.\d+){0,2})(?=$|[\s,;)\]])/

export function scriptGenerator (description) {
  const m = GENERATOR_RE.exec(String(description || ''))
  return m ? { name: m[1], version: m[2] } : null
}

// metadata.license: only the two well-known values are shown.
export function scriptLicense (license) {
  const l = String(license || '').trim().toLowerCase()
  return l === 'paid' || l === 'free' ? l : null
}
