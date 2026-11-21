<template>
  <div v-if="url" class="script-overlay" @pointerdown.stop @pointerup.stop @click.stop @dblclick.stop>
    <button type="button" class="script-toggle" :class="{ 'is-on': enabled }"
            :title="enabled ? 'Hide script overlay' : 'Show script overlay'" @click="toggle">
      <b-icon pack="mdi" icon="pulse" size="is-small"/>
    </button>

    <div v-show="enabled" ref="panel" class="script-panel" :style="{ left: pos.x + '%', top: pos.y + '%' }">
      <div class="script-head" @pointerdown="dragStart">
        <b-icon pack="mdi" icon="drag" size="is-small" class="grip"/>
        <span class="script-title">Script</span>
        <b-icon v-if="currentIsAi" pack="mdi" icon="robot-outline" size="is-small" class="script-ai" title="AI script"/>
        <select v-if="scripts.length > 1" class="script-pick" :value="selectedId" :title="selectedName"
                @pointerdown.stop @change="$emit('select', Number($event.target.value))">
          <option v-for="o in scriptOptions" :key="o.id" :value="o.id" :title="o.name">{{ o.label }}</option>
        </select>
        <span class="script-state">{{ status }}</span>
      </div>
      <div class="script-body">
        <canvas ref="graph" class="script-graph"></canvas>
        <div class="script-meter" title="Current position">
          <div class="script-meter-fill" ref="meter"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
// Funscript overlay for the player: a draggable panel showing the strokes around
// the playhead (coloured by speed, like the heatmaps) and the current position.
// Off by default; on/off and the panel position are remembered per browser.
import { defineComponent } from 'vue'

const STORE_KEY = 'xbvr.scriptOverlay'
const BEFORE = 2.5 // seconds of past strokes shown left of the playhead
const AFTER = 3.5 // seconds of upcoming strokes

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

