<template>
  <div class="scene-card" 
       @mouseenter="hovering = true; startPreview()" 
       @mouseleave="hovering = false; stopPreview()"
       @click="showDetails(item)">
    
    <!-- Video Preview Overlay - covers entire card -->
    <transition name="fade">
      <div class="preview-overlay" v-if="preview && item.has_preview" @click="showDetails(item)">
        <video ref="previewVideo" :src="`/api/dms/preview/${item.scene_id}`" autoplay loop muted></video>
      </div>
    </transition>

    <!-- Main Thumbnail (aspect ratio from settings) -->
    <div class="thumbnail-wrapper" :style="{ aspectRatio: sceneCardAspectRatioValue }">
      <div class="thumbnail-img" :style='{ opacity: item.is_available ? 1.0 : isAvailOpactiy }'>
        <img
          ref="coverImage"
          :src="coverImageSrc"
          @error="onImageError"
          @load="onImageLoad"
          class="cover-image"
          :class="{ 'is-loaded': imageLoaded }"
          :style="{ objectFit: sceneCardScaleToFit ? 'contain' : 'cover' }"
        />
      </div>
      
      <!-- Tags Overlay - Top Left -->
      <div class="tags-overlay">
        <b-tag type="is-info" v-if="videoFilesCount > 1 && !item.is_multipart" class="mini-tag">
          <b-icon pack="mdi" icon="file" size="is-small"/>
          <span>{{videoFilesCount}}</span>
        </b-tag>
        <b-tag v-if="item.is_scripted" class="mini-tag funscript-tag">
          <b-icon pack="mdi" icon="pulse" size="is-small"/>
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
        <b-tag type="is-info" v-if="item.cuepoints != null && item.cuepoints.length > 0 && this.$store.state.optionsWeb.web.sceneCuepoint" class="mini-tag">
          <b-icon pack="mdi" icon="skip-next-outline" size="is-small"/>
          <span v-if="item.cuepoints.length > 1">{{item.cuepoints.length}}</span>
        </b-tag>
      </div>

      <!-- Top Right: Duration & Rating -->
      <div class="top-right-tags">
        <b-tag type="is-warning" v-if="item.star_rating > 0" class="mini-tag star-tag">
          <b-icon pack="mdi" icon="star" size="is-small"/>
          {{item.star_rating}}
        </b-tag>
        <b-tag v-if="item.duration > 0 && this.$store.state.optionsWeb.web.sceneDuration" class="mini-tag duration-tag">
          {{item.duration}}m
        </b-tag>
      </div>

      <!-- Heatmap Strip(s) -->
      <div v-if="this.$store.state.optionsWeb.web.showScriptHeatmap && (files = getFunscripts(this.$store.state.optionsWeb.web.showAllHeatmaps))" class="heatmap-strip">
        <div v-if="files.length" class="heatmap-stack" :class="{'single-heatmap': files.length === 1, 'multi-heatmap': files.length > 1}">
          <img v-for="file in files" :key="file.id" :src="getHeatmapURL(file.id)"/>
        </div>
      </div>


      <!-- Hover Actions - anchored inside thumbnail -->
      <div class="hover-actions" :class="{ 'is-visible': hovering, 'is-preview': preview && item.has_preview }" @click.stop>
        <hidden-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneHidden"/>
        <watchlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
        <trailerlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneTrailerlist"/>
        <favourite-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
        <wishlist-button v-if="this.$store.state.optionsWeb.web.sceneWishlist && !item.is_available" :item="item"/>
        <watched-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatched"/>
        <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit"/>
        <link-stashdb-button :item="item" v-if="!this.stashLinkExists" objectType="scene"/>
        <b-tooltip v-for="(altsrc, idx) in alternateSources" :key="idx" type="is-light" :label="altsrc.title" :delay="100">
          <a :href="altsrc.url" target="_blank" class="alt-link" @click.stop>
            <vue-load-image>
              <template #image><img :src="getImageURL(altsrc.site_icon)" class="alt-img"/></template>
              <template #error><b-icon pack="mdi" icon="link" size="is-small"/></template>
            </vue-load-image>
          </a>
        </b-tooltip>
      </div>
    </div>

    <!-- Info Section -->
    <div class="info-section" :class="{ 'is-preview': preview && item.has_preview }">
      <div class="title-row">
        <div class="scene-title">{{item.title}}</div>
        <span v-if="hasActiveStatus" class="status-icons">
          <span v-if="item.is_hidden" class="status-icon status-danger" data-tooltip="Hidden">
            <b-icon pack="mdi" icon="eye-off" size="is-small"/>
          </span>
          <span v-if="item.watchlist" class="status-icon status-primary" data-tooltip="Watchlist">
            <b-icon pack="mdi" icon="calendar-check" size="is-small"/>
          </span>
          <span v-if="item.trailerlist && !item.is_available" class="status-icon status-primary" data-tooltip="Trailer List">
            <b-icon pack="mdi" icon="movie-open-check" size="is-small"/>
          </span>
          <span v-if="item.favourite" class="status-icon status-danger" data-tooltip="Favourite">
            <b-icon pack="mdi" icon="heart" size="is-small"/>
          </span>
          <span v-if="item.wishlist" class="status-icon status-info" data-tooltip="Wishlist">
            <b-icon pack="mdi" icon="oil-lamp" size="is-small"/>
          </span>
          <span v-if="item.is_watched" class="status-icon status-dark" data-tooltip="Watched">
            <b-icon pack="mdi" icon="eye-check" size="is-small"/>
          </span>
        </span>
      </div>
      <div class="meta-row">
        <span class="site-link">
          <img v-if="siteIconUrl && showSiteLogo" :src="siteIconUrl" class="site-icon" alt=""/>
          <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer" @click.stop>
            <b-icon pack="mdi" icon="link-lock" custom-size="mdi-14px"/>
          </a>
          <a :href="item.scene_url" :class="{'site-subscribed': item.is_subscribed}" target="_blank" rel="noreferrer" @click.stop>{{item.site}}</a>
          <a v-if="stashLinkExists" :href="getStashdbUrl()" target="_blank" class="stashdb-link" @click.stop>
            <img src="https://guidelines.stashdb.org/favicon.ico" class="stashdb-icon" alt="StashDB"/>
          </a>
        </span>
        <span v-if="item.release_date !== '0001-01-01T00:00:00Z'" class="release-date">
          {{format(parseISO(item.release_date), dateFormat)}}
        </span>
      </div>
    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import { format, parseISO } from 'date-fns'
