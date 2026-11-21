// Funscript file helpers shared by the scene card, the scene details and the player.

// AI-generated scripts are tagged in the file name with "ai" as a separate word:
// "clip_ai.funscript", "clip_ai2", "clip_ai_v2", "clip.ai", "clip - AI", "clip (AI)".
// "ai" inside a word ("Kaia", "rain") does not count.
const AI_SCRIPT_RE = /(^|[\s._\-()[\]])ai(\d+)?(?=$|[\s._\-()[\]])/i

export function isAiScript (filename) {
  if (!filename) return false
  return AI_SCRIPT_RE.test(filename.replace(/\.funscript$/i, ''))
}
