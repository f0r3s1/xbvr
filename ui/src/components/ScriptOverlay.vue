<template>
  <div v-if="url" class="script-overlay" @pointerdown.stop @pointerup.stop @click.stop @dblclick.stop>
    <button type="button" class="script-toggle" :class="{ 'is-on': enabled }"
            :title="enabled ? 'Hide script overlay' : 'Show script overlay'" @click="toggle">
      <b-icon pack="mdi" icon="pulse" size="is-small"/>
    </button>

    <div v-show="enabled" ref="panel" class="script-panel" :class="{ 'is-multi': lanes.length > 1, 'is-sized': !!size.h, 'is-compact': compact }"
         :style="panelStyle">
      <div class="script-head" @pointerdown="dragStart">
        <b-icon pack="mdi" icon="drag" size="is-small" class="grip"/>
        <span class="script-title">{{ lanes.length > 1 ? 'Compare' : 'Script' }}</span>
        <b-icon v-if="lanes.length === 1 && lanes[0].ai" pack="mdi" icon="robot-outline" size="is-small" class="script-ai" :title="$t('AI script')"/>
        <span v-if="lanes.length === 1 && options.length > 1" class="script-name" :title="lanes[0].name">{{ lanes[0].label }}</span>
        <span class="script-state">{{ status }}</span>
        <button v-if="options.length > 1" type="button" class="script-menu-btn" :class="{ 'is-open': menuOpen }"
                :title="$t('Choose scripts to show and compare')" @pointerdown.stop @click="toggleMenu">
          <b-icon pack="mdi" icon="tune-variant" size="is-small"/>
        </button>
      </div>

      <!-- checklist: which scripts get a lane; the star is the scene's saved script -->
      <div v-if="menuOpen" ref="menu" class="script-menu" :style="menuStyle" @pointerdown.stop>
        <div v-for="o in options" :key="o.id" class="script-menu-row">
          <label class="script-check" :title="o.name">
            <input type="checkbox" :checked="shown.includes(o.id)" :disabled="shown.includes(o.id) && shown.length === 1"
                   @change="toggleShown(o.id)"/>
            <span class="lane-dot" :style="{ background: shown.includes(o.id) ? laneColour(shown.indexOf(o.id)) : 'transparent' }"></span>
            <span class="script-check-label">{{ o.label }}</span>
            <b-icon v-if="o.ai" pack="mdi" icon="robot-outline" custom-size="mdi-14px" class="script-ai" :title="$t('AI script')"/>
          </label>
          <button type="button" class="script-star" :class="{ 'is-main': o.selected }"
                  :title="o.selected ? 'Selected script for this scene' : 'Use as the scene\'s selected script'"
                  @click="!o.selected && $emit('select', o.id)">
            <b-icon pack="mdi" :icon="o.selected ? 'star' : 'star-outline'" custom-size="mdi-14px"/>
          </button>
        </div>
      </div>

      <div class="script-body">
        <canvas ref="graph" class="script-graph" :style="size.h ? null : { height: graphHeight + 'px' }"></canvas>
      </div>
      <div class="script-resize" :title="$t('Drag to resize, double-click to reset')" @pointerdown.stop="resizeStart" @dblclick.stop="resetSize">
        <b-icon pack="mdi" icon="resize-bottom-right" custom-size="mdi-14px"/>
      </div>
    </div>
  </div>
</template>

<script>
// Draggable funscript overlay for the player; several scripts can share the playhead in lanes to compare them
import { defineComponent } from 'vue'

const STORE_KEY = 'xbvr.scriptOverlay'
// fixed time scale: resizing shows more or less time around the playhead instead of stretching the wave
const PX_PER_SEC = 48
const PAST_SHARE = 0.4 // part of the window before the playhead
// narrower than this the wave is dropped and only the up/down bars remain
const COMPACT_W = 160
const BAR_W = 14
const BAR_GAP = 6
const MAX_LANES = 4
const LANE_COLOURS = ['#9d84ef', '#5aa5e6', '#3fbf86', '#f2a65a']

function loadPrefs () {
  try {
    return JSON.parse(localStorage.getItem(STORE_KEY)) || {}
  } catch {
    return {}
  }
}