export default defineComponent({
  name: 'ScriptOverlay',
  emits: ['select'],

  props: {
    url: { type: String, default: '' },
    scripts: { type: Array, default: () => [] }, // [{ id, name, selected }]
    video: { type: Function, required: true } // () => HTMLVideoElement
  },

  data () {
    const prefs = loadPrefs()
    return {
      enabled: prefs.enabled === true,
      pos: { x: prefs.x ?? 3, y: prefs.y ?? 4 },
      status: ''
    }
  },

  computed: {
    selectedId () {
      const s = this.scripts.find(x => x.selected)
      return s ? s.id : null
    },
    currentIsAi () {
      const s = this.scripts.find(x => x.selected)
      return !!(s && s.ai)
    },
    selectedName () {
      const s = this.scripts.find(x => x.selected)
      return s ? s.name : ''
    },
    // short labels: drop the extension and the part all names share
    // ("Clip.mkv.funscript" + "Clip - Script v2.funscript" -> "mkv", "Script v2")
    scriptOptions () {
      const base = this.scripts.map(x => x.name.replace(/\.funscript$/i, ''))
      let pre = base[0] || ''
      for (const b of base) while (pre && !b.startsWith(pre)) pre = pre.slice(0, -1)
      return this.scripts.map((x, i) => {
        const rest = base[i].slice(pre.length).replace(/^[\s._\-\u2013()[\]]+/, '').trim()
        return { id: x.id, name: x.name, label: (x.ai ? 'AI · ' : '') + (rest || 'Default') }
      })
    }
  },

  created () {
    this.actions = [] // [{ at: seconds, pos: 0..100 }], not reactive
    this.raf = 0
    this.loadedUrl = ''
  },

  watch: {
    url () { if (this.enabled) this.fetchScript() },
    enabled (on) {
      this.savePrefs()
      if (on) {
        this.fetchScript()
        this.$nextTick(() => this.startLoop())
      } else {
        this.stopLoop()
      }
    }
  },

  mounted () {
    if (this.enabled) {
      this.fetchScript()
      this.startLoop()
    }
  },

  beforeUnmount () {
    this.stopLoop()
    window.removeEventListener('pointermove', this.dragMove, true)
    window.removeEventListener('pointerup', this.dragEnd, true)
  },

  methods: {
    toggle () {
      this.enabled = !this.enabled
    },

    savePrefs () {
      try {
        localStorage.setItem(STORE_KEY, JSON.stringify({ enabled: this.enabled, x: this.pos.x, y: this.pos.y }))
      } catch {
        // storage blocked: the overlay still works, it just won't remember
      }
    },

    async fetchScript () {
      if (!this.url || this.loadedUrl === this.url) return
      const url = this.url
      this.loadedUrl = url
      this.actions = []
      this.status = 'loading'
      try {
        const data = await (await fetch(url)).json()
        if (url !== this.url) return
        const inv = data.inverted === true
        this.actions = (data.actions || [])
          .map(a => ({ at: a.at / 1000, pos: inv ? 100 - a.pos : a.pos }))
          .sort((a, b) => a.at - b.at)
        this.status = this.actions.length ? '' : 'empty'
      } catch {
        if (url === this.url) this.status = 'unreadable'
      }
    },

    // index of the last action at or before t
    indexAt (t) {
      const a = this.actions
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

    positionAt (t) {
      const a = this.actions
      const i = this.indexAt(t)
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
        const start = t - BEFORE
        const span = BEFORE + AFTER
        const pad = 4
        const x = s => ((s - start) / span) * w
        const y = p => pad + (1 - p / 100) * (h - pad * 2)

        // mid line
        ctx.strokeStyle = 'rgba(255,255,255,0.08)'
        ctx.lineWidth = 1
        ctx.beginPath()
        ctx.moveTo(0, h / 2)
        ctx.lineTo(w, h / 2)
        ctx.stroke()

        const a = this.actions
        if (a.length) {
          let i = Math.max(0, this.indexAt(start))
          ctx.lineWidth = 2
          ctx.lineJoin = 'round'
          ctx.lineCap = 'round'
          for (; i < a.length - 1 && a[i].at <= start + span; i++) {
            const p = a[i]
            const n = a[i + 1]
            const speed = Math.abs(n.pos - p.pos) / Math.max(n.at - p.at, 0.001)
            ctx.strokeStyle = speedColour(speed)
            ctx.globalAlpha = n.at < t ? 0.45 : 1 // played strokes dimmed
            ctx.beginPath()
            ctx.moveTo(x(p.at), y(p.pos))
            ctx.lineTo(x(n.at), y(n.pos))
            ctx.stroke()
          }
          ctx.globalAlpha = 1
        }

        // playhead + current point
        const px = x(t)
        ctx.strokeStyle = 'rgba(255,255,255,0.55)'
        ctx.lineWidth = 1
        ctx.beginPath()
        ctx.moveTo(px, 0)
        ctx.lineTo(px, h)
        ctx.stroke()
        const cur = this.positionAt(t)
        if (a.length) {
          ctx.fillStyle = '#fff'
          ctx.beginPath()
          ctx.arc(px, y(cur), 3.5, 0, Math.PI * 2)
          ctx.fill()
        }
        if (this.$refs.meter) this.$refs.meter.style.height = (a.length ? cur : 0) + '%'
      }
      this.raf = requestAnimationFrame(this.draw)
    },

    // ── dragging (positions kept as % of the player so it survives resizes) ──
    dragStart (e) {
      if (e.button !== 0) return
      const panel = this.$refs.panel
      const box = this.$el.getBoundingClientRect()
      const r = panel.getBoundingClientRect()
      this.dragging = { dx: e.clientX - r.left, dy: e.clientY - r.top, box, w: r.width, h: r.height }
      // capture phase: the overlay stops pointer events from bubbling (so the
      // player does not toggle play), which must not swallow the drag's end
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
.script-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px 2px 4px;
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
.script-state {
  margin-left: auto;
  opacity: 0.6;
}
.script-ai {
  color: #5aa5e6;
}
.script-pick {
  min-width: 0;
  max-width: 150px;
  margin-left: 6px;
  padding: 1px 4px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.06);
  color: inherit;
  font: inherit;
  font-size: 11px;
  text-overflow: ellipsis;
  cursor: pointer;
}
.script-pick option {
  color: #e0e0e4;
  background: #1c1c26;
}
.script-body {
  display: flex;
  gap: 6px;
  padding: 0 8px 8px;
}
.script-graph {
  flex: 1;
  min-width: 0;
  height: 64px;
  display: block;
}
.script-meter {
  position: relative;
  width: 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.1);
  overflow: hidden;
}
.script-meter-fill {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 0;
  border-radius: 4px;
  background: linear-gradient(to top, #3e8ed0, #7957d5);
}
</style>
