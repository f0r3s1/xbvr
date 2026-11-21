// Funscript file helpers shared by the scene card, the scene details and the player.

// AI-generated scripts are tagged in the file name with "ai" as a separate word:
// "clip_ai.funscript", "clip_ai2", "clip_ai_v2", "clip.ai", "clip - AI", "clip (AI)".
// "ai" inside a word ("Kaia", "rain") does not count.
const AI_SCRIPT_RE = /(^|[\s._\-()[\]])ai(\d+)?(?=$|[\s._\-()[\]])/i

export function isAiScript (filename) {
  if (!filename) return false
  return AI_SCRIPT_RE.test(filename.replace(/\.funscript$/i, ''))
}

// Generators write "<Name> v<version>" into metadata.description ("Tool v1.0").
// The field is free text, so the rule is strict: a "v" followed by one to three
// dot-separated integers (v1, v1.0, v1.0.0) and nothing glued after it, and the
// word right before it is the engine name. Anything else is not a generator.
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