// speed in position units per second -> colour (blue, green, yellow, red)
const STOPS = [[0, [62, 142, 208]], [120, [72, 199, 142]], [260, [255, 221, 87]], [400, [241, 70, 104]]]
function speedColour (v) {
  if (v >= STOPS[STOPS.length - 1][0]) return `rgb(${STOPS[STOPS.length - 1][1].join(',')})`
  for (let i = 1; i < STOPS.length; i++) {
    if (v <= STOPS[i][0]) {
      const [a, ca] = STOPS[i - 1]
      const [b, cb] = STOPS[i]
      const k = (v - a) / (b - a)
      return `rgb(${ca.map((c, j) => Math.round(c + (cb[j] - c) * k)).join(',')})`
    }
  }
  return `rgb(${STOPS[0][1].join(',')})`
}

const scriptUrl = id => '/api/dms/file/' + id + '?dnt=true'

export default defineComponent({
  name: 'ScriptOverlay',
  emits: ['select'],

  props: {
    url: { type: String, default: '' }, // main script (used alone when no list is given)
    scripts: { type: Array, default: () => [] }, // [{ id, name, selected, ai, gen }]
    video: { type: Function, required: true } // () => HTMLVideoElement
  },

  data () {
    const prefs = loadPrefs()
    return {
      enabled: prefs.enabled === true,
      pos: { x: prefs.x ?? 3, y: prefs.y ?? 4 },
      // panel size in % of the player (scales with fullscreen); null = automatic
      size: { w: prefs.w ?? null, h: prefs.h ?? null },
      status: '',
      menuOpen: false,
      compact: false, // only the position bars (very small panel)
      menuStyle: {},
      extra: [] // script ids shown next to the main one
    }
  },

  computed: {

    // short labels: drop the extension and the part all names share
    options () {
      const base = this.scripts.map(x => x.name.replace(/\.funscript$/i, ''))
      let pre = base[0] || ''
      for (const b of base) while (pre && !b.startsWith(pre)) pre = pre.slice(0, -1)
      return this.scripts.map((x, i) => {
        const rest = base[i].slice(pre.length).replace(/^[\s._\-\u2013()[\]]+/, '').trim()
        // generator + version when the script names it ("AutoScript v1.0"), else the distinguishing part of the name
        let label = rest
        if (x.gen) label = `${x.gen.name} v${x.gen.version}`
        else if (!rest || /^ai\d*$/i.test(rest)) label = x.ai ? 'AI' + rest.slice(2) : 'Original'
        return { ...x, label }
      })
    },
    mainId () {
      const s = this.scripts.find(x => x.selected)
      return s ? s.id : null
    },
    shown () {
      const ids = this.mainId !== null ? [this.mainId] : []
      for (const id of this.extra) if (!ids.includes(id) && this.scripts.some(s => s.id === id)) ids.push(id)
      return ids.slice(0, MAX_LANES)
    },
    lanes () {
      if (!this.scripts.length) return [{ key: this.url, url: this.url, label: this.$t('Script'), name: '', ai: false }]
      return this.shown.map(id => {
        const o = this.options.find(x => x.id === id)
        return { key: 'f' + id, url: scriptUrl(id), label: o.label, name: o.name, ai: o.ai }
      })
    },
    panelStyle () {
      return {
        left: this.pos.x + '%',
        top: this.pos.y + '%',
        width: this.size.w ? this.size.w + '%' : null,
        height: this.size.h ? this.size.h + '%' : null
      }
    },
    graphHeight () {
      return this.lanes.length > 1 ? this.lanes.length * 40 : 64
    }
  },

  created () {
    this.cache = new Map() // url -> { actions, status }, not reactive
    this.raf = 0
  },

  watch: {
    lanes () { if (this.enabled) this.fetchLanes() },
    url () { this.extra = []; this.menuOpen = false },
    enabled (on) {
      this.savePrefs()
      if (on) {
        this.fetchLanes()
        this.$nextTick(() => this.startLoop())
      } else {
        this.stopLoop()
        this.menuOpen = false
      }
    }
  },

  mounted () {
    if (this.enabled) {
      this.fetchLanes()
      this.startLoop()
    }
  },

  beforeUnmount () {
    this.stopLoop()
    window.removeEventListener('pointermove', this.dragMove, true)
    window.removeEventListener('pointerup', this.dragEnd, true)
    window.removeEventListener('pointermove', this.resizeMove, true)
    window.removeEventListener('pointerup', this.resizeEnd, true)
  },

  methods: {
    toggle () {
      this.enabled = !this.enabled
    },
    laneColour (i) {
      return LANE_COLOURS[i % LANE_COLOURS.length]
    },
    // open the checklist on the side of the panel that has room inside the player
    toggleMenu () {
      this.menuOpen = !this.menuOpen
      if (!this.menuOpen) return
      this.menuStyle = { visibility: 'hidden' }
      this.$nextTick(() => {
        const box = this.$el.getBoundingClientRect()
        const p = this.$refs.panel.getBoundingClientRect()
        const m = this.$refs.menu
        if (!m) return
        const mw = m.offsetWidth
        const mh = m.offsetHeight
        const st = {}
        if (p.right - 6 - mw >= box.left + 4) st.right = '6px'
        else st.left = Math.max(6, Math.min(box.left + 4 - p.left, box.right - 4 - mw - p.left)) + 'px'
        if (p.top + 28 + mh <= box.bottom - 4 || p.top - box.top < mh) st.top = '28px'
        else st.bottom = 'calc(100% - 24px)'
        this.menuStyle = st
      })
    },
    toggleShown (id) {
      if (this.shown.includes(id)) {
        if (id === this.mainId) return // the main script always has a lane
        this.extra = this.extra.filter(x => x !== id)
      } else if (this.shown.length < MAX_LANES) {
        this.extra = [...this.extra, id]
      }
    },

    savePrefs () {
      try {
        localStorage.setItem(STORE_KEY, JSON.stringify({ enabled: this.enabled, x: this.pos.x, y: this.pos.y, w: this.size.w, h: this.size.h }))
      } catch {
        // storage blocked: the overlay still works, it just won't remember
      }
    },

    async fetchLanes () {
      const missing = this.lanes.filter(l => !this.cache.has(l.url))
      this.updateStatus()
      await Promise.all(missing.map(async l => {
        this.cache.set(l.url, { actions: [], status: 'loading' })
        this.updateStatus()
        try {
          const data = await (await fetch(l.url)).json()
          const inv = data.inverted === true
          const actions = (data.actions || [])
            .map(a => ({ at: a.at / 1000, pos: inv ? 100 - a.pos : a.pos, c: '' }))
            .sort((a, b) => a.at - b.at)
          // colour of the stroke from each action to the next, worked out once
          for (let i = 0; i < actions.length - 1; i++) {
            const p = actions[i]
            const n = actions[i + 1]
            p.c = speedColour(Math.abs(n.pos - p.pos) / Math.max(n.at - p.at, 0.001))
          }
          this.cache.set(l.url, { actions, status: actions.length ? '' : 'empty' })
        } catch {
          this.cache.set(l.url, { actions: [], status: 'unreadable' })
        }
        this.updateStatus()
      }))
    },
    updateStatus () {
      const st = this.lanes.map(l => (this.cache.get(l.url) || {}).status).filter(Boolean)
      this.status = st.includes('loading') ? 'loading' : (this.lanes.length === 1 ? (st[0] || '') : '')
    },

    indexAt (a, t) {
      let lo = 0
      let hi = a.length - 1
      if (!a.length || t < a[0].at) return -1
      while (lo < hi) {
        const mid = (lo + hi + 1) >> 1
        if (a[mid].at <= t) lo = mid
        else hi = mid - 1
      }
      return lo
    },
    positionAt (a, t) {
      const i = this.indexAt(a, t)
      if (i < 0) return a.length ? a[0].pos : 0
      if (i >= a.length - 1) return a[a.length - 1].pos
      const p = a[i]
      const n = a[i + 1]
      return p.pos + (n.pos - p.pos) * ((t - p.at) / (n.at - p.at || 1))
    },

    startLoop () {
      if (!this.raf) this.raf = requestAnimationFrame(this.draw)
    },
    stopLoop () {
      if (this.raf) cancelAnimationFrame(this.raf)
      this.raf = 0
    },

    draw () {
      this.raf = 0
      if (!this.enabled) return
      const cv = this.$refs.graph
      const v = this.video()
      if (cv && v) {
        const dpr = Math.min(window.devicePixelRatio || 1, 2)
        const w = cv.clientWidth
        const h = cv.clientHeight
        if (cv.width !== Math.round(w * dpr) || cv.height !== Math.round(h * dpr)) {
          cv.width = Math.round(w * dpr)
          cv.height = Math.round(h * dpr)
        }
        const ctx = cv.getContext('2d')
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
        ctx.clearRect(0, 0, w, h)

        const t = v.currentTime || 0
        const multi = this.lanes.length > 1
        const compact = w < COMPACT_W - 16
        if (compact !== this.compact) this.compact = compact
        if (compact) {
          this.drawBars(ctx, w, h, t)
          this.raf = requestAnimationFrame(this.draw)
          return
        }
        const meterW = 6
        const gw = w - meterW - 8 // graph width, meter on the right
        const span = Math.max(gw, 1) / PX_PER_SEC
        const start = t - span * PAST_SHARE
        const laneH = h / this.lanes.length
        const x = s => ((s - start) / span) * gw
        const px = x(t)

        this.lanes.forEach((lane, li) => {
          const top = li * laneH
          const pad = multi ? 5 : 4
          const y = p => top + pad + (1 - p / 100) * (laneH - pad * 2)
          const a = (this.cache.get(lane.url) || {}).actions || []

          ctx.strokeStyle = 'rgba(255,255,255,0.08)'
          ctx.lineWidth = 1
          ctx.beginPath()
          ctx.moveTo(0, top + laneH / 2)
          ctx.lineTo(gw, top + laneH / 2)
          ctx.stroke()
          if (li > 0) {
            ctx.strokeStyle = 'rgba(255,255,255,0.14)'
            ctx.beginPath()
            ctx.moveTo(0, top)
            ctx.lineTo(w, top)
            ctx.stroke()
          }

          if (a.length) {
            // Strokes are clipped to the graph so they slide in from the edge instead of popping in.
            ctx.lineWidth = multi ? 1.5 : 2
            ctx.lineJoin = 'round'
            ctx.lineCap = 'round'
            const i0 = Math.max(0, this.indexAt(a, start))
            const end = start + span + 1 // one second past the edge, covers the line caps
            const strokes = (from, to) => {
              for (let i = i0; i < a.length - 1 && a[i].at <= end; i++) {
                const p = a[i]
                const n = a[i + 1]
                if (n.at < from || p.at > to) continue
                ctx.strokeStyle = p.c
                ctx.beginPath()
                ctx.moveTo(x(p.at), y(p.pos))
                ctx.lineTo(x(n.at), y(n.pos))
                ctx.stroke()
              }
            }
            ctx.save()
            ctx.beginPath()
            ctx.rect(0, top, px, laneH)
            ctx.clip()
            ctx.globalAlpha = 0.45 // played part dimmed
            strokes(start - 1, t)
            ctx.restore()
            ctx.save()
            ctx.beginPath()
            ctx.rect(px, top, gw - px, laneH)
            ctx.clip()
            strokes(t, end)
            ctx.restore()
            const cur = this.positionAt(a, t)
            ctx.fillStyle = '#fff'
            ctx.beginPath()
            ctx.arc(px, y(cur), multi ? 3 : 3.5, 0, Math.PI * 2)
            ctx.fill()
            const mx = w - meterW
            const mTop = top + pad
            const mH = laneH - pad * 2
            ctx.fillStyle = 'rgba(255,255,255,0.1)'
            ctx.fillRect(mx, mTop, meterW, mH)
            ctx.fillStyle = multi ? this.laneColour(li) : '#7957d5'
            ctx.fillRect(mx, mTop + mH * (1 - cur / 100), meterW, mH * (cur / 100))
          }

          if (multi) {
            ctx.font = '600 10px system-ui, sans-serif'
            const label = (lane.ai && !/^AI/.test(lane.label) ? 'AI · ' : '') + lane.label
            const tw = Math.min(ctx.measureText(label).width, gw - 20)
            ctx.fillStyle = 'rgba(14,14,20,0.75)'
            ctx.fillRect(2, top + 2, tw + 16, 14)
            ctx.fillStyle = this.laneColour(li)
            ctx.beginPath()
            ctx.arc(8, top + 9, 3, 0, Math.PI * 2)
            ctx.fill()
            ctx.fillStyle = '#e0e0e4'
            ctx.fillText(label, 14, top + 13, gw - 20)
          }
        })

        ctx.strokeStyle = 'rgba(255,255,255,0.55)'
        ctx.lineWidth = 1
        ctx.beginPath()
        ctx.moveTo(px, 0)
        ctx.lineTo(px, h)
        ctx.stroke()
      }
      this.raf = requestAnimationFrame(this.draw)
    },

    // compact mode: one vertical up/down bar per lane, side by side
    drawBars (ctx, w, h, t) {
      const n = this.lanes.length
      const barW = Math.min(BAR_W, (w - (n - 1) * BAR_GAP) / n)
      const total = n * barW + (n - 1) * BAR_GAP
      let x0 = (w - total) / 2
      this.lanes.forEach((lane, li) => {
        const a = (this.cache.get(lane.url) || {}).actions || []
        ctx.fillStyle = 'rgba(255,255,255,0.1)'
        ctx.fillRect(x0, 0, barW, h)
        if (a.length) {
          const cur = this.positionAt(a, t)
          ctx.fillStyle = n > 1 ? this.laneColour(li) : '#7957d5'
          ctx.fillRect(x0, h * (1 - cur / 100), barW, h * (cur / 100))
        }
        x0 += barW + BAR_GAP
      })
    },

    // ── dragging (positions kept as % of the player so it survives resizes) ──
    dragStart (e) {
      if (e.button !== 0) return
      const panel = this.$refs.panel
      const box = this.$el.getBoundingClientRect()
      const r = panel.getBoundingClientRect()
      this.dragging = { dx: e.clientX - r.left, dy: e.clientY - r.top, box, w: r.width, h: r.height }
      // capture phase: the overlay stops bubbling so play does not toggle, which must not swallow the drag end
      window.addEventListener('pointermove', this.dragMove, true)
      window.addEventListener('pointerup', this.dragEnd, true)
      e.preventDefault()
    },
    dragMove (e) {
      const d = this.dragging
      if (!d) return
      const left = Math.max(0, Math.min(d.box.width - d.w, e.clientX - d.box.left - d.dx))
      const top = Math.max(0, Math.min(d.box.height - d.h, e.clientY - d.box.top - d.dy))
      this.pos = { x: (left / d.box.width) * 100, y: (top / d.box.height) * 100 }
    },
    dragEnd () {
      this.dragging = null
      window.removeEventListener('pointermove', this.dragMove, true)
      window.removeEventListener('pointerup', this.dragEnd, true)
      this.savePrefs()
    },

    // ── resizing from the bottom-right grip, kept inside the player ──
    resizeStart (e) {
      if (e.button !== 0) return
      const box = this.$el.getBoundingClientRect()
      const r = this.$refs.panel.getBoundingClientRect()
      this.resizing = { box, left: r.left, top: r.top, dx: r.right - e.clientX, dy: r.bottom - e.clientY }
      window.addEventListener('pointermove', this.resizeMove, true)
      window.addEventListener('pointerup', this.resizeEnd, true)
      e.preventDefault()
    },
    resizeMove (e) {
      const d = this.resizing
      if (!d) return
      // smallest: room for the header and ~24 px per lane
      const narrow = e.clientX + d.dx - d.left < COMPACT_W
      const minH = narrow ? 74 : 34 + this.lanes.length * 24
      // narrowest: the bars (14 px each, 6 px apart) or the header's grip + menu button
      const n = this.lanes.length
      const minW = Math.max(56, 16 + n * BAR_W + (n - 1) * BAR_GAP)
      const w = Math.max(minW, Math.min(d.box.right - d.left, e.clientX + d.dx - d.left))
      const h = Math.max(minH, Math.min(d.box.bottom - d.top, e.clientY + d.dy - d.top))
      this.size = { w: (w / d.box.width) * 100, h: (h / d.box.height) * 100 }
    },
    resizeEnd () {
      this.resizing = null
      window.removeEventListener('pointermove', this.resizeMove, true)
      window.removeEventListener('pointerup', this.resizeEnd, true)
      this.savePrefs()
    },
    resetSize () {
      this.size = { w: null, h: null }
      this.savePrefs()
    }
  }
})
</script>

