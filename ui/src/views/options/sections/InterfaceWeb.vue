<template>
  <OptionsSection :title="$t('Web UI')" :loading="isLoading">
    <div class="content">

      <div class="settings-layout">
        <div class="settings-col">

          <div class="settings-group">
            <h5 class="group-title">{{ $t('Scene Cards') }}</h5>
            <div class="settings-row">
              <div class="setting-item">
                <label class="setting-label">{{ $t('Aspect Ratio') }}</label>
                <b-select size="is-small" v-model="sceneCardAspectRatio">
                  <option>1:1</option>
                  <option>3:2</option>
                  <option>16:9</option>
                </b-select>
              </div>
              <div class="setting-item">
                <label class="setting-label">{{ $t('Date Format') }}</label>
                <b-select size="is-small" v-model="dateFormat">
                  <option value="yyyy-MM-dd">2025-01-15</option>
                  <option value="MMM d, yyyy">{{ $t('Jan 15, 2025') }}</option>
                  <option value="MMMM d, yyyy">{{ $t('January 15, 2025') }}</option>
                  <option value="dd/MM/yyyy">15/01/2025</option>
                  <option value="MM/dd/yyyy">01/15/2025</option>
                </b-select>
              </div>
            </div>
            <div class="settings-row" style="margin-top: 0.5rem">
              <div class="setting-item">
                <label class="setting-label">{{ $t('Buttons') }}</label>
                <b-select size="is-small" v-model="cardButtonStyle">
                  <option value="classic">{{ $t('Classic') }}</option>
                  <option value="glass">{{ $t('Liquid glass') }}</option>
                </b-select>
              </div>
            </div>
            <div class="settings-row" style="margin-top: 0.5rem">
              <div class="setting-item">
                <b-switch v-model="sceneCardScaleToFit" type="is-dark" size="is-small">
                  {{ $t('Scale to fit') }}
                </b-switch>
              </div>
              <div class="setting-item">
                <b-switch v-model="showSiteLogo" type="is-dark" size="is-small">
                  {{ $t('Show site logo') }}
                </b-switch>
              </div>
            </div>
            <div class="setting-item slider-item">
              <label class="setting-label">{{ $t('Unavailable scene opacity') }}</label>
              <div class="slider-wrap">
                <b-slider :min="0" :max="100" :step="10" :tooltip="false" v-model="isAvailOpacity" size="is-small"></b-slider>
                <span class="slider-value">{{ isAvailOpacity }}%</span>
              </div>
            </div>
          </div>

          <div class="settings-group">
            <h5 class="group-title">{{ $t('Upcoming Scenes') }}</h5>
            <div class="settings-row">
              <div class="setting-item">
                <b-switch v-model="showUpcoming" type="is-dark" size="is-small">
                  {{ $t('Show in scene list') }}
                </b-switch>
              </div>
              <div class="setting-item" v-if="showUpcoming">
                <label class="setting-label">{{ $t('Releasing within') }}</label>
                <b-select size="is-small" v-model="upcomingDays">
                  <option :value="7">{{ $t('1 week') }}</option>
                  <option :value="14">{{ $t('2 weeks') }}</option>
                  <option :value="30">{{ $t('1 month') }}</option>
                  <option :value="90">{{ $t('3 months') }}</option>
                  <option :value="0">{{ $t('Any time') }}</option>
                </b-select>
              </div>
            </div>
            <p class="setting-hint">{{ $t('Scenes VRBangers sites, VRSpy and SLR studios announce before release. Picked up by the next scrape.') }}</p>
          </div>

          <div class="settings-group">
            <h5 class="group-title">{{ $t('Visible Buttons') }}</h5>
            <div class="toggle-grid">
              <b-switch v-model="sceneHidden" type="is-danger" size="is-small">{{ $t('Hidden') }}</b-switch>
              <b-switch v-model="sceneWatchlist" type="is-default" size="is-small">{{ $t('Watchlist') }}</b-switch>
              <b-switch v-model="sceneFavourite" type="is-danger" size="is-small">{{ $t('Favourite') }}</b-switch>
              <b-switch v-model="sceneWishlist" type="is-info" size="is-small">{{ $t('Wishlist') }}</b-switch>
              <b-switch v-model="sceneTrailerlist" type="is-default" size="is-small">{{ $t('Trailer List') }}</b-switch>
              <b-switch v-model="sceneWatched" type="is-dark" size="is-small">{{ $t('Watched') }}</b-switch>
              <b-switch v-model="sceneEdit" type="is-dark" size="is-small">{{ $t('Edit') }}</b-switch>
              <b-switch v-model="sceneDuration" type="is-dark" size="is-small">{{ $t('Duration') }}</b-switch>
              <b-switch v-model="sceneScriptSpeed" type="is-dark" size="is-small">{{ $t('Script speed') }}</b-switch>
              <b-switch v-model="sceneCuepoint" type="is-dark" size="is-small">{{ $t('Cuepoints') }}</b-switch>
              <b-switch v-model="hspFile" type="is-dark" size="is-small">{{ $t('HSP File') }}</b-switch>
              <b-switch v-model="subtitlesFile" type="is-dark" size="is-small">{{ $t('Subtitles') }}</b-switch>
              <b-switch v-model="ScriptHeatmap" type="is-dark" size="is-small">{{ $t('Heatmap') }}</b-switch>
              <b-switch v-if="ScriptHeatmap" v-model="AllHeatmaps" type="is-dark" size="is-small">{{ $t('All Heatmaps') }}</b-switch>
              <b-switch v-model="openInNewWindow" type="is-dark" size="is-small">{{ $t('Open in New Window') }}</b-switch>
            </div>
          </div>

          <div class="settings-group">
            <h5 class="group-title">{{ $t('Actor Cards') }}</h5>
            <div class="settings-row">
              <div class="setting-item">
                <label class="setting-label">{{ $t('Aspect Ratio') }}</label>
                <b-select size="is-small" v-model="actorCardAspectRatio">
                  <option>1:1</option>
                  <option>2:3</option>
                  <option>9:16</option>
                </b-select>
              </div>
              <div class="setting-item">
                <b-switch v-model="actorCardScaleToFit" type="is-dark" size="is-small">
                  {{ $t('Scale to fit') }}
                </b-switch>
              </div>
            </div>
          </div>

          <div class="settings-group">
            <h5 class="group-title">{{ $t('General') }}</h5>
            <div class="settings-row">
              <div class="setting-item">
                <label class="setting-label">{{ $t('Tag Sort') }}</label>
                <b-select size="is-small" v-model="tagSort">
                  <option value="by-tag-count">{{ $t('By Count') }}</option>
                  <option value="alphabetically">A-Z</option>
                </b-select>
              </div>
              <div class="setting-item">
                <label class="setting-label">{{ $t('Theme') }}</label>
                <b-select size="is-small" v-model="theme">
                  <option value="auto">{{ $t('Auto') }}</option>
                  <option value="dark">{{ $t('Dark') }}</option>
                  <option value="light">{{ $t('Light') }}</option>
                </b-select>
              </div>
              <div class="setting-item language-item">
                <label class="setting-label">{{ $t('Language') }}</label>
                <b-select size="is-small" v-model="language">
                  <option value="">{{ $t('Auto') }}</option>
                  <option v-for="l in languages" :key="l.code" :value="l.code">{{ l.name }}</option>
                </b-select>
                <p class="language-hint">
                  {{ $t('Missing or wrong translation?') }}
                  <a href="https://github.com/f0r3s1/xbvr/tree/master/ui/src/locales" target="_blank" rel="noopener">{{ $t('Suggest a fix on GitHub') }}</a>
                </p>
              </div>
              <div class="setting-item">
                <b-switch v-model="accentColorTags" type="is-dark" size="is-small">
                  {{ $t('Accent color tags') }}
                </b-switch>
              </div>
              <div class="setting-item">
                <b-switch v-model="updateCheck" type="is-dark" size="is-small">
                  {{ $t('Check for updates') }}
                </b-switch>
              </div>
            </div>
          </div>
        </div>

        <div class="preview-col">
          <div class="preview-sticky">
            <h5 class="group-title">{{ $t('Preview') }}</h5>
            <div class="preview-card-demo">
              <SceneCard v-if="sampleScene" :item="sampleScene" demo/>
            </div>
          </div>
        </div>
      </div>

    </div>
  </OptionsSection>
