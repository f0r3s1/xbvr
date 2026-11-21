#!/usr/bin/env bun
// Seeds a local XBVR instance with safe-for-work dummy data for UI work and
// Playwright checks. Run through `make local-seed` (docker); it expects:
//   - XBVR reachable at XBVR_URL (default http://host.docker.internal:9999)
//   - the instance's media volume mounted at MEDIA_DIR (default /media),
//     which XBVR itself sees at the same path.
//
// What you get (deterministic, re-runnable):
//   - SCENES fictional scenes (default 240) from 8 studios, ~48 people, neutral tags,
//     landscape photos from picsum.photos as covers/galleries
//   - a scanned storage folder containing:
//       4 linked videos      file name matches a scene exactly (one with a .funscript,
//                            one with an _ai.funscript)
//       1 near-miss video    name that *should* match but doesn't (+ its funscript) -
//                            a test case for the matcher
//       1 unrelated video    belongs to no scene
//       1 orphan funscript   script without a video
//     the remaining scenes have no files at all.
// All videos are hard links to one small CC-licensed sample clip.

import { existsSync, linkSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const baseUrl = (process.env.XBVR_URL || 'http://host.docker.internal:9999').replace(/\/$/, '')
const mediaDir = process.env.MEDIA_DIR || '/media'
const sceneCount = parseInt(process.env.SCENES || '240', 10)
const sampleUrl = process.env.SAMPLE_VIDEO_URL ||
  'https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4'

const libraryDir = join(mediaDir, 'library') // what XBVR scans
const cacheDir = join(mediaDir, '.seed-cache') // outside the scanned folder

// ── deterministic randomness ────────────────────────────────────────
let state = 0x2545f491
const rand = () => {
  state ^= state << 13; state ^= state >>> 17; state ^= state << 5
  return ((state >>> 0) % 1_000_000) / 1_000_000
}
const pick = arr => arr[Math.floor(rand() * arr.length)]
const pickN = (arr, n) => {
  const copy = [...arr]
  const out = []
  while (out.length < n && copy.length) out.push(copy.splice(Math.floor(rand() * copy.length), 1)[0])
  return out
}

// ── fictional catalogue ─────────────────────────────────────────────
const studios = [
  { scraper: 'seed-aurora', name: 'Aurora Studio' },
  { scraper: 'seed-northwind', name: 'Northwind VR' },
  { scraper: 'seed-lumen', name: 'Lumen Works' },
  { scraper: 'seed-tidal', name: 'Tidal Frames' },
  { scraper: 'seed-summit', name: 'Summit 180' },
  { scraper: 'seed-meridian', name: 'Meridian Labs' },
  { scraper: 'seed-cobalt', name: 'Cobalt Pictures' },
  { scraper: 'seed-harbor', name: 'Harbor Light' }
]
const firstNames = ['Ava', 'Mila', 'Nora', 'Iris', 'Luna', 'Cleo', 'Zara', 'Esme', 'Ruby', 'Ivy', 'Maya', 'Lena', 'Ada', 'Skye', 'Tess', 'June', 'Wren', 'Faye', 'Gwen', 'Hana']
const lastNames = ['Rivers', 'Stone', 'Vale', 'Hart', 'Frost', 'Lane', 'Reed', 'Moss', 'Gray', 'Wilde', 'Brooks', 'Snow', 'Quinn', 'Marsh', 'Sloane']
const people = [...new Set(Array.from({ length: 48 }, (_, i) => `${firstNames[i % firstNames.length]} ${lastNames[(i * 7) % lastNames.length]}`))]
const tags = ['Outdoor', 'Beach', 'City Lights', 'Mountain', 'Forest', 'Sunset', 'Studio', 'Rooftop', 'Cozy',
  'Cooking', 'Yoga', 'Dance', 'Travel', 'Road Trip', 'Rainy Day', 'Winter', 'Summer', 'Pool', 'Garden', 'Library']
const titleA = ['Morning', 'Golden', 'Quiet', 'Electric', 'Hidden', 'Endless', 'Lazy', 'Midnight', 'Sunny', 'Velvet', 'Wild', 'Crystal']
const titleB = ['Coast', 'Afternoon', 'Getaway', 'Rooftop', 'Garden', 'Horizon', 'Studio Session', 'Road Trip', 'Cabin', 'Harbor', 'Market', 'Lake House']
const synopsisBits = [
  'A slow, scenic walkthrough filmed in native 8K with a fixed tripod height.',
  'Shot on location over two days with natural light only.',
  'An easy-going session with plenty of ambient sound and long takes.',
  'Features a guided tour, a picnic and a sunset finish.',
  'Includes a short behind-the-scenes segment at the end.',
  'Captured with a stabilized rig to keep motion comfortable in a headset.'
]
const img = (seed, w, h) => `https://picsum.photos/seed/${seed}/${w}/${h}`
const pad = n => String(n).padStart(4, '0')
const peopleDetails = Object.fromEntries(people.map((p, i) => [p, { ImageUrl: img(`person-${i}`, 400, 600), ProfileUrl: '', Source: 'seed' }]))

const start = Date.UTC(2021, 0, 1)
const span = Date.UTC(2026, 10, 1) - start

const scenes = Array.from({ length: sceneCount }, (_, i) => {
  const studio = pick(studios)
  const id = `${studio.scraper}-${pad(i + 1)}`
  const title = `${pick(titleA)} ${pick(titleB)}${rand() < 0.25 ? ' Part ' + (1 + Math.floor(rand() * 3)) : ''}`
  const cast = pickN(people, 1 + Math.floor(rand() * 2.2))
  const slug = title.replace(/[^A-Za-z0-9]+/g, '_')
  const res = pick(['8K', '6K', '5K'])
  return {
    _id: id,
    xbvr_site: studio.scraper,
    scene_id: pad(i + 1),
    scene_type: 'VR',
    title,
    studio: studio.name,
    site: studio.name,
    covers: [img(id, 1200, 675)],
    gallery: Array.from({ length: 4 + Math.floor(rand() * 5) }, (_, g) => img(`${id}-g${g}`, 1200, 800)),
    tags: [...pickN(tags, 2 + Math.floor(rand() * 4)), pick(['180°', '360°']), res],
    cast,
    filename: [`${studio.name.replace(/\s+/g, '')}_${slug}_${pad(i + 1)}_${res}_180x180_3dh.mp4`],
    duration: 15 + Math.floor(rand() * 55),
    synopsis: `${pick(synopsisBits)} ${pick(synopsisBits)}`,
    released: new Date(start + rand() * span).toISOString().slice(0, 10),
    homepage_url: `https://example.com/${studio.scraper}/${pad(i + 1)}`,
    members_url: '',
    trailer_type: '',
    trailer_source: '',
    chromakey: '',
    has_script_Download: rand() < 0.3,
    ai_script: rand() < 0.15,
    human_script: rand() < 0.2,
    actor_details: Object.fromEntries(cast.map(p => [p, peopleDetails[p]]))
  }
})

// ── helpers ─────────────────────────────────────────────────────────
const api = async (method, path, body) => {
  const res = await fetch(`${baseUrl}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined
  })
  if (!res.ok) throw new Error(`${method} ${path} → HTTP ${res.status} ${await res.text()}`)
  const text = await res.text()
  return text ? JSON.parse(text) : null
}
const sleep = ms => new Promise(r => setTimeout(r, ms))
const sceneTotal = async () => (await api('POST', '/api/scene/list', { dlState: 'any', limit: 1, offset: 0 })).results

// A plausible random stroke pattern covering the sample clip.
const funscript = (durationMs = 10_000) => {
  const actions = []
  let at = 0
  let up = true
  while (at <= durationMs) {
    actions.push({ at, pos: up ? 60 + Math.floor(rand() * 40) : Math.floor(rand() * 40) })
    up = !up
    at += 250 + Math.floor(rand() * 550)
  }
  return JSON.stringify({ version: '1.0', inverted: false, range: 100, actions })
}

// ── 1. scenes ───────────────────────────────────────────────────────
// The restore handler sniffs for the literal `"bundleVersion":"1"`, so keep the JSON compact.
const bundle = JSON.stringify({ timestamp: new Date().toISOString(), bundleVersion: '1', scenes })
await api('POST', '/api/task/bundle/restore', { uploadData: bundle })
for (let i = 0; i < 120 && (await sceneTotal()) < sceneCount; i++) await sleep(500)
console.log(`scenes:   ${await sceneTotal()} in library`)

// ── 2. media files ──────────────────────────────────────────────────
mkdirSync(cacheDir, { recursive: true })
const sample = join(cacheDir, 'sample.mp4')
if (!existsSync(sample)) {
  const res = await fetch(sampleUrl)
  if (!res.ok) throw new Error(`sample video download failed: HTTP ${res.status}`)
  writeFileSync(sample, Buffer.from(await res.arrayBuffer()))
}

rmSync(libraryDir, { recursive: true, force: true })
mkdirSync(libraryDir, { recursive: true })
const video = name => linkSync(sample, join(libraryDir, name))
const script = name => writeFileSync(join(libraryDir, name), funscript())
const counts = { linked: 0, funscript: 0, aiFunscript: 0, nearMiss: 0, unrelated: 0, orphanScripts: 0 }
const order = pickN(scenes, scenes.length)
const ref = i => order[i].filename[0]
const base = n => n.replace(/\.mp4$/, '')

// linked: exact file name; one plain funscript, one *_ai funscript, two video-only
video(ref(0)); script(`${base(ref(0))}.funscript`); counts.linked++; counts.funscript++
video(ref(1)); script(`${base(ref(1))}_ai.funscript`); counts.linked++; counts.aiFunscript++
video(ref(2)); counts.linked++
video(ref(3)); counts.linked++

// near-miss: a human would pair it with the scene, the exact-name matcher does not
const nearMiss = ref(4).replace(/_(8K|6K|5K)_/, '_4K_').replace(/\.mp4$/, '_ai.mp4')
video(nearMiss); script(`${base(nearMiss)}.funscript`); counts.nearMiss++; counts.funscript++

// unrelated video + orphan script
video('GOPR0042.mp4'); counts.unrelated++
script('orphan_script.funscript'); counts.orphanScripts++
const linkedCount = counts.linked + counts.nearMiss

// ── 3. storage folder + rescan ──────────────────────────────────────
const { volumes } = await api('GET', '/api/options/storage')
if (!(volumes || []).some(v => v.path === libraryDir)) {
  await api('POST', '/api/options/storage', { type: 'local', path: libraryDir })
}
await api('GET', '/api/task/rescan')

console.log(`files:    ${counts.linked} linked, ${counts.nearMiss} near-miss, ${counts.unrelated} unrelated videos`)
console.log(`scripts:  ${counts.funscript} .funscript, ${counts.aiFunscript} _ai.funscript, ${counts.orphanScripts} orphans`)
console.log(`no files: ${scenes.length - linkedCount} scenes`)
console.log('rescan started - files will appear linked within a few seconds')