<style scoped>
/* covers the player but only its children take pointer events */
.script-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 3;
}
.script-overlay > * {
  pointer-events: auto;
}

.script-toggle {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(6px);
  color: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  opacity: 0.7;
  transition: opacity 0.15s, background 0.15s;
}
.script-toggle:hover {
  opacity: 1;
}
.script-toggle.is-on {
  background: rgba(62, 142, 208, 0.9);
  color: #fff;
  opacity: 1;
}

.script-panel {
  position: absolute;
  width: min(300px, 70%);
  border-radius: 8px;
  background: rgba(14, 14, 20, 0.72);
  backdrop-filter: blur(8px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.35);
  color: #e0e0e4;
  user-select: none;
  touch-action: none;
}
.script-panel.is-multi {
  width: min(340px, 80%);
}
.script-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px 2px 4px;
  font-size: 11px;
  cursor: grab;
}
.script-head:active {
  cursor: grabbing;
}
.grip {
  opacity: 0.5;
}
.script-title {
  font-weight: 600;
  letter-spacing: 0.02em;
}
.script-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  opacity: 0.7;
}
.script-state {
  margin-left: auto;
  opacity: 0.6;
}
.script-ai {
  color: #5aa5e6;
}
.script-menu-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 20px;
  margin-left: 4px;
  border: none;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  color: inherit;
  cursor: pointer;
}
.script-state + .script-menu-btn {
  margin-left: 4px;
}
.script-menu-btn:hover,
.script-menu-btn.is-open {
  background: rgba(255, 255, 255, 0.18);
}

