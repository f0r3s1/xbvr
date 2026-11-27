<template>
  <div class="scene-card" :class="{ 'is-fresh': item.fresh, 'is-morph': morphing }"
       @mouseenter="hovering = true; startPreview()" 
       @mouseleave="hovering = demo; demo || stopPreview()"
       @click="!demo && showDetails(item)">
    
    <GlassMorph v-if="isGlass && item.has_preview" :active="morphing" :source="() => $refs.previewVideo" :bar="() => $refs.hoverBar" @ready="glassReady = $event"/>

    <transition name="fade">
      <div class="preview-overlay" v-if="preview && item.has_preview" @click="showDetails(item)">
        <video ref="previewVideo" :src="`/api/dms/preview/${item.scene_id}`" autoplay loop muted></video>
      </div>
    </transition>

    <div class="thumbnail-wrapper" :style="{ aspectRatio: sceneCardAspectRatioValue }">
      <div class="thumbnail-img" :style='{ opacity: item.is_available ? 1.0 : isAvailOpactiy }'>
        <img v-if="!imageFailed" decoding="async"
          ref="coverImage"
          v-queued-src="coverImageSrc"
          @error="onImageError" @queue-slow="onImageSlow"
          @load="onImageLoad"
          class="cover-image"
          :class="{ 'is-loaded': imageLoaded }"
          :style="{ objectFit: sceneCardScaleToFit ? 'contain' : 'cover' }"
        />
        <span v-if="item.cover_url && !imageLoaded && !imageFailed" class="cover-loading skeleton-block" aria-hidden="true"></span>
        <span v-if="imageFailed || !item.cover_url" class="cover-missing" :title="$t('No image')">
          <b-icon pack="mdi" icon="image-off-outline"/>
        </span>
      </div>
      
      <div class="tags-overlay">
        <b-tag type="is-info" v-if="videoFilesCount > 1 && !item.is_multipart" class="mini-tag">
          <b-icon pack="mdi" icon="file" size="is-small"/>
          <span>{{videoFilesCount}}</span>
        </b-tag>
        <b-tag v-if="item.is_scripted" class="mini-tag funscript-tag" :title="scriptBadgeTitle">
          <b-icon pack="mdi" :icon="aiScripts === 'all' ? 'robot-outline' : 'pulse'" size="is-small"/>
          <span v-if="scriptFilesCount > 1">{{scriptFilesCount}}</span>
        </b-tag>
        <b-tag type="is-info" v-if="hspFilesCount > 0 && this.$store.state.optionsWeb.web.showHspFile" class="mini-tag">
          <b-icon pack="mdi" icon="safety-goggles" size="is-small"/>
          <span v-if="hspFilesCount > 1">{{hspFilesCount}}</span>
        </b-tag>
        <b-tag type="is-info" v-if="subtitlesFilesCount > 0 && this.$store.state.optionsWeb.web.showSubtitlesFile" class="mini-tag">
          <b-icon pack="mdi" icon="subtitles" size="is-small"/>
          <span v-if="subtitlesFilesCount > 1">{{subtitlesFilesCount}}</span>
        </b-tag>
        <b-tag type="is-info" v-if="cuepointCount > 0 && $store.state.optionsWeb.web.sceneCuepoint" class="mini-tag">
          <b-icon pack="mdi" icon="skip-next-outline" size="is-small"/>
          <span v-if="cuepointCount > 1">{{ cuepointCount }}</span>
        </b-tag>
      </div>

      <!-- Upcoming: announced by the site, not released yet -->
      <span v-if="upcomingLabel" class="upcoming-badge">{{ upcomingLabel }}</span>

      <div class="top-right-tags">
        <b-tag type="is-warning" v-if="item.star_rating > 0" class="mini-tag star-tag">
          <b-icon pack="mdi" icon="star" size="is-small"/>
          {{item.star_rating}}
        </b-tag>
        <b-tag v-if="scriptSpeed && $store.state.optionsWeb.web.sceneScriptSpeed" class="mini-tag duration-tag" :title="$t('Average speed of the selected script')">
          {{ $t('{speed} avg', { speed: scriptSpeed }) }}
        </b-tag>
        <b-tag v-if="item.duration > 0 && this.$store.state.optionsWeb.web.sceneDuration" class="mini-tag duration-tag">
          {{item.duration}}m
        </b-tag>
      </div>

      <div v-if="this.$store.state.optionsWeb.web.showScriptHeatmap && (files = getFunscripts(this.$store.state.optionsWeb.web.showAllHeatmaps))" class="heatmap-strip">
        <div v-if="files.length" class="heatmap-stack" :class="{'single-heatmap': files.length === 1, 'multi-heatmap': files.length > 1}">
          <img decoding="async" v-for="file in files" :key="file.id" v-queued-src="getHeatmapURL(file.id)"/>
        </div>
      </div>


      <div v-if="hovering" ref="hoverBar" class="hover-actions" :class="{ 'is-visible': morphing ? glassReady : hovering, 'is-preview': preview && item.has_preview, 'is-glass': isGlass, 'is-morph': morphing }" @click.stop @mouseover="anchorTooltip">
        <hidden-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneHidden"/>
        <watchlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
        <trailerlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneTrailerlist"/>
        <favourite-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
        <wishlist-button v-if="this.$store.state.optionsWeb.web.sceneWishlist && !item.is_available" :item="item"/>
        <watched-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatched"/>
        <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit"/>
        <link-stashdb-button :item="item" v-if="!this.stashLinkExists" objectType="scene"/>
        <b-tooltip v-for="(altsrc, idx) in alternateSources" :key="idx" type="is-light" :label="altsrc.title" :delay="100" append-to-body>
          <a :href="altsrc.url" target="_blank" class="alt-link" @click.stop>
            <vue-load-image>
              <template #image><img loading="lazy" decoding="async" :src="getImageURL(altsrc.site_icon, '64x')" class="alt-img"/></template>
              <template #error><b-icon pack="mdi" icon="link" size="is-small"/></template>
            </vue-load-image>
          </a>
        </b-tooltip>
      </div>
    </div>

    <div class="info-section" :class="{ 'is-preview': preview && item.has_preview }">
      <div class="title-row">
        <div class="scene-title">{{item.title}}</div>
        <span v-if="hasActiveStatus" class="status-icons">
          <span v-if="item.is_hidden" class="status-icon status-danger" :data-tooltip="$t('Hidden')">
            <b-icon pack="mdi" icon="eye-off" size="is-small"/>
          </span>
          <span v-if="item.watchlist" class="status-icon status-primary" :data-tooltip="$t('Watchlist')">
            <b-icon pack="mdi" icon="calendar-check" size="is-small"/>
          </span>
          <span v-if="item.trailerlist && !item.is_available" class="status-icon status-primary" :data-tooltip="$t('Trailer List')">
            <b-icon pack="mdi" icon="movie-open-check" size="is-small"/>
          </span>
          <span v-if="item.favourite" class="status-icon status-danger" :data-tooltip="$t('Favourite')">
            <b-icon pack="mdi" icon="heart" size="is-small"/>
          </span>
          <span v-if="item.wishlist" class="status-icon status-info" :data-tooltip="$t('Wishlist')">
            <b-icon pack="mdi" icon="oil-lamp" size="is-small"/>
          </span>
          <span v-if="item.is_watched" class="status-icon status-dark" :data-tooltip="$t('Watched')">
            <b-icon pack="mdi" icon="eye-check" size="is-small"/>
          </span>
        </span>
      </div>
      <div class="meta-row">
        <span class="site-link">
          <SiteIcon v-if="showSiteLogo && item.site" :site="site" :name="item.site"/>
          <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer" @click.stop>
            <b-icon pack="mdi" icon="link-lock" custom-size="mdi-14px"/>
          </a>
          <a :href="item.scene_url" :class="{'site-subscribed': item.is_subscribed}" target="_blank" rel="noreferrer" @click.stop>{{item.site}}</a>
          <a v-if="stashLinkExists" :href="getStashdbUrl()" target="_blank" class="stashdb-link" @click.stop>
            <img loading="lazy" decoding="async" src="https://guidelines.stashdb.org/favicon.ico" class="stashdb-icon" alt="StashDB"/>
          </a>
        </span>
        <span v-if="hasDate(item.release_date)" class="release-date">
          {{format(utcDay(item.release_date), dateFormat)}}
        </span>
      </div>
    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';
