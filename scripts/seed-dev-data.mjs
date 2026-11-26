#!/usr/bin/env bun
// Seeds a local XBVR instance with safe-for-work dummy data for UI work and
// Playwright checks. Run through `make local-seed` (docker); it expects:
//   - XBVR reachable at XBVR_URL (default http://host.docker.internal:9999)
//   - the instance's media volume mounted at MEDIA_DIR (default /media),
//     which XBVR itself sees at the same path.
//
// What you get (deterministic, re-runnable):
//   - SCENES fictional scenes (default 240) from 8 studios, PEOPLE people (default 48, plus five
//     near-duplicate names for the duplicate actors review), neutral tags,
//     landscape photos from picsum.photos as covers/galleries
//   - a scanned storage folder containing:
//       4 linked videos      file name matches a scene exactly (one with a hand
//                            .funscript and an _ai.funscript to compare, one with
//                            an _ai.funscript from a newer generator version)
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
// PEOPLE (default 48) scales the cast for load tests. The first 48 names are the
// original demo cast; after that every first/last pair, then pairs with a middle
// initial, so all names stay unique.
const peopleCount = parseInt(process.env.PEOPLE || '48', 10)
const people = [...new Set(Array.from({ length: Math.min(peopleCount, 48) }, (_, i) => `${firstNames[i % firstNames.length]} ${lastNames[(i * 7) % lastNames.length]}`))]
for (let k = 0, seen = new Set(people); people.length < peopleCount; k++) {
  const pairs = firstNames.length * lastNames.length
  const tier = Math.floor(k / pairs)
  const first = firstNames[k % firstNames.length]
  const last = lastNames[Math.floor(k / firstNames.length) % lastNames.length]
  const name = tier ? `${first} ${String.fromCharCode(65 + (tier - 1) % 26)}. ${last}` : `${first} ${last}`
  if (!seen.has(name)) { seen.add(name); people.push(name) }
}
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
// Unique stock photos: picsum's seed URLs collide (different seeds, same photo),
// so pull the catalogue once and hand out distinct ids. Order is shuffled with
// the deterministic PRNG so runs stay reproducible.
const photoIds = []
for (let page = 1; page <= 6; page++) {
  const res = await fetch(`https://picsum.photos/v2/list?page=${page}&limit=100`)
  if (!res.ok) throw new Error(`picsum catalogue: HTTP ${res.status}`)
  photoIds.push(...(await res.json()).map(p => p.id))
}
const photos = pickN(photoIds, photoIds.length)
const img = (n, w, h) => `https://picsum.photos/id/${photos[n % photos.length]}/${w}/${h}`
const PERSON_PHOTOS = 0 // ids 0..47 people, 48.. scene covers, galleries reuse the rest
const pad = n => String(n).padStart(4, '0')
const peopleDetails = Object.fromEntries(people.map((p, i) => [p, { ImageUrl: img(PERSON_PHOTOS + i, 400, 600), ProfileUrl: '', Source: 'seed' }]))

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
    covers: [img(people.length + i, 1200, 675)],
    gallery: Array.from({ length: 4 + Math.floor(rand() * 5) }, () => img(Math.floor(rand() * photos.length), 1200, 800)),
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