/* floating checklist under the header: opens over the graph (and past the panel
   edge when needed), so resizing the panel never squeezes it */
.script-menu {
  position: absolute;
  top: 28px;
  right: 6px;
  z-index: 10;
  width: 230px;
  max-width: calc(100vw - 32px);
  max-height: 260px;
  overflow-y: auto;
  padding: 4px;
  border-radius: 8px;
  background: rgba(22, 22, 30, 0.97);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45), 0 0 0 1px rgba(255, 255, 255, 0.08);
}
.script-menu-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.script-check {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 4px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
}
.script-check:hover {
  background: rgba(255, 255, 255, 0.06);
}
.script-check input {
  margin: 0;
  accent-color: #7957d5;
}
.script-check :deep(.icon) {
  flex: 0 0 auto;
  width: 14px;
  height: 14px;
  margin-left: -2px;
}
.script-check-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lane-dot {
  flex: 0 0 auto;
  width: 7px;
  height: 7px;
  border-radius: 50%;
}
.script-star :deep(.icon) {
  width: 14px;
  height: 14px;
}
.script-star {
  display: inline-flex;
  padding: 2px;
  border: none;
  background: none;
  color: rgba(255, 255, 255, 0.45);
  cursor: pointer;
}
.script-star:hover {
  color: #ffdd57;
}
.script-star.is-main {
  color: #ffdd57;
  cursor: default;
}

