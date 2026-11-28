<template>
  <div class="glass-morph" :class="{ on: active && drawn }" aria-hidden="true">
    <div ref="glass" class="glass"><canvas ref="lens" class="lens"/></div>
    <svg class="rim">
      <defs>
        <linearGradient :id="fadeId" x1="0" x2="1">
          <stop offset="0" stop-color="#fff" stop-opacity="0"/>
          <stop offset="0.08" stop-color="#fff" :stop-opacity="p.rim"/>
          <stop offset="0.92" stop-color="#fff" :stop-opacity="p.rim"/>
          <stop offset="1" stop-color="#fff" stop-opacity="0"/>
        </linearGradient>
      </defs>
      <path ref="rim" :stroke="`url(#${fadeId})`"/>
    </svg>
  </div>
</template>

<script>
// liquid glass slab that flows up into the button bar on hover; drawn with WebGL so it refracts in every browser
import { glassDefaults } from '@/glassDefaults'

const INFO_H = 52
const ease = t => 1 - Math.pow(1 - t, 4)

const FRAG = `#version 300 es
precision highp float; in vec2 q; out vec4 o;
uniform sampler2D tex; uniform vec2 src; uniform float w, cardH, boxH, y0, top, xL;
uniform float strength, edge, sat, bright, tint, sheen;
// depth inside the glass from one smooth slab+bar outline; the card frame is not a glass edge
float depth(vec2 p){
  float slab = p.y - y0;
  float bar = min(p.y - top, p.x - xL);
  float h = clamp(0.5 + 0.5 * (slab - bar) / 8.0, 0.0, 1.0);
  return mix(bar, slab, h) + 8.0 * h * (1.0 - h);
}
void main(){
  vec2 p = vec2(q.x * w, (1.0 - q.y) * boxH);
  float d = depth(p);
  vec2 n = normalize(vec2(depth(p + vec2(1.0, 0.0)) - depth(p - vec2(1.0, 0.0)), depth(p + vec2(0.0, 1.0)) - depth(p - vec2(0.0, 1.0))) + 1e-5);
  float k = 1.0 - smoothstep(0.0, edge, d);
  vec2 s = p + n * strength * k * k;
  vec2 card = vec2(s.x, cardH - boxH + s.y);
  float sc = max(w / src.x, cardH / src.y);
  vec3 c = texture(tex, (card - vec2(w, cardH) * 0.5) / (src * sc) + 0.5).rgb;
  float l = dot(c, vec3(0.2126, 0.7152, 0.0722));
  c = mix(vec3(l), c, sat) * bright;
  c = mix(c, vec3(0.0), tint);
  c = mix(c, vec3(1.0), k * k * 0.18);  // the bent rim catches light instead of darkening the edge
  c += sheen * (1.0 - smoothstep(0.0, 0.55, (q.x + 1.0 - q.y) * 0.5));
  o = vec4(c, 1.0);
}`

// one shared WebGL context: browsers drop contexts past ~16 and a lost one draws white; cards copy its result
let shared = null
function lens () {
  if (shared) return shared
  const canvas = document.createElement('canvas')
  const gl = canvas.getContext('webgl2', { premultipliedAlpha: false })
  if (!gl) return null
  canvas.addEventListener('webglcontextlost', () => { shared = null })
  const sh = (type, src) => { const o = gl.createShader(type); gl.shaderSource(o, src); gl.compileShader(o); return o }
  const prog = gl.createProgram()
  gl.attachShader(prog, sh(gl.VERTEX_SHADER, '#version 300 es\nin vec2 pos; out vec2 q; void main(){ q = pos * 0.5 + 0.5; gl_Position = vec4(pos, 0.0, 1.0); }'))
  gl.attachShader(prog, sh(gl.FRAGMENT_SHADER, FRAG))
  gl.linkProgram(prog); gl.useProgram(prog)
  gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer())
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]), gl.STATIC_DRAW)
  const loc = gl.getAttribLocation(prog, 'pos'); gl.enableVertexAttribArray(loc); gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0)
  gl.bindTexture(gl.TEXTURE_2D, gl.createTexture())
  for (const [k, v] of [[gl.TEXTURE_MIN_FILTER, gl.LINEAR], [gl.TEXTURE_MAG_FILTER, gl.LINEAR], [gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE], [gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE]]) gl.texParameteri(gl.TEXTURE_2D, k, v)
  const loc2 = {}
  shared = { canvas, gl, U: n => loc2[n] || (loc2[n] = gl.getUniformLocation(prog, n)) }
  return shared
}