import SiteIcon from '@/components/SiteIcon.vue'
import GlassMorph from '@/components/GlassMorph.vue'
import { imageURL } from '@/imageURL'

import { format, parseISO } from 'date-fns'
import { utcDay, hasDate } from '@/utcDay'
import WatchlistButton from '../../components/WatchlistButton'
import FavouriteButton from '../../components/FavouriteButton'
import WishlistButton from '../../components/WishlistButton'
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'
import LinkStashdbButton from '../../components/LinkStashdbButton'
import TrailerlistButton from '../../components/TrailerlistButton'
import HiddenButton from '../../components/HiddenButton'
import ky from 'ky'
import { getAlternateSources } from '@/api'
import VueLoadImage from 'vue-load-image'
import { isAiScript } from '@/scripts'

export default defineComponent({
  name: 'SceneCard',
  props: { item: Object, reRead: Boolean, demo: Boolean },
  components: { SiteIcon, GlassMorph, WatchlistButton, FavouriteButton, WishlistButton, WatchedButton, EditButton, LinkStashdbButton, TrailerlistButton, HiddenButton, VueLoadImage },

  data () {
    return {
      preview: false,
      hovering: this.demo,
      glassReady: false,
      format,
      parseISO,
      alternateSources: [],
      stashLinkExists: false,
      loadDirect: false,
      proxyRetried: false,
      imageFailed: false,
      imageLoaded: false,
    }
  },

  mounted () {
    if (this.demo) this.startPreview()
    this.loadAlternateSources()
  },

  computed: {
    cuepointCount () {
      return this.item.cuepoint_count ?? (this.item.cuepoints || []).length
    },
    scriptSpeed () {
      const scripts = (this.item.file || []).filter(f => f.type === 'script' && f.script_avg_speed > 0)
      const script = scripts.find(f => f.is_selected_script) || scripts[0]
      return script ? Math.round(script.script_avg_speed) : 0
    },
    videoFilesCount () {
      if (this.item.file == null) { return 0 }
      return this.item.file.filter(obj => obj.type === 'video').length
    },
    // 'all' / 'some' / 'none' of the local scripts are AI-generated (by file name)
    aiScripts () {
      const scripts = (this.item.file || []).filter(f => f.type === 'script')
      const ai = scripts.filter(f => isAiScript(f.filename)).length
      if (!ai) return 'none'
      return ai === scripts.length ? 'all' : 'some'
    },
    // "AI script · 212 avg speed · 84 strokes/min" (selected script, else first)
    scriptBadgeTitle () {
      const scripts0 = (this.item.file || []).filter(f => f.type === 'script')
      const nAi = scripts0.filter(f => isAiScript(f.filename)).length
      const kind = this.aiScripts === 'all' ? (scripts0.length > 1 ? this.$t('{n} AI scripts', { n: scripts0.length }) : this.$t('AI script'))
        : this.aiScripts === 'some' ? this.$t('{n} scripts, {ai} AI', { n: scripts0.length, ai: nAi }) : (scripts0.length > 1 ? this.$t('{n} scripts', { n: scripts0.length }) : this.$t('Script'))
      const scripts = (this.item.file || []).filter(f => f.type === 'script')
      const s = scripts.find(f => f.is_selected_script) || scripts[0]
      if (!s || !s.script_actions) return kind
      return `${kind} · ${this.$t('{n} avg speed', { n: Math.round(s.script_avg_speed) })} · ${this.$t('{n} strokes/min', { n: Math.round(s.script_strokes_per_min) })}`
    },
    scriptFilesCount () {
      if (this.item.file == null) { return 0 }
      return this.item.file.filter(obj => obj.type === 'script').length
    },
    hspFilesCount () {
      if (this.item.file == null) { return 0 }
      return this.item.file.filter(obj => obj.type === 'hsp').length
    },
    subtitlesFilesCount () {
      if (this.item.file == null) { return 0 }
      return this.item.file.filter(obj => obj.type === 'subtitles').length
    },
    isAvailOpactiy () {
      if (this.$store.state.optionsWeb.web.isAvailOpacity == undefined) {
        return .4
      }
      return this.$store.state.optionsWeb.web.isAvailOpacity / 100
    },
    sceneCardAspectRatioValue () {
      const r = this.$store.state.optionsWeb.web.sceneCardAspectRatio || '16:9'
      const parts = r.split(':')
      if (parts.length === 2) return `${parts[0]} / ${parts[1]}`
      return '16 / 9'
    },
    sceneCardScaleToFit () {
      return this.$store.state.optionsWeb.web.sceneCardScaleToFit
    },
    showSiteLogo () {
      return this.$store.state.optionsWeb.web.showSiteLogo !== false
    },
    dateFormat () {
      return this.$store.state.optionsWeb.web.dateFormat || 'yyyy-MM-dd'
    },
    // the liquid glass morph runs over a playing preview; other cards keep the plain glass bar
    morphing () {
      return this.isGlass && this.hovering && this.preview && this.item.has_preview
    },
    isGlass () {
      return this.$store.state.optionsWeb.web.cardButtonStyle === 'glass'
    },
    site () {
      return (this.$store.state.optionsSites.items || []).find(s => s.id === this.item.scraper_id) || null
    },
    // "Upcoming in 5 days"; sites without a date only say "Upcoming"
    upcomingLabel () {
      if (!this.item.is_upcoming) return ''
      const d = this.item.release_date
      if (!hasDate(d)) return this.$t('Upcoming')
      const release = utcDay(d)
      const today = new Date()
      today.setHours(0, 0, 0, 0)
      const days = Math.round((release.setHours(0, 0, 0, 0) - today) / 86400000)
      if (days < 0) return this.$t('Upcoming')
      if (days === 0) return this.$t('Upcoming today')
      return days === 1 ? this.$t('Upcoming tomorrow') : this.$t('Upcoming in {n} days', { n: days })
    },
    hasActiveStatus () {
      return this.item.is_hidden || this.item.favourite || this.item.is_watched ||
             this.item.watchlist || this.item.wishlist || (this.item.trailerlist && !this.item.is_available)
    },
    coverImageSrc () {
      const u = this.item.cover_url
      if (!u || !u.startsWith('http') || this.loadDirect) return u
      // XS-M cards are at most ~350 css px wide, so 700 px stays sharp on 2x screens
      const width = Number(this.$store.state.sceneList.filters.cardSize) >= 4 ? '1200x' : '700x'
      const url = `/img/${width}/` + encodeURI(u.includes('%') ? decodeURI(u) : u)
      return url
    }
  },

  beforeUnmount () {
    clearTimeout(this.directTimer)
  },

  methods: {
    utcDay,
    hasDate,
    // a tooltip opens to the right unless it would pass the card's edge
    anchorTooltip (event) {
      const el = event.target.closest('[data-tooltip]')
      if (!el) return
      const card = event.currentTarget.parentElement.getBoundingClientRect()
      const width = el.getBoundingClientRect().left + parseFloat(getComputedStyle(el, '::after').width) + 16
      el.classList.toggle('tip-right', width > card.right)
    },
    // our server is slow on this cover: load the original from its site instead of waiting
    onImageSlow () {
      this.fallBack()
    },
    onImageError (event) {
      if (!event?.target?.getAttribute('src')) return // cancelled by the image queue, not a real failure
      this.fallBack()
    },
    fallBack () {
      // some sites refuse server-side fetches but serve the browser, so load the original directly
      if (!this.loadDirect && this.item.cover_url?.startsWith('http')) {
        this.loadDirect = true
        this.imageLoaded = false
        this.directTimer = setTimeout(() => { if (!this.imageLoaded) this.fallBack() }, 20000)
      } else if (this.loadDirect && !this.proxyRetried) {
        // the server keeps preparing an image the browser gave up on, so ask it once more
        this.proxyRetried = true
        clearTimeout(this.directTimer)
        this.directTimer = setTimeout(() => { this.loadDirect = false }, 4000)
      } else {
        this.imageFailed = true
        if (this.item && this.item.id) {
          fetch('/api/health/image-error', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ scene_id: this.item.id }),
          }).catch(() => {})
        }
      }
    },
    onImageLoad () {
      const tooSmall = !this.$refs.coverImage || this.$refs.coverImage.naturalWidth <= 100
      if (tooSmall && !this.loadDirect) {
        this.onImageError()
        return
      }
      this.imageLoaded = true
    },
    async loadAlternateSources () {
      this.stashLinkExists = false
      try {
        const response = await getAlternateSources(this.item.id)
        if (response == null) return

        this.alternateSources = response
          .filter(altsrc => altsrc.external_source.startsWith("alternate scene ") || altsrc.external_source == "stashdb scene")
          .map(altsrc => {
            const extdata = JSON.parse(altsrc.external_data)
            let title
            if (altsrc.external_source.startsWith("alternate scene ")) {
              title = extdata.scene?.title || this.$t('No Title')
            } else if (altsrc.external_source == "stashdb scene") {
              title = extdata.title || this.$t('No Title')
            }
            if (altsrc.external_source.includes('stashdb')) {
              this.stashLinkExists = true
            }
            return { ...altsrc, title }
          })
      } catch { /* silent */ }
    },
    startPreview () {
      if (this.item.has_preview) {
        this.preview = true
      }
    },
    stopPreview () {
      this.preview = false
    },
    getStashdbUrl () {
      const stashdbSource = this.alternateSources.find(s => s.external_source === 'stashdb scene')
      return stashdbSource ? stashdbSource.url : '#'
    },
    getImageURL (u, size = '1200x') {
      return imageURL(u, size)
    },
    showDetails (scene) {
      // a card opened from ActorDetails lacks preloaded tags and cast, so reread the scene
      if (this.reRead) {
        ky.get('/api/scene/'+scene.id).json().then(data => {
          if (data.id != 0){
            this.$store.commit('overlay/showDetails', { scene: data })
          }
        })
      } else {
        this.$store.dispatch('overlay/openDetails', { scene })
      }
      this.$store.commit('overlay/hideActorDetails')
    },
    getHeatmapURL (fileId) {
      return `/api/dms/heatmap/${fileId}`
    },
    getFunscripts (showAll) {
      if (showAll) {
        return this.item.file !== null && this.item.file.filter(a => a.type === 'script' && a.has_heatmap);
      } else {
        if (this.item.file !== null) {
          let script
          if ((script = this.item.file.find((a) => a.type === 'script' && a.has_heatmap && a.is_selected_script))) {
            return [script]
          }
          if ((script = this.item.file.find((a) => a.type === 'script' && a.has_heatmap))) {
            return [script]
          }
        }
        return false;
      }
    }
  },
});
</script>