.script-body {
  padding: 0 8px 8px;
}
.script-graph {
  width: 100%;
  min-width: 0;
  display: block;
}
/* compact: just the grip and the menu button in the header */
.script-panel.is-compact .script-head .script-title,
.script-panel.is-compact .script-head .script-name,
.script-panel.is-compact .script-head .script-state,
.script-panel.is-compact .script-head .script-ai {
  display: none;
}
.script-panel.is-compact .script-menu-btn {
  margin-left: auto;
}
/* resized: header on top, the graph fills the rest */
.script-panel.is-sized {
  display: flex;
  flex-direction: column;
}
.script-panel.is-sized .script-body {
  position: relative;
  flex: 1;
  min-height: 0;
}
/* out of the layout: the canvas buffer size must not become the panel's minimum
   (that made the panel grow-only) */
.script-panel.is-sized .script-graph {
  position: absolute;
  left: 8px;
  top: 0;
  /* explicit size: a canvas does not stretch between left/right like a div */
  width: calc(100% - 16px);
  height: calc(100% - 8px);
}
.script-resize {
  position: absolute;
  right: 3px;
  bottom: 3px;
  width: 14px;
  height: 14px;
  line-height: 0;
  color: rgba(255, 255, 255, 0.35);
  cursor: nwse-resize;
  touch-action: none;
}
.script-resize :deep(.icon) {
  display: block;
  width: 14px;
  height: 14px;
}
.script-resize :deep(.mdi::before) {
  display: block;
  line-height: 14px;
}
.script-resize:hover {
  color: rgba(255, 255, 255, 0.8);
}
</style>
