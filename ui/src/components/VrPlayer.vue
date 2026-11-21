<template>
  <video-player ref="player" class="xbvr-player">
    <video-skin ref="skin" class="xbvr-skin" :style="{ aspectRatio }">
      <video ref="video" playsinline preload="none" :loop="loop" disableremoteplayback x-webkit-airplay="deny" :style="{ opacity: vrActive || !started ? 0 : 1 }"
             @playing="started = true" @emptied="started = false"></video>
      <canvas v-show="vrActive && started" ref="canvas" class="vr-canvas"></canvas>
    </video-skin>
  </video-player>
</template>

<script>
// Video.js 10 player (custom elements + skin) with our own VR view.
//
// Video.js 10 has no plugin system, so the old videojs-vr plugin is replaced by a
// small three.js layer: the <video> inside the skin keeps playing (hidden) and is
// drawn as a texture on a 180° hemisphere or a 360° sphere, under the skin's
// controls. Drag to look around, scroll (or zoom()) to change the field of view.
//
// The public methods mirror the bits of the Video.js 8 player API the views used
// (play, pause, paused, currentTime, duration, poster, dispose) so callers stay simple.
import { defineComponent } from 'vue'
import '@videojs/html/video/player'
import '@videojs/html/video/skin'

const MIN_FOV = 30
const MAX_FOV = 130

// File projection (from the scan) -> how to draw it.
export function vrMode (projection) {
  switch (projection) {
    case 'flat':
    case 'NONE':
    case '':
    case undefined:
    case null:
      return null
    case '360_mono': return { sphere: 360, eye: 'full' }
    case '360_tb': return { sphere: 360, eye: 'top' }
    case '360_sbs': return { sphere: 360, eye: 'left' }
    case '180_mono': return { sphere: 180, eye: 'full' }
    // VR180 fisheye formats (side by side, left eye drawn), by lens field of view
    case 'fisheye': return { fisheye: 180, eye: 'left' }
    case 'fisheye190':
    case 'rf52': return { fisheye: 190, eye: 'left' }
    case 'mkx200': return { fisheye: 200, eye: 'left' }
    case 'mkx220':
    case 'vrca220': return { fisheye: 220, eye: 'left' }
    default: return { sphere: 180, eye: 'left' } // 180_sbs and unknown
  }
}

