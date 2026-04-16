<template>
  <div class="container">
    <b-loading :is-full-page="false" v-model="isLoading" />
    <div class="content">
      <h3>{{ $t('Web UI') }}</h3>
      <hr />

      <div class="settings-layout">
        <!-- Left: Settings -->
        <div class="settings-col">

          <!-- Scene Cards -->
          <div class="settings-group">
            <h5 class="group-title">Scene Cards</h5>
            <div class="settings-row">
              <div class="setting-item">
                <label class="setting-label">Aspect Ratio</label>
                <b-select size="is-small" v-model="sceneCardAspectRatio">
                  <option>1:1</option>
                  <option>3:2</option>
                  <option>16:9</option>
                </b-select>
              </div>
              <div class="setting-item">
                <label class="setting-label">Date Format</label>
                <b-select size="is-small" v-model="dateFormat">
                  <option value="yyyy-MM-dd">2025-01-15</option>
                  <option value="MMM d, yyyy">Jan 15, 2025</option>
                  <option value="MMMM d, yyyy">January 15, 2025</option>
                  <option value="dd/MM/yyyy">15/01/2025</option>
                  <option value="MM/dd/yyyy">01/15/2025</option>
                </b-select>
              </div>
            </div>
            <div class="settings-row" style="margin-top: 0.5rem">
              <div class="setting-item">
                <b-switch v-model="sceneCardScaleToFit" type="is-dark" size="is-small">
                  Scale to fit
                </b-switch>
              </div>
              <div class="setting-item">
                <b-switch v-model="showSiteLogo" type="is-dark" size="is-small">
                  Show site logo
                </b-switch>
              </div>
            </div>
            <div class="setting-item slider-item">
              <label class="setting-label">Unavailable scene opacity</label>
              <div class="slider-wrap">
                <b-slider :min="0" :max="100" :step="10" :tooltip="false" v-model="isAvailOpacity" size="is-small"></b-slider>
                <span class="slider-value">{{ isAvailOpacity }}%</span>
              </div>
            </div>
          </div>

          <!-- Visible Buttons -->
          <div class="settings-group">
            <h5 class="group-title">Visible Buttons</h5>
            <div class="toggle-grid">
              <b-switch v-model="sceneHidden" type="is-danger" size="is-small">Hidden</b-switch>
              <b-switch v-model="sceneWatchlist" type="is-default" size="is-small">Watchlist</b-switch>
              <b-switch v-model="sceneFavourite" type="is-danger" size="is-small">Favourite</b-switch>
              <b-switch v-model="sceneWishlist" type="is-info" size="is-small">Wishlist</b-switch>
              <b-switch v-model="sceneTrailerlist" type="is-default" size="is-small">Trailer List</b-switch>
              <b-switch v-model="sceneWatched" type="is-dark" size="is-small">Watched</b-switch>
              <b-switch v-model="sceneEdit" type="is-dark" size="is-small">Edit</b-switch>
              <b-switch v-model="sceneDuration" type="is-dark" size="is-small">Duration</b-switch>
              <b-switch v-model="sceneCuepoint" type="is-dark" size="is-small">Cuepoints</b-switch>
              <b-switch v-model="hspFile" type="is-dark" size="is-small">HSP File</b-switch>
              <b-switch v-model="subtitlesFile" type="is-dark" size="is-small">Subtitles</b-switch>
              <b-switch v-model="ScriptHeatmap" type="is-dark" size="is-small">Heatmap</b-switch>
              <b-switch v-if="ScriptHeatmap" v-model="AllHeatmaps" type="is-dark" size="is-small">All Heatmaps</b-switch>
              <b-switch v-model="openInNewWindow" type="is-dark" size="is-small">Open in New Window</b-switch>
            </div>
          </div>

          <!-- Actor Cards -->
          <div class="settings-group">
            <h5 class="group-title">Actor Cards</h5>
            <div class="settings-row">
              <div class="setting-item">
                <label class="setting-label">Aspect Ratio</label>
                <b-select size="is-small" v-model="actorCardAspectRatio">
                  <option>1:1</option>
                  <option>2:3</option>
                  <option>9:16</option>
                </b-select>
              </div>
              <div class="setting-item">
                <b-switch v-model="actorCardScaleToFit" type="is-dark" size="is-small">
                  Scale to fit
                </b-switch>
              </div>
            </div>
          </div>

          <!-- General -->
          <div class="settings-group">
            <h5 class="group-title">General</h5>
            <div class="settings-row">
              <div class="setting-item">
                <label class="setting-label">Tag Sort</label>
                <b-select size="is-small" v-model="tagSort">
                  <option value="by-tag-count">By Count</option>
                  <option value="alphabetically">A-Z</option>
                </b-select>
              </div>
              <div class="setting-item">
                <label class="setting-label">Theme</label>
                <b-select size="is-small" v-model="theme">
                  <option value="auto">Auto</option>
                  <option value="dark">Dark</option>
                  <option value="light">Light</option>
                </b-select>
              </div>
              <div class="setting-item">
                <b-switch v-model="accentColorTags" type="is-dark" size="is-small">
                  Accent color tags
                </b-switch>
              </div>
            </div>
          </div>
        </div>

        <!-- Right: Live Preview -->
        <div class="preview-col">
          <div class="preview-sticky">
            <h5 class="group-title">Preview</h5>
            <div class="preview-card">
              <div class="preview-thumb" :style="{ aspectRatio: previewAspectRatio }">
                <img v-if="previewCoverUrl" :src="previewCoverUrl" :style="{ objectFit: sceneCardScaleToFit ? 'contain' : 'cover' }"/>
                <div v-else class="preview-placeholder-img"></div>

                <!-- Tags top-left -->
                <div class="preview-tags-left">
                  <span class="p-tag p-tag-info">
                    <b-icon pack="mdi" icon="file" size="is-small"/><span>2</span>
                  </span>
                  <span class="p-tag p-tag-funscript">
                    <b-icon pack="mdi" icon="pulse" size="is-small"/>
                  </span>
                  <span v-if="hspFile" class="p-tag p-tag-info">
                    <b-icon pack="mdi" icon="safety-goggles" size="is-small"/>
                  </span>
                  <span v-if="subtitlesFile" class="p-tag p-tag-info">
                    <b-icon pack="mdi" icon="subtitles" size="is-small"/>
                  </span>
                  <span v-if="sceneCuepoint" class="p-tag p-tag-info">
                    <b-icon pack="mdi" icon="skip-next-outline" size="is-small"/><span>3</span>
                  </span>
                </div>

                <!-- Tags top-right -->
                <div class="preview-tags-right">
                  <span class="p-tag p-tag-star">
                    <b-icon pack="mdi" icon="star" size="is-small"/>4
                  </span>
                  <span v-if="sceneDuration" class="p-tag p-tag-duration">
                    45m
                  </span>
                </div>

                <!-- Heatmap -->
                <div v-if="ScriptHeatmap" class="preview-heatmap">
                  <svg width="100%" height="8" preserveAspectRatio="none">
                    <defs>
                      <linearGradient id="heatGrad">
                        <stop offset="0%" stop-color="#3b82f6"/>
                        <stop offset="15%" stop-color="#22c55e"/>
                        <stop offset="30%" stop-color="#eab308"/>
                        <stop offset="45%" stop-color="#f97316"/>
                        <stop offset="55%" stop-color="#ef4444"/>
                        <stop offset="65%" stop-color="#f97316"/>
                        <stop offset="75%" stop-color="#22c55e"/>
                        <stop offset="85%" stop-color="#3b82f6"/>
                        <stop offset="100%" stop-color="#22c55e"/>
                      </linearGradient>
                    </defs>
                    <rect width="100%" height="8" rx="3" fill="url(#heatGrad)"/>
                  </svg>
                </div>

                <!-- Buttons — always visible -->
                <div class="preview-actions">
                  <span v-if="sceneHidden" class="p-btn p-btn-danger">
                    <b-icon pack="mdi" icon="eye-off" size="is-small"/>
                  </span>
                  <span v-if="sceneWatchlist" class="p-btn p-btn-primary">
                    <b-icon pack="mdi" icon="calendar-check" size="is-small"/>
                  </span>
                  <span v-if="sceneTrailerlist" class="p-btn p-btn-primary">
                    <b-icon pack="mdi" icon="movie-open-check" size="is-small"/>
                  </span>
                  <span v-if="sceneFavourite" class="p-btn p-btn-danger">
                    <b-icon pack="mdi" icon="heart" size="is-small"/>
                  </span>
                  <span v-if="sceneWishlist" class="p-btn p-btn-info">
                    <b-icon pack="mdi" icon="oil-lamp" size="is-small"/>
                  </span>
                  <span v-if="sceneWatched" class="p-btn p-btn-dark">
                    <b-icon pack="mdi" icon="eye-check" size="is-small"/>
                  </span>
                  <span v-if="sceneEdit" class="p-btn p-btn-dark">
                    <b-icon pack="mdi" icon="pencil" size="is-small"/>
                  </span>
                </div>
              </div>

              <!-- Info -->
              <div class="preview-info">
                <div class="preview-title-row">
                  <span class="preview-title">{{ previewTitle }}</span>
                  <span class="preview-status-icons">
                    <b-icon v-if="sceneFavourite" pack="mdi" icon="heart" size="is-small" class="pi-danger"/>
                    <b-icon v-if="sceneWatchlist" pack="mdi" icon="calendar-check" size="is-small" class="pi-primary"/>
                    <b-icon v-if="sceneWatched" pack="mdi" icon="eye-check" size="is-small" class="pi-dark"/>
                  </span>
                </div>
                <div class="preview-meta">
                  <span class="preview-site">
                    <img v-if="showSiteLogo && previewSiteIcon" :src="previewSiteIcon" class="preview-site-icon"/>
                    {{ previewSiteName }}
                  </span>
                  <span class="preview-date">{{ previewFormattedDate }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import ky from 'ky'
import { format, parseISO } from 'date-fns'

let saveTimer = null

export default defineComponent({
  name: 'InterfaceWeb',

  data () {
    return {
      sampleScene: null
    }
  },

  async mounted () {
    this.$store.dispatch('optionsWeb/load')
    this.loadSampleScene()
  },

  methods: {
    debouncedSave () {
      if (saveTimer) clearTimeout(saveTimer)
      saveTimer = setTimeout(() => {
        this.$store.dispatch('optionsWeb/save')
      }, 600)
    },
    async loadSampleScene () {
      try {
        const data = await ky.post('/api/scene/list', {
          json: { limit: 1, offset: 0 },
          timeout: 5000
        }).json()
        if (data.scenes && data.scenes.length > 0) {
          this.sampleScene = data.scenes[0]
        }
      } catch (e) {
        // no scenes available
      }
    }
  },

  watch: {
    '$store.state.optionsWeb.web': {
      handler () {
        if (!this.$store.state.optionsWeb.loading) {
          this.debouncedSave()
        }
      },
      deep: true
    }
  },

  computed: {
    previewTitle () {
      return this.sampleScene ? this.sampleScene.title : 'Amazing VR Experience Vol. 3'
    },
    previewSiteName () {
      return this.sampleScene ? this.sampleScene.site : 'StudioName'
    },
    previewFormattedDate () {
      const fmt = this.dateFormat
      if (this.sampleScene && this.sampleScene.release_date && this.sampleScene.release_date !== '0001-01-01T00:00:00Z') {
        return format(parseISO(this.sampleScene.release_date), fmt)
      }
      return format(new Date(2025, 0, 15), fmt)
    },
    previewAspectRatio () {
      const r = this.sceneCardAspectRatio || '16:9'
      const parts = r.split(':')
      if (parts.length === 2) return `${parts[0]} / ${parts[1]}`
      return '16 / 9'
    },
    previewCoverUrl () {
      if (!this.sampleScene || !this.sampleScene.cover_url) return ''
      const u = this.sampleScene.cover_url
      if (!u.startsWith('http')) return u
      if (u.search('%') === -1) return '/img/700x/' + encodeURI(u)
      return '/img/700x/' + encodeURI(decodeURI(u))
    },
    previewSiteIcon () {
      if (!this.sampleScene) return null
      const sites = this.$store.state.optionsSites.items
      if (!sites || !sites.length) return null
      const site = sites.find(s => s.id === this.sampleScene.scraper_id)
      if (!site || !site.avatar_url) return null
      const u = site.avatar_url
      if (!u.startsWith('http')) return u
      if (u.search('%') === -1) return '/img/200x/' + encodeURI(u)
      return '/img/200x/' + encodeURI(decodeURI(u))
    },
    tagSort: {
      get () { return this.$store.state.optionsWeb.web.tagSort },
      set (value) { this.$store.state.optionsWeb.web.tagSort = value }
    },
    sceneHidden: {
      get () { return this.$store.state.optionsWeb.web.sceneHidden },
      set (value) { this.$store.state.optionsWeb.web.sceneHidden = value }
    },
    sceneWatchlist: {
      get () { return this.$store.state.optionsWeb.web.sceneWatchlist },
      set (value) { this.$store.state.optionsWeb.web.sceneWatchlist = value }
    },
    sceneFavourite: {
      get () { return this.$store.state.optionsWeb.web.sceneFavourite },
      set (value) { this.$store.state.optionsWeb.web.sceneFavourite = value }
    },
    sceneWishlist: {
      get () { return this.$store.state.optionsWeb.web.sceneWishlist },
      set (value) { this.$store.state.optionsWeb.web.sceneWishlist = value }
    },
    sceneTrailerlist: {
      get () { return this.$store.state.optionsWeb.web.sceneTrailerlist },
      set (value) { this.$store.state.optionsWeb.web.sceneTrailerlist = value }
    },
    sceneWatched: {
      get () { return this.$store.state.optionsWeb.web.sceneWatched },
      set (value) { this.$store.state.optionsWeb.web.sceneWatched = value }
    },
    sceneEdit: {
      get () { return this.$store.state.optionsWeb.web.sceneEdit },
      set (value) { this.$store.state.optionsWeb.web.sceneEdit = value }
    },
    ScriptHeatmap: {
      get () { return this.$store.state.optionsWeb.web.showScriptHeatmap },
      set (value) { this.$store.state.optionsWeb.web.showScriptHeatmap = value }
    },
    AllHeatmaps: {
      get () { return this.$store.state.optionsWeb.web.showAllHeatmaps },
      set (value) { this.$store.state.optionsWeb.web.showAllHeatmaps = value }
    },
    sceneDuration: {
      get () { return this.$store.state.optionsWeb.web.sceneDuration },
      set (value) { this.$store.state.optionsWeb.web.sceneDuration = value }
    },
    sceneCuepoint: {
      get () { return this.$store.state.optionsWeb.web.sceneCuepoint },
      set (value) { this.$store.state.optionsWeb.web.sceneCuepoint = value }
    },
    hspFile: {
      get () { return this.$store.state.optionsWeb.web.showHspFile },
      set (value) { this.$store.state.optionsWeb.web.showHspFile = value }
    },
    subtitlesFile: {
      get () { return this.$store.state.optionsWeb.web.showSubtitlesFile },
      set (value) { this.$store.state.optionsWeb.web.showSubtitlesFile = value }
    },
    openInNewWindow: {
      get () { return this.$store.state.optionsWeb.web.showOpenInNewWindow },
      set (value) { this.$store.state.optionsWeb.web.showOpenInNewWindow = value }
    },
    isAvailOpacity: {
      get () {
        if (this.$store.state.optionsWeb.web.isAvailOpacity == undefined) {
          return 40
        }
        return this.$store.state.optionsWeb.web.isAvailOpacity
      },
      set (value) { this.$store.state.optionsWeb.web.isAvailOpacity = value }
    },
    sceneCardAspectRatio: {
      get () { return this.$store.state.optionsWeb.web.sceneCardAspectRatio },
      set (value) { this.$store.state.optionsWeb.web.sceneCardAspectRatio = value }
    },
    sceneCardScaleToFit: {
      get () { return this.$store.state.optionsWeb.web.sceneCardScaleToFit },
      set (value) { this.$store.state.optionsWeb.web.sceneCardScaleToFit = value }
    },
    actorCardAspectRatio: {
      get () { return this.$store.state.optionsWeb.web.actorCardAspectRatio },
      set (value) { this.$store.state.optionsWeb.web.actorCardAspectRatio = value }
    },
    actorCardScaleToFit: {
      get () { return this.$store.state.optionsWeb.web.actorCardScaleToFit },
      set (value) { this.$store.state.optionsWeb.web.actorCardScaleToFit = value }
    },
    showSiteLogo: {
      get () { return this.$store.state.optionsWeb.web.showSiteLogo !== false },
      set (value) { this.$store.state.optionsWeb.web.showSiteLogo = value }
    },
    dateFormat: {
      get () { return this.$store.state.optionsWeb.web.dateFormat || 'yyyy-MM-dd' },
      set (value) { this.$store.state.optionsWeb.web.dateFormat = value }
    },
    accentColorTags: {
      get () { return this.$store.state.optionsWeb.web.accentColorTags !== false },
      set (value) { this.$store.state.optionsWeb.web.accentColorTags = value }
    },
    updateCheck: {
      get () { return this.$store.state.optionsWeb.web.updateCheck },
      set (value) { this.$store.state.optionsWeb.web.updateCheck = value }
    },
    theme: {
      get () { return this.$store.state.optionsWeb.web.theme || 'auto' },
      set (value) { this.$store.state.optionsWeb.web.theme = value }
    },
    isLoading: function () {
      return this.$store.state.optionsWeb.loading
    }
  },
});
</script>

<style scoped>
/* ── Layout ── */
.settings-layout {
  display: flex;
  gap: 2rem;
  align-items: flex-start;
}
.settings-col {
  flex: 1;
  min-width: 0;
}
.preview-col {
  flex: 0 0 280px;
}
.preview-sticky {
  position: sticky;
  top: 1rem;
}

/* ── Settings ── */
.settings-group {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid rgba(128, 128, 128, 0.15);
}
.settings-group:last-child {
  border-bottom: none;
}
.group-title {
  margin-bottom: 0.75rem !important;
  font-weight: 600;
  font-size: 0.95rem;
}
.settings-row {
  display: flex;
  flex-wrap: wrap;
  gap: 1.25rem;
  align-items: center;
}
.setting-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.setting-label {
  font-size: 0.85rem;
  font-weight: 500;
  white-space: nowrap;
}
.slider-item {
  margin-top: 0.75rem;
  flex-direction: column;
  align-items: flex-start;
}
.slider-wrap {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  max-width: 320px;
}
.slider-wrap .b-slider {
  flex: 1;
}
.slider-value {
  font-size: 0.8rem;
  min-width: 32px;
  text-align: right;
  opacity: 0.7;
}
.toggle-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 0.5rem 1rem;
}