export default {
  name: 'GlassMorph',
  props: {
    active: { type: Boolean, default: false },
    source: { type: Function, required: true },   // returns the <video> or <img> behind the glass
    bar: { type: Function, required: true }       // returns the button bar element
  },
  emits: ['ready'],
  data () {
    return { t: 0, drawn: false, fadeId: `glass-rim-${Math.random().toString(36).slice(2, 8)}` }
  },
  computed: {
    p () {
      const web = this.$store.state.optionsWeb.web
      const p = { ...glassDefaults, ...(web.glassCard || {}) }
      const theme = web.theme === 'auto' || !web.theme ? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light') : web.theme
      p.tint = theme === 'light' ? p.tintLight : p.tintDark
      return p
    }
  },
  watch: {
    active: {
      handler (on) {
        if (on) { this.animate(1); return }
        // leave: fade out together with the buttons, then reset the shape while invisible
        cancelAnimationFrame(this.anim)
        this.$emit('ready', false)
        clearTimeout(this.resetTimer)
        this.resetTimer = setTimeout(() => { if (!this.active) { this.t = 0; this.drawn = false; this.render(); cancelAnimationFrame(this.loop) } }, 150)
      },
      immediate: true
    },
    p () { this.render() }
  },
  beforeUnmount () {
    clearTimeout(this.resetTimer); cancelAnimationFrame(this.anim); cancelAnimationFrame(this.loop)
  },
  methods: {
    animate (to) {
      cancelAnimationFrame(this.anim)
      const from = this.t, start = performance.now(), dur = this.p.duration * Math.abs(to - from)
      if (to) this.startLoop()
      const step = now => {
        const k = Math.min(1, (now - start) / Math.max(1, dur))
        this.t = from + (to - from) * k
        this.render()
        this.$emit('ready', to === 1 && this.t > 0.75 && this.drawn)
        if (k < 1) this.anim = requestAnimationFrame(step)
        else if (!to) cancelAnimationFrame(this.loop)
      }
      this.anim = requestAnimationFrame(step)
    },
    // outline of the slab plus the bar rising from its right end; at t=0 the bar lies flat inside the slab
    geometry () {
      const card = this.$el.parentElement, bar = this.bar()
      const w = card.clientWidth, cardH = card.clientHeight, barW = bar ? bar.offsetWidth : 0, barH = bar ? bar.offsetHeight : 0
      const boxH = INFO_H + barH, y0 = boxH - INFO_H, e = ease(this.t), R = this.p.slabRadius
      return { w, cardH, boxH, y0, R, top: y0 - barH * e, xL: w - R - (barW - R) * e }
    },
    render () {
      if (!this.$el || !this.$el.parentElement) return
      const g = this.geometry(), P = this.p
      const f = Math.min(P.fillet, g.y0 - g.top, Math.max(0, (g.xL - g.R) / 2)), r = Math.min(P.pillRadius, (g.y0 - g.top) / 2)
      const path = (o, y0, top) => `M${o} ${y0 + g.R} Q${o} ${y0} ${g.R} ${y0} L${g.xL - f} ${y0} Q${g.xL} ${y0} ${g.xL} ${y0 - f} L${g.xL} ${top + r} Q${g.xL} ${top} ${g.xL + r} ${top} L${g.w - g.R} ${top} Q${g.w - o} ${top} ${g.w - o} ${top + g.R}`
      this.$refs.glass.style.height = g.boxH + 'px'
      this.$refs.glass.style.clipPath = `path("M0 ${g.boxH} L${path(0, g.y0, g.top).slice(1)} L${g.w} ${g.boxH} Z")`
      const dy = g.cardH - g.boxH + 0.5
      this.$refs.rim.setAttribute('d', path(1, g.y0 + dy, g.top + dy))
      this.geo = g
    },
    startLoop () {
      cancelAnimationFrame(this.loop)
      const draw = () => { this.drawLens(); this.loop = requestAnimationFrame(draw) }
      this.loop = requestAnimationFrame(draw)
    },
    // the glass waits for the video's first real frame; a blank frame from a just-started decoder is skipped
    pickFrame () {
      const src = this.source()
      const sw = src && (src.videoWidth || src.naturalWidth), sh = src && (src.videoHeight || src.naturalHeight)
      if (!sw || src.readyState < 2 || src.complete === false) return null
      const fw = Math.min(sw, 640), fh = Math.round(sh * fw / sw)
      const frame = this.frame || (this.frame = document.createElement('canvas'))
      if (frame.width !== fw || frame.height !== fh) { frame.width = fw; frame.height = fh }
      try { frame.getContext('2d').drawImage(src, 0, 0, fw, fh) } catch { return null }
      if (src !== this.goodSource) {
        if (this.isBlank(frame)) return null
        this.goodSource = src
      }
      return { frame, sw, sh }
    },
    isBlank (frame) {
      const probe = this.probe || (this.probe = document.createElement('canvas'))
      probe.width = probe.height = 8
      const ctx = probe.getContext('2d', { willReadFrequently: true })
      ctx.drawImage(frame, 0, 0, 8, 8)
      const d = ctx.getImageData(0, 0, 8, 8).data
      let lo = 255, hi = 0
      for (let i = 0; i < d.length; i += 4) { const l = d[i] + d[i + 1] + d[i + 2]; lo = Math.min(lo, l); hi = Math.max(hi, l) }
      return d[3] === 0 || hi - lo < 6
    },
    drawLens () {
      const g = this.geo
      const L = g && lens()
      if (!L) return
      const got = this.pickFrame()
      if (!got) return
      const { frame, sw, sh: shh } = got
      const { gl, U } = L, c = this.$refs.lens, dpr = devicePixelRatio, P = this.p
      const cw = Math.round(g.w * dpr), ch = Math.round(g.boxH * dpr)
      if (L.canvas.width !== cw || L.canvas.height !== ch) { L.canvas.width = cw; L.canvas.height = ch }
      gl.viewport(0, 0, cw, ch)
      try { gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, frame) } catch { return }
      for (const k of ['w', 'cardH', 'boxH', 'y0', 'top', 'xL']) gl.uniform1f(U(k), g[k])
      gl.uniform2f(U('src'), sw, shh)
      gl.uniform1f(U('strength'), P.refraction); gl.uniform1f(U('edge'), 18)
      gl.uniform1f(U('sat'), P.saturation / 100); gl.uniform1f(U('bright'), P.brightness)
      gl.uniform1f(U('tint'), P.tint); gl.uniform1f(U('sheen'), P.sheen)
      gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4)
      if (c.width !== cw || c.height !== ch) { c.width = cw; c.height = ch }
      c.getContext('2d').drawImage(L.canvas, 0, 0)
      if (!this.drawn) { this.drawn = true; if (this.active && this.t > 0.75) this.$emit('ready', true) }
    }
  }
}
</script>

<style scoped>
/* on exit the glass fades with the buttons and preview instead of retracting on its own */
.glass-morph {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  overflow: hidden;
  z-index: 21;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.12s ease-out;
}
.glass-morph.on {
  opacity: 1;
  transition: opacity 0.15s ease-in;
}
.glass {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
}
.lens {
  display: block;
  width: 100%;
  height: 100%;
}
.rim {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
}
.rim path {
  fill: none;
  stroke-width: 1;
}
</style>