</template>

<script>
import OptionsSection from '../OptionsSection.vue'
import SceneCard from '../../scenes/SceneCard.vue'
import { defineComponent } from 'vue';

import ky from 'ky'
import { availableLocales, chosenLanguage, setLanguage } from '@/i18n'

let saveTimer = null

export default defineComponent({
  name: 'InterfaceWeb',
  components: { OptionsSection, SceneCard },

  data () {
    return {
      chosenLocale: chosenLanguage(),
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
      } catch { /* no scenes available */ }
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
    languages () {
      return availableLocales.map(code => {
        let name = code
        try { name = new Intl.DisplayNames([code], { type: 'language' }).of(code) } catch { /* unknown code: show it as is */ }
        return { code, name }
      })
    },
    language: {
      get () { return this.chosenLocale || '' },
      set (code) {
        setLanguage(code || null)
        this.chosenLocale = code || null
      }
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
    sceneScriptSpeed: {
      get () { return this.$store.state.optionsWeb.web.sceneScriptSpeed },
      set (value) { this.$store.state.optionsWeb.web.sceneScriptSpeed = value }
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
    cardButtonStyle: {
      get () { return this.$store.state.optionsWeb.web.cardButtonStyle },
      set (value) { this.$store.state.optionsWeb.web.cardButtonStyle = value }
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
    showUpcoming: {
      get () { return this.$store.state.optionsWeb.web.showUpcoming !== false },
      set (value) { this.$store.state.optionsWeb.web.showUpcoming = value }
    },
    upcomingDays: {
      get () { return this.$store.state.optionsWeb.web.upcomingDays ?? 14 },
      set (value) { this.$store.state.optionsWeb.web.upcomingDays = value }
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
.preview-card-demo {
  pointer-events: none;
}
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
.language-item {
  flex-wrap: wrap;
}
.language-hint {
  flex-basis: 100%;
  margin: 0;
  font-size: 0.75rem;
  opacity: 0.6;
}
.setting-label {
  font-size: 0.85rem;
  font-weight: 500;
  white-space: nowrap;
}
.setting-hint {
  margin: 0.4rem 0 0;
  font-size: 0.78rem;
  opacity: 0.65;
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
.preview-status-icons {
  display: flex;
  gap: 3px;
  flex-shrink: 0;
}
.pi-danger :deep(.mdi) { color: #f14668 !important; }
.pi-primary :deep(.mdi) { color: #7957d5 !important; }
.pi-dark :deep(.mdi) { color: #363636 !important; }
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