<style scoped>
.scene-card.is-fresh {
  animation: fresh-in 2.5s ease-out;
}
@keyframes fresh-in {
  0% { opacity: 0; transform: translateY(-6px); box-shadow: 0 0 0 2px #7957d5; }
  15% { opacity: 1; transform: none; box-shadow: 0 0 0 2px #7957d5; }
  100% { box-shadow: 0 0 0 2px transparent; }
}

.scene-card:has(.hover-actions) .upcoming-badge {
  opacity: 0;
}
/* Card Container - Light UI */
.scene-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  transition: box-shadow 0.2s ease;
  cursor: pointer;
  isolation: isolate;
  position: relative;
}

.scene-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 5;
}

/* Preview Overlay - covers the whole card, sits below info section */
.preview-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 20;
  background: #000;
  overflow: hidden;
  border-radius: inherit;
}

.preview-overlay video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
/* liquid glass: the bar sits on the glass shape, which draws its own surface; text stays readable on media */
.hover-actions.is-morph {
  left: auto;
  right: 0;
  bottom: 0;
  background: none !important;
  box-shadow: none !important;
  -webkit-backdrop-filter: none !important;
  backdrop-filter: none !important;
  padding: 3px 6px;
  transition: opacity 0.2s ease, transform 0.25s ease;
  transform: translateY(4px);
}
.hover-actions.is-morph.is-visible {
  transform: none;
}
.scene-card.is-morph .info-section {
  z-index: 22;
  background: transparent !important;
  -webkit-backdrop-filter: none !important;
  backdrop-filter: none !important;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.7);
}
.scene-card.is-morph .info-section::before {
  display: none;
}
/* the morph draws the whole glass; the classic bar's bottom shade would darken the button area twice */
.scene-card.is-morph .thumbnail-wrapper::after {
  display: none !important;
}
.scene-card.is-morph .scene-title {
  color: #fff !important;
}
.scene-card.is-morph .meta-row,
.scene-card.is-morph .site-link a,
.scene-card.is-morph .release-date {
  color: rgba(255, 255, 255, 0.82) !important;
}