export default defineComponent({
  name: 'VrPlayer',

  props: {
    aspectRatio: { type: String, default: '1 / 1' },
    loop: { type: Boolean, default: true }
  },

  emits: ['ready'],

  data () {
    return {
      vrActive: false,
      // Black with just the play button until playback starts (no poster or
      // stretched first frame behind it).
      started: false
    }
  },

  created () {
    // three.js state lives outside Vue reactivity on purpose
    this.three = null
    this.mode = null
    this.lon = 0
    this.lat = 0
    this.fov = 90
    this.raf = 0
    this.drag = null
  },

  mounted () {
    const skin = this.$refs.skin
    this.onPointerDown = e => {
      if (!this.vrActive || e.button !== 0) return
      this.drag = { x: e.clientX, y: e.clientY, lon: this.lon, lat: this.lat, moved: false }
    }
    this.onPointerMove = e => {
      if (!this.drag) return
      const dx = e.clientX - this.drag.x
      const dy = e.clientY - this.drag.y
      if (Math.abs(dx) + Math.abs(dy) > 4) this.drag.moved = true
      const k = this.fov / (skin.clientHeight || 500)
      this.lon = this.drag.lon - dx * k
      this.lat = Math.max(-85, Math.min(85, this.drag.lat + dy * k))
      if (this.mode && this.mode.sphere === 180) this.lon = Math.max(-90, Math.min(90, this.lon))
    }
    this.onPointerUp = e => {
      // a drag must not also count as a click (which would toggle play/pause)
      if (this.drag && this.drag.moved) {
        e.stopPropagation()
        skin.addEventListener('click', ev => ev.stopPropagation(), { capture: true, once: true })
      }
      this.drag = null
    }
    this.onWheel = e => {
      if (!this.vrActive || this.overControls(e)) return // let menus and sliders scroll
      e.preventDefault()
      this.zoom(e.deltaY < 0)
    }
    skin.addEventListener('pointerdown', this.onPointerDown)
    window.addEventListener('pointermove', this.onPointerMove)
    window.addEventListener('pointerup', this.onPointerUp, true)
    skin.addEventListener('wheel', this.onWheel, { passive: false })
    this.styleSkin()
    this.resizeObserver = new ResizeObserver(() => this.resize())
    this.resizeObserver.observe(skin)
    this.$emit('ready', this)
  },

  beforeUnmount () {
    this.dispose()
  },

  methods: {
    // ── player API used by the views ──────────────────────────────────────
    el () { return this.$refs.skin },
    video () { return this.$refs.video },
    play () { const v = this.$refs.video; return v ? v.play().catch(() => {}) : Promise.resolve() },
    pause () { if (this.$refs.video) this.$refs.video.pause() },
    paused () { return this.$refs.video ? this.$refs.video.paused : true },
    duration () { return this.$refs.video ? this.$refs.video.duration || 0 : 0 },
    currentTime (t) {
      const v = this.$refs.video
      if (!v) return 0
      if (t !== undefined) v.currentTime = t
      return v.currentTime
    },
    poster (url) {
      if (this.$refs.player) this.$refs.player.setAttribute('poster', url || '')
    },

    // Load a source. projection is a file projection ('180_sbs', '360_tb', 'flat', ...).
    async load (src, projection) {
      const v = this.$refs.video
      if (!v) return
      this.mode = vrMode(projection)
      this.lon = 0
      this.lat = 0
      this.fov = this.mode && this.mode.sphere === 360 ? 90 : 100
      v.src = src
      v.load()
      if (this.mode) {
        await this.startVR()
      } else {
        this.stopVR()
      }
    },

    zoom (zoomIn) {
      if (!this.vrActive) return
      this.fov = Math.max(MIN_FOV, Math.min(MAX_FOV, this.fov + (zoomIn ? -2 : 2)))
    },

    dispose () {
      this.stopVR()
      const skin = this.$refs.skin
      if (skin) {
        skin.removeEventListener('pointerdown', this.onPointerDown)
        skin.removeEventListener('wheel', this.onWheel)
      }
      window.removeEventListener('pointermove', this.onPointerMove)
      window.removeEventListener('pointerup', this.onPointerUp, true)
      if (this.resizeObserver) this.resizeObserver.disconnect()
      if (this.three) {
        this.three.texture.dispose()
        this.three.renderer.dispose()
        this.three = null
      }
      const v = this.$refs.video
      if (v) {
        v.pause()
        v.removeAttribute('src')
        v.load()
      }
    },

    // True when the event comes from the control bar, a menu or a popover.
    overControls (e) {
      return e.composedPath().some(n => n.tagName && /^MEDIA-(CONTROLS|MENU|VOLUME-POPOVER|TIME-SLIDER|TOOLTIP|DIALOG)/.test(n.tagName))
    },

    // The packaged skin renders in a shadow root, so our page CSS cannot reach it.
    // Add a small sheet there: no cast/AirPlay buttons (never used), and no bright
    // focus outline around the whole player when it takes keyboard focus. Buttons
    // keep their own focus rings.
    styleSkin () {
      const skin = this.$refs.skin
      const apply = () => {
        const root = skin && skin.shadowRoot
        if (!root) return false
        const sheet = new CSSStyleSheet()
        sheet.replaceSync(`
          media-cast-button, media-airplay-button { display: none !important; }
          media-container:focus, media-container:focus-visible { outline: none !important; box-shadow: none !important; }
        `)
        root.adoptedStyleSheets = [...root.adoptedStyleSheets, sheet]
        return true
      }
      if (!apply()) customElements.whenDefined('video-skin').then(() => requestAnimationFrame(apply))
    },

    // ── VR view ───────────────────────────────────────────────────────────
    async startVR () {
      const THREE = await import('three')
      if (!this.$refs.canvas) return
      if (!this.three) {
        const renderer = new THREE.WebGLRenderer({ canvas: this.$refs.canvas, antialias: true })
        renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
        renderer.outputColorSpace = THREE.SRGBColorSpace
        const scene = new THREE.Scene()
        const camera = new THREE.PerspectiveCamera(this.fov, 1, 0.1, 1100)
        const texture = new THREE.VideoTexture(this.$refs.video)
        texture.colorSpace = THREE.SRGBColorSpace
        this.three = { THREE, renderer, scene, camera, texture, mesh: null }
      }
      this.buildMesh()
      this.vrActive = true
      this.$nextTick(() => {
        this.resize()
        if (!this.raf) this.raf = requestAnimationFrame(this.render)
      })
    },

    stopVR () {
      this.vrActive = false
      if (this.raf) cancelAnimationFrame(this.raf)
      this.raf = 0
    },

    buildMesh () {
      const { THREE, scene, texture } = this.three
      if (this.three.mesh) {
        scene.remove(this.three.mesh)
        this.three.mesh.geometry.dispose()
        this.three.mesh.material.dispose()
      }
      const mesh = this.mode.fisheye ? this.fisheyeMesh(THREE, texture) : this.equirectMesh(THREE, texture)
      scene.add(mesh)
      this.three.mesh = mesh
    },

    // Equirectangular 180/360: image centre (u = 0.5) ends up straight ahead (-Z).
    equirectMesh (THREE, texture) {
      const full = this.mode.sphere === 360
      const geometry = full
        ? new THREE.SphereGeometry(500, 64, 48)
        : new THREE.SphereGeometry(500, 64, 48, Math.PI / 2, Math.PI)
      geometry.scale(-1, 1, 1) // view from inside
      const uv = geometry.attributes.uv
      for (let i = 0; i < uv.count; i++) {
        if (this.mode.eye === 'left') uv.setX(i, uv.getX(i) * 0.5)
        else if (this.mode.eye === 'top') uv.setY(i, uv.getY(i) * 0.5 + 0.5)
      }
      uv.needsUpdate = true
      const mesh = new THREE.Mesh(geometry, new THREE.MeshBasicMaterial({ map: texture }))
      mesh.rotation.y = -Math.PI / 2
      return mesh
    },

    // Equidistant fisheye (VR180 lenses): a spherical cap of the lens field of
    // view around -Z. Each vertex samples the image at a radius proportional to
    // its angle from the lens axis, inside the left eye's circle.
    fisheyeMesh (THREE, texture) {
      const half = (this.mode.fisheye / 2) * Math.PI / 180
      const geometry = new THREE.SphereGeometry(500, 96, 64, 0, Math.PI * 2, 0, half)
      geometry.rotateX(-Math.PI / 2) // cap pole (+Y) -> straight ahead (-Z)
      const pos = geometry.attributes.position
      const uv = geometry.attributes.uv
      const cu = this.mode.eye === 'left' ? 0.25 : 0.5
      const ru = this.mode.eye === 'left' ? 0.25 : 0.5
      for (let i = 0; i < pos.count; i++) {
        const x = pos.getX(i)
        const y = pos.getY(i)
        const z = pos.getZ(i)
        const len = Math.hypot(x, y, z) || 1
        const theta = Math.acos(Math.max(-1, Math.min(1, -z / len)))
        const r = theta / half
        const phi = Math.atan2(y, x)
        uv.setXY(i, cu + r * Math.cos(phi) * ru, 0.5 + r * Math.sin(phi) * 0.5)
      }
      uv.needsUpdate = true
      return new THREE.Mesh(geometry, new THREE.MeshBasicMaterial({ map: texture, side: THREE.BackSide }))
    },

    resize () {
      if (!this.three || !this.$refs.skin) return
      const w = this.$refs.skin.clientWidth
      const h = this.$refs.skin.clientHeight
      if (!w || !h) return
      this.three.renderer.setSize(w, h, false)
      this.three.camera.aspect = w / h
      this.three.camera.updateProjectionMatrix()
    },

    render () {
      if (!this.vrActive || !this.three) { this.raf = 0; return }
      const { camera, renderer, scene } = this.three
      if (camera.fov !== this.fov) {
        camera.fov = this.fov
        camera.updateProjectionMatrix()
      }
      const lon = this.lon * Math.PI / 180
      const lat = this.lat * Math.PI / 180
      camera.lookAt(Math.sin(lon) * Math.cos(lat), Math.sin(lat), -Math.cos(lon) * Math.cos(lat))
      renderer.render(scene, camera)
      this.raf = requestAnimationFrame(this.render)
    }
  }
})
</script>

<style scoped>
.xbvr-skin {
  display: block;
  width: 100%;
  background: #000;
}
.xbvr-skin video {
  width: 100%;
  height: 100%;
}
.vr-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
  touch-action: none;
  cursor: grab;
}
.vr-canvas:active {
  cursor: grabbing;
}
</style>