import WatchlistButton from '../../components/WatchlistButton'
import FavouriteButton from '../../components/FavouriteButton'
import WishlistButton from '../../components/WishlistButton'
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'
import LinkStashdbButton from '../../components/LinkStashdbButton'
import TrailerlistButton from '../../components/TrailerlistButton'
import HiddenButton from '../../components/HiddenButton'
import ky from 'ky'
import VueLoadImage from 'vue-load-image'

export default defineComponent({
  name: 'SceneCard',
  props: { item: Object, reRead: Boolean },
  components: { WatchlistButton, FavouriteButton, WishlistButton, WatchedButton, EditButton, LinkStashdbButton, TrailerlistButton, HiddenButton, VueLoadImage },

  data () {
    return {
      preview: false,
      hovering: false,
      format,
      parseISO,
      alternateSources: [],
      stashLinkExists: false,
      imageRetryCount: 0,
      imageLoaded: false,
      imageKey: 0,
    }
  },

  mounted () {
    this.loadAlternateSources()
  },

  computed: {
    videoFilesCount () {
      if (this.item.file == null) { return 0 }
      return this.item.file.filter(obj => obj.type === 'video').length
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
    siteIconUrl () {
      const sites = this.$store.state.optionsSites.items
      if (!sites || !sites.length) return null
      const site = sites.find(s => s.id === this.item.scraper_id)
      if (!site || !site.avatar_url) return null
      return this.getImageURL(site.avatar_url)
    },
    hasActiveStatus () {
      return this.item.is_hidden || this.item.favourite || this.item.is_watched ||
             this.item.watchlist || this.item.wishlist || (this.item.trailerlist && !this.item.is_available)
    },
    coverImageSrc () {
      // Add imageKey to force re-fetch on retry
      const u = this.item.cover_url
      if (!u || !u.startsWith('http')) return u
      let url
      if (u.search("%") == -1) {
        url = '/img/1200x/' + encodeURI(u)
      } else {
        url = '/img/1200x/' + encodeURI(decodeURI(u))
      }
      // Append cache-buster on retry to force new request
      return this.imageKey > 0 ? url + '?retry=' + this.imageKey : url
    }
  },

  methods: {
    onImageError () {
      if (this.imageRetryCount < 2) {
        this.imageRetryCount++
        this.imageLoaded = false
        setTimeout(() => {
          this.imageKey++
        }, this.imageRetryCount * 1000)
      } else {
        this.imageLoaded = true
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
      if (this.$refs.coverImage && this.$refs.coverImage.naturalWidth > 100) {
        this.imageLoaded = true
        this.imageRetryCount = 0
      } else if (this.imageRetryCount < 2) {
        this.imageRetryCount++
        this.imageLoaded = false
        setTimeout(() => {
          this.imageKey++
        }, this.imageRetryCount * 1000)
      } else {
        this.imageLoaded = true
      }
    },
    async loadAlternateSources () {
      this.stashLinkExists = false
      try {
        const response = await ky.get('/api/scene/alternate_source/' + this.item.id).json()
        if (response == null) return

        this.alternateSources = response
          .filter(altsrc => altsrc.external_source.startsWith("alternate scene ") || altsrc.external_source == "stashdb scene")
          .map(altsrc => {
            const extdata = JSON.parse(altsrc.external_data)
            let title
            if (altsrc.external_source.startsWith("alternate scene ")) {
              title = extdata.scene?.title || 'No Title'
            } else if (altsrc.external_source == "stashdb scene") {
              title = extdata.title || 'No Title'
            }
            if (altsrc.external_source.includes('stashdb')) {
              this.stashLinkExists = true
            }
            return { ...altsrc, title }
          })
      } catch (error) {
        // Silent error handling
      }
    },
    startPreview () {
      if (this.item.has_preview) {
        // No delay - just fade transition
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
    getImageURL (u) {
      if (!u.startsWith('http')) return u
      if (u.search("%") == -1) {
        return '/img/1200x/' + encodeURI(u)
      } else {
        return '/img/1200x/' + encodeURI(decodeURI(u))
      }
    },
    showDetails (scene) {
      // reRead is required when the SceneCard is clicked from the ActorDetails
      // the Scenes associated Tables such as Tags, Cast arwon't be Preloaded and
      // will cause errors when the Details Overlay loads
      if (this.reRead) {
        ky.get('/api/scene/'+scene.id).json().then(data => {
          if (data.id != 0){
            this.$store.commit('overlay/showDetails', { scene: data })
          }
        })
      } else {
        this.$store.commit('overlay/showDetails', { scene: scene })
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
          let script;
          if (script = this.item.file.find((a) => a.type === 'script' && a.has_heatmap && a.is_selected_script)) {
            return [script]
          }
          if (script = this.item.file.find((a) => a.type === 'script' && a.has_heatmap)) {
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
/* Card Container - Light UI */
.scene-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  transition: box-shadow 0.2s ease;
  cursor: pointer;
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
  border-radius: 8px;
}

.preview-overlay video {
  width: 100%;
  height: 100%;
  object-fit: cover;
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
.thumbnail-wrapper {
  position: relative;
  width: 100%;
  overflow: hidden;
  border-radius: 8px 8px 0 0;
  flex-shrink: 0;
}

.thumbnail-img {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  transition: transform 0.3s ease;
  overflow: hidden;
}

.thumbnail-img .cover-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center center;
  display: block;
  transition: filter 0.5s ease;
  filter: blur(15px);
}

.thumbnail-img .cover-image.is-loaded {
  filter: blur(0);
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
.hover-actions :deep(.button.is-primary),
.hover-actions :deep(.button.is-primary.is-outlined) {
  background: #7957d5 !important; color: #fff !important;
}
.hover-actions :deep(.button.is-primary.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-primary.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button.is-danger),
.hover-actions :deep(.button.is-danger.is-outlined) {
  background: #f14668 !important; color: #fff !important;
}
.hover-actions :deep(.button.is-danger.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-danger.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button.is-info),
.hover-actions :deep(.button.is-info.is-outlined) {
  background: #3e8ed0 !important; color: #fff !important;
}
.hover-actions :deep(.button.is-info.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-info.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button.is-link),
.hover-actions :deep(.button.is-link.is-outlined) {
  background: #485fc7 !important; color: #fff !important;
}
.hover-actions :deep(.button.is-link.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-link.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button.is-warning),
.hover-actions :deep(.button.is-warning.is-outlined) {
  background: #ffdd57 !important; color: rgba(0,0,0,0.7) !important;
}
.hover-actions :deep(.button.is-warning.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-warning.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button.is-success),
.hover-actions :deep(.button.is-success.is-outlined) {
  background: #48c78e !important; color: #fff !important;
}
.hover-actions :deep(.button.is-success.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-success.is-outlined:hover) { opacity: 1; }

.hover-actions :deep(.button.is-dark),
.hover-actions :deep(.button.is-dark.is-outlined) {
  background: #363636 !important; color: #fff !important;
}
.hover-actions :deep(.button.is-dark.is-outlined) { opacity: 0.7; }
.hover-actions :deep(.button.is-dark.is-outlined:hover) { opacity: 1; }

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
  background: #e8e8e8;
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
}
.info-section:has(.status-icon:hover) {
  z-index: 50;
}

/* Preview mode: frosted glass over video */
.info-section.is-preview {
  background: rgba(0,0,0,0.55);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
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

.site-icon {
  width: 16px;
  height: 16px;
  max-width: 16px;
  max-height: 16px;
  border-radius: 2px;
  object-fit: cover;
  flex-shrink: 0;
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