/* Fade transition — quick crossfade */
.fade-enter-active {
  transition: opacity 0.2s ease-out;
}
.fade-leave-active {
  transition: opacity 0.1s ease-in;
}
.fade-enter, .fade-leave-to {
  opacity: 0;
}

/* Thumbnail - aspect ratio set via inline style from settings */
/* The wrapper must not clip: the hover-button tooltips have to be able to
   extend past the card edge. The image and preview layers round and clip
   themselves instead. */
.thumbnail-wrapper {
  position: relative;
  width: 100%;
  border-radius: 8px 8px 0 0;
  flex-shrink: 0;
}

.thumbnail-img {
  background: var(--card-placeholder);
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  transition: transform 0.3s ease;
  overflow: hidden;
  border-radius: inherit;
}

.thumbnail-img .cover-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center center;
  display: block;
  transition: filter 0.5s ease, opacity 0.3s ease;
  filter: blur(15px);
}

.thumbnail-img .cover-image.is-loaded {
  filter: blur(0);
  opacity: 1;
}
.thumbnail-img .cover-image:not(.is-loaded) {
  opacity: 0;
}
.cover-loading {
  position: absolute;
  inset: 0;
  border-radius: 0;
}
.cover-missing {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.6rem;
  opacity: 0.35;
}