/* ── Preview Card ── */
.preview-card {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  cursor: default;
}
.preview-thumb {
  position: relative;
  background: #1a1a2e;
  overflow: hidden;
}
.preview-thumb > img {
  position: absolute;
  top: 0; left: 0;
  width: 100%; height: 100%;
  object-position: center center;
}
.preview-placeholder-img {
  position: absolute;
  top: 0; left: 0;
  width: 100%; height: 100%;
  background: linear-gradient(135deg, #2a2a4a 0%, #1a1a3e 50%, #2a2a4a 100%);
}

/* Tags */
.preview-tags-left {
  position: absolute;
  top: 6px; left: 6px;
  display: flex;
  gap: 4px;
  z-index: 2;
}
.preview-tags-right {
  position: absolute;
  top: 6px; right: 6px;
  display: flex;
  gap: 4px;
  z-index: 2;
}
.p-tag {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  background: rgba(0,0,0,0.7);
  color: #fff;
}
.p-tag :deep(.icon) {
  width: 12px !important; height: 12px !important;
}
.p-tag :deep(.mdi) {
  font-size: 12px !important;
  color: #fff !important;
}
.p-tag-info {
  background: rgba(0,0,0,0.7);
}
.p-tag-funscript {
  background: rgba(62, 142, 208, 0.95);
}
.p-tag-star {
  background: rgba(255, 221, 87, 0.95);
  color: rgba(0,0,0,0.7);
}
.p-tag-star :deep(.mdi) {
  color: rgba(0,0,0,0.7) !important;
}
.p-tag-duration {
  background: rgba(0,0,0,0.8);
}

/* Heatmap */
.preview-heatmap {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0 4px 4px;
  z-index: 2;
}
.preview-heatmap svg {
  display: block;
  border-radius: 3px;
  border: 1px solid rgba(255,255,255,0.25);
}

/* Buttons — always visible in preview */
.preview-actions {
  position: absolute;
  bottom: 6px; right: 6px;
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  justify-content: flex-end;
  z-index: 3;
}
.preview-heatmap ~ .preview-actions {
  bottom: 16px;
}
.p-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px; height: 28px;
  border-radius: 5px;
  color: #fff;
  cursor: default;
}
.p-btn-primary { background: #7957d5; }
.p-btn-danger { background: #f14668; }
.p-btn-info { background: #3e8ed0; }
.p-btn-dark { background: #363636; }
.p-btn :deep(.icon) {
  width: 16px !important; height: 16px !important;
}
.p-btn :deep(.mdi) {
  font-size: 16px !important;
  color: #fff !important;
}

/* Info section — fixed height */
.preview-info {
  padding: 7px 10px;
  height: 52px;
  box-sizing: border-box;
}
.preview-title-row {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 20px;
  overflow: hidden;
}
.preview-title {
  font-size: 14px;
  font-weight: 500;
  line-height: 1.3;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.preview-status-icons {
  display: flex;
  gap: 3px;
  flex-shrink: 0;
}
.pi-danger :deep(.mdi) { color: #f14668 !important; }
.pi-primary :deep(.mdi) { color: #7957d5 !important; }
.pi-dark :deep(.mdi) { color: #363636 !important; }
.preview-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  opacity: 0.7;
  height: 18px;
  gap: 8px;
}
.preview-site {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow: hidden;
}
.preview-site-icon {
  width: 16px; height: 16px;
  max-width: 16px; max-height: 16px;
  border-radius: 2px;
  object-fit: cover;
  flex-shrink: 0;
}
.preview-date {
  white-space: nowrap;
  flex-shrink: 0;
}

/* ── Dark mode ── */
html[data-theme="dark"] .preview-card {
  background: #1c1c26;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}
html[data-theme="dark"] .pi-dark :deep(.mdi) { color: #d4d4d8 !important; }

/* ── Responsive ── */
@media screen and (max-width: 768px) {
  .settings-layout {
    flex-direction: column-reverse;
  }
  .preview-col {
    flex: none;
    width: 100%;
    max-width: 280px;
    margin: 0 auto;
  }
  .preview-sticky {
    position: static;
  }
  .settings-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }
  .toggle-grid {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