// Duplicate people for the Library Health "duplicate actors" review: the same
// person recorded under a slightly different name, one variant per detection rule.
// They replace the cast of the last few scenes (no extra randomness used).
const duplicateVariants = [
  people[0].toLowerCase(), // case only            -> 100%
  people[2].replace('a', 'á'), // accent            -> 100%
  people[4].replace(' ', ''), // no space           -> 97%
  people[1].split(' ').reverse().join(' '), // order -> 95%
  people[3] + 'e', // one letter more              -> similar spelling
]
duplicateVariants.forEach((name, k) => {
  const sc = scenes[scenes.length - 1 - k]
  sc.cast = [name]
  sc.actor_details = { [name]: { ImageUrl: img(PERSON_PHOTOS + [0, 2, 4, 1, 3][k], 400, 600), ProfileUrl: '', Source: 'seed' } }
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

// A plausible random stroke pattern covering the sample clip, with funscript
// metadata (creator, generator description, chapters) for the UI to display.
// `tempo` is the average ms between turning points, `depth` the stroke size (0-1).
const chapters = [
  { name: 'Intro', startTime: '00:00:00.000', endTime: '00:00:03.000' },
  { name: 'Fast', startTime: '00:00:03.000', endTime: '00:00:07.000' },
  { name: 'Slow', startTime: '00:00:07.000', endTime: '00:00:10.000' },
]
const funscript = ({ durationMs = 10_000, tempo = 520, depth = 0.8, title = '', creator = 'demo', description = '', license = '' } = {}) => {
  const actions = []
  let at = 0
  let up = true
  const lo = Math.round(50 - depth * 50)
  const hi = Math.round(50 + depth * 50)
  while (at <= durationMs) {
    const jitter = Math.floor(rand() * 15)
    actions.push({ at, pos: up ? hi - jitter : lo + jitter })
    up = !up
    at += Math.round(tempo * (0.6 + rand() * 0.8))
  }
  return JSON.stringify({
    version: '1.0', inverted: false, range: 100,
    metadata: { creator, title, description, license, duration: Math.round(durationMs / 1000), chapters, tags: [], performers: [] },
    actions,
  })
}

// ── 1. scenes ───────────────────────────────────────────────────────
// The restore handler sniffs for the literal `"bundleVersion":"1"`, so keep the JSON compact.
// Upcoming announcements: release dates ahead of today, shown faded with an
// "Upcoming · in N days" badge and limited by the Web UI "Releasing within" setting.
const now = Date.now()
for (const [k, days] of [2, 6, 11, 19, 33, 58].entries()) {
  const studio = studios[k % studios.length]
  const id = `${studio.scraper}-up${k + 1}`
  scenes.push({
    ...scenes[k],
    _id: id,
    xbvr_site: studio.scraper,
    scene_id: `up${k + 1}`,
    title: `${pick(titleA)} ${pick(titleB)} (Coming Soon)`,
    studio: studio.name,
    site: studio.name,
    filename: [],
    released: new Date(now + days * 86400000).toISOString().slice(0, 10),
    homepage_url: `https://example.com/${studio.scraper}/up${k + 1}`,
    upcoming: true
  })
}

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
const script = (name, opts = {}) => writeFileSync(join(libraryDir, name), funscript({ title: name.replace(/\.funscript$/, ''), ...opts }))
const counts = { linked: 0, funscript: 0, aiFunscript: 0, nearMiss: 0, unrelated: 0, orphanScripts: 0 }
const order = pickN(scenes, scenes.length)
const ref = i => order[i].filename[0]
const base = n => n.replace(/\.mp4$/, '')

// linked: exact file name. The first video has a hand script and an AI script to
// compare (different tempo and depth), the second an AI script from a newer
// generator version, two are video-only.
video(ref(0))
script(`${base(ref(0))}.funscript`, { tempo: 480, depth: 0.9, license: 'Free' }); counts.funscript++
script(`${base(ref(0))}_ai.funscript`, { tempo: 380, depth: 0.7, description: 'AutoScript v1.0', license: 'Paid' }); counts.aiFunscript++
counts.linked++
video(ref(1)); script(`${base(ref(1))}_ai.funscript`, { tempo: 300, depth: 0.6, description: 'AutoScript v1.1', license: 'Paid' }); counts.linked++; counts.aiFunscript++
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

// ── 4. previews ─────────────────────────────────────────────────────
// Rescan runs in the background; once the linked scenes are available, render
// their hover previews (ffmpeg in the container, a few seconds per scene).
const available = async () => (await api('POST', '/api/scene/list', { dlState: 'available', limit: 1, offset: 0 })).results
for (let i = 0; i < 60 && await available() < counts.linked; i++) await sleep(1000)
await api('GET', '/api/task/preview/generate')

// ── 5. birth dates ──────────────────────────────────────────────────
// The bundle format carries no actor details, so two in three actors get a
// birth date through the regular edit endpoint.
const { actors } = await api('POST', '/api/actor/list', { limit: 5000, offset: 0, sort: 'name_asc' })
for (const [i, a] of (actors || []).filter(a => !a.birth_date || a.birth_date.startsWith('0001')).entries()) {
  if (i % 3 === 2) continue
  const full = await api('GET', `/api/actor/${a.id}`)
  const year = 1988 + Math.floor(rand() * 14)
  full.birth_date = `${year}-${pad(1 + Math.floor(rand() * 12)).slice(2)}-${pad(1 + Math.floor(rand() * 28)).slice(2)}T00:00:00Z`
  await api('POST', `/api/actor/edit/${a.id}`, full)
}

console.log(`files:    ${counts.linked} linked, ${counts.nearMiss} near-miss, ${counts.unrelated} unrelated videos`)
console.log(`scripts:  ${counts.funscript} .funscript, ${counts.aiFunscript} _ai.funscript, ${counts.orphanScripts} orphans`)
console.log(`no files: ${scenes.length - linkedCount} scenes`)
console.log('rescan done, preview generation started - hover previews appear within a minute')