/* Tags Overlay - Top Left */
.tags-overlay {
  position: absolute;
  top: 6px;
  left: 6px;
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
  pointer-events: none;
  z-index: 3;
  max-width: 70%;
}

/* Top Right Tags */
.top-right-tags {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 3px;
  pointer-events: none;
  z-index: 3;
}

.mini-tag {
  font-size: 10px !important;
  padding: 4px 6px !important;
  height: 22px !important;
  line-height: 1 !important;
  background: rgba(0, 0, 0, 0.7) !important;
  color: #fff !important;
  backdrop-filter: blur(4px);
  border: none !important;
  border-radius: 4px !important;
  font-weight: 500 !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  gap: 3px !important;
}

.mini-tag.star-tag {
  background: rgba(255, 221, 87, 0.95) !important;
  color: rgba(0,0,0,0.7) !important;
}

.upcoming-badge {
  position: absolute;
  left: 6px;
  bottom: 6px;
  z-index: 3;
  pointer-events: none;
  padding: 3px 7px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.75);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.02em;
  backdrop-filter: blur(4px);
}

.mini-tag.duration-tag {
  background: rgba(0, 0, 0, 0.8) !important;
  color: #fff !important;
}

/* Funscript Tag - matching info blue */
.mini-tag.funscript-tag {
  background: rgba(62, 142, 208, 0.95) !important;
  color: #fff !important;
}

.mini-tag .icon {
  margin: 0 !important;
  width: 12px !important;
  height: 12px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

.mini-tag .icon .mdi {
  font-size: 12px !important;
  line-height: 1 !important;
}

.mini-tag span {
  line-height: 1 !important;
}

/* Heatmap Strip */
.heatmap-strip {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 0 6px 5px;
  z-index: 3;
  transition: opacity 0.15s ease;
}
.scene-card:hover .heatmap-strip {
  opacity: 0;
}

.heatmap-stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* Single heatmap - taller */
.heatmap-stack.single-heatmap img {
  width: 100%;
  height: 14px;
  border-radius: 3px;
  border: 1px solid rgba(255, 255, 255, 0.25);
}

/* Multiple heatmaps - smaller to fit */
.heatmap-stack.multi-heatmap img {
  width: 100%;
  height: 6px;
  border-radius: 2px;
  border: 1px solid rgba(255, 255, 255, 0.25);
}

/* Default fallback */
.heatmap-stack img {
  width: 100%;
  height: 8px;
  border-radius: 3px;
  border: 1px solid rgba(255, 255, 255, 0.25);
}

/* Status icons - inline next to title */
.status-icons {
  display: inline-flex;
  gap: 3px;
  align-items: center;
  flex-shrink: 0;
}
.status-icon {
  position: relative;
  width: 14px !important;
  height: 14px !important;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: default;
}
.status-icon :deep(.icon) { width: 14px !important; height: 14px !important; }
.status-icon :deep(.mdi) { font-size: 14px !important; color: inherit !important; }
.status-icon.status-primary,
.status-icon.status-primary:hover { color: #7957d5 !important; }
.status-icon.status-danger,
.status-icon.status-danger:hover { color: #f14668 !important; }
.status-icon.status-info,
.status-icon.status-info:hover { color: #3e8ed0 !important; }
.status-icon.status-dark,
.status-icon.status-dark:hover { color: #363636 !important; }
html[data-theme="dark"] .status-icon.status-dark,
html[data-theme="dark"] .status-icon.status-dark:hover { color: #d4d4d8 !important; }

/* CSS tooltips for status icons — same style as button tooltips */
.status-icon[data-tooltip]::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 6px);
  right: 0;
  padding: 4px 8px;
  background: rgba(0,0,0,0.85);
  color: #fff;
  font-size: 11px;
  font-weight: 400;
  white-space: nowrap;
  border-radius: 4px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.1s ease;
  z-index: 100;
}
.status-icon[data-tooltip]:hover::after {
  opacity: 1;
}

/* Hover Actions - positioned at bottom-right of thumbnail */
.hover-actions {
  position: absolute;
  bottom: 6px;
  left: 6px;
  right: 6px;
  display: flex;
  flex-wrap: wrap-reverse;
  justify-content: flex-end;
  gap: 4px;
  align-items: flex-end;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease;
  z-index: 25;
}

.hover-actions.is-visible {
  opacity: 1;
  pointer-events: auto;
}



/* Kill margins on direct children */
.hover-actions > * {
  margin: 0 !important;
}

/* Base button reset — fixed 28x28, solid filled */
.hover-actions :deep(.button) {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  min-height: 28px !important;
  max-width: 28px !important;
  max-height: 28px !important;
  border-radius: 5px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  line-height: 1 !important;
  box-sizing: border-box !important;
  margin: 0 !important;
  padding: 0 !important;
  border: none !important;
  background: #fff !important;
  color: #363636 !important;
}
/* Solid colors — active = full Bulma color, outlined = same color at 70% opacity */
.hover-actions:not(.is-glass) :deep(.button.is-primary),
.hover-actions:not(.is-glass) :deep(.button.is-primary.is-outlined) {
  background: #7957d5 !important; color: #fff !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-primary.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-primary.is-outlined:hover) { opacity: 1; }

.hover-actions:not(.is-glass) :deep(.button.is-danger),
.hover-actions:not(.is-glass) :deep(.button.is-danger.is-outlined) {
  background: #f14668 !important; color: #fff !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-danger.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-danger.is-outlined:hover) { opacity: 1; }

.hover-actions:not(.is-glass) :deep(.button.is-info),
.hover-actions:not(.is-glass) :deep(.button.is-info.is-outlined) {
  background: #3e8ed0 !important; color: #fff !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-info.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-info.is-outlined:hover) { opacity: 1; }

.hover-actions:not(.is-glass) :deep(.button.is-link),
.hover-actions:not(.is-glass) :deep(.button.is-link.is-outlined) {
  background: #485fc7 !important; color: #fff !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-link.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-link.is-outlined:hover) { opacity: 1; }

.hover-actions:not(.is-glass) :deep(.button.is-warning),
.hover-actions:not(.is-glass) :deep(.button.is-warning.is-outlined) {
  background: #ffdd57 !important; color: rgba(0,0,0,0.7) !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-warning.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-warning.is-outlined:hover) { opacity: 1; }

.hover-actions:not(.is-glass) :deep(.button.is-success),
.hover-actions:not(.is-glass) :deep(.button.is-success.is-outlined) {
  background: #48c78e !important; color: #fff !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-success.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-success.is-outlined:hover) { opacity: 1; }

.hover-actions:not(.is-glass) :deep(.button.is-dark),
.hover-actions:not(.is-glass) :deep(.button.is-dark.is-outlined) {
  background: #363636 !important; color: #fff !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-dark.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-dark.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button .icon) {
  margin: 0 !important;
  width: 16px !important;
  height: 16px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

.hover-actions :deep(.button .icon .mdi) {
  font-size: 16px !important;
  line-height: 1 !important;
}

.hover-actions :deep(.button span:not(.icon)) {
  display: none !important;
}

/* Alt Source Images */
.alt-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  min-width: 28px;
  min-height: 28px;
  background: #fff;
  border-radius: 5px;
  transition: all 0.15s ease;
  box-sizing: border-box;
}
.alt-link:hover {
  background: rgba(127, 127, 127, 0.18);
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
}

/* Preview mode alt-links — keep solid */
.hover-actions.is-preview .alt-link {
  background: #fff;
  box-shadow: 0 1px 4px rgba(0,0,0,0.3);
}
.hover-actions.is-preview .alt-link:hover {
  background: #e8e8e8;
}

.alt-img {
  width: 18px;
  height: 18px;
  border-radius: 2px;
  object-fit: cover;
  display: block;
}

/* Info Section — fixed height so all cards match */
.info-section {
  position: relative;
  z-index: 21;
  padding: 7px 10px;
  height: 52px;
  box-sizing: border-box;
  transition: background 0.2s ease, color 0.2s ease;
  background: transparent;
  border-radius: 0 0 8px 8px;
}
.info-section:has(.status-icon:hover) {
  z-index: 50;
}

/* Preview mode: frosted glass over video */
/* over a playing preview the info fades in from above: darkening and blur ramp up, no visible edge */
.info-section.is-preview::before {
  content: '';
  position: absolute;
  inset: -36px 0 0 0;
  z-index: -1;
  border-radius: inherit;
  pointer-events: none;
  background: linear-gradient(to bottom, rgba(0, 0, 0, 0) 0%, rgba(0, 0, 0, 0.45) 45%, rgba(0, 0, 0, 0.7) 100%);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  mask-image: linear-gradient(to bottom, transparent 0%, #000 55%);
  -webkit-mask-image: linear-gradient(to bottom, transparent 0%, #000 55%);
}


.title-row {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  height: 20px;
}

.scene-title {
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary, #333);
  line-height: 1.3;
  flex: 1;
  min-width: 0;
  transition: color 0.2s ease;
}

.info-section.is-preview .scene-title {
  color: #fff !important;
}


.meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-secondary, #666);
  min-width: 0;
  height: 18px;
  gap: 8px;
}


.info-section.is-preview .meta-row {
  color: rgba(255,255,255,0.7);
}

.info-section.is-preview .site-link a {
  color: rgba(255,255,255,0.8);
}

.info-section.is-preview .release-date {
  color: rgba(255,255,255,0.5);
}

.site-link {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.site-link > div,
.site-link .stashdb-link {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  line-height: 0;
}

.site-link a {
  color: inherit !important;
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.site-link a:hover {
  color: inherit !important;
  text-decoration: underline;
}

.site-subscribed {
  background: var(--primary, #7957d5);
  color: white !important;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 9px;
}

/* Stashdb link in info */
.stashdb-link {
  display: inline-flex;
  align-items: center;
  margin-left: 2px;
}

.stashdb-icon {
  width: 14px;
  height: 14px;
  opacity: 0.7;
  transition: opacity 0.15s;
}

.stashdb-link:hover .stashdb-icon {
  opacity: 1;
}

.release-date {
  opacity: 0.6;
  font-size: 10px;
  white-space: nowrap;
  flex-shrink: 0;
}

/* Instant CSS tooltips for buttons */
.hover-actions :deep([data-tooltip]) {
  position: relative;
}

.hover-actions :deep([data-tooltip])::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 6px);
  right: 0;
  padding: 4px 8px;
  background: rgba(0,0,0,0.85);
  color: #fff;
  font-size: 11px;
  font-weight: 400;
  white-space: nowrap;
  border-radius: 4px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.1s ease;
  z-index: 30;
}

.hover-actions :deep([data-tooltip]:hover)::after {
  opacity: 1;
}

.hover-actions :deep([data-tooltip]:not(.tip-right))::after {
  left: 0;
  right: auto;
}

/* ── Dark Mode ── */
html[data-theme="dark"] .scene-card {
  background: #1c1c26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
html[data-theme="dark"] .scene-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
}
html[data-theme="dark"] .scene-title {
  color: #e0e0e4;
}
html[data-theme="dark"] .meta-row {
  color: #888;
}
html[data-theme="dark"] .site-link a:hover {
  color: inherit;
}
html[data-theme="dark"] .release-date {
  color: #999;
}

</style>
