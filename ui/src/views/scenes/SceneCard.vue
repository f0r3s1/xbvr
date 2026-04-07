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


    </div>

    <!-- Hover Actions -->
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
            <img slot="image" :src="getImageURL(altsrc.site_icon)" class="alt-img"/>
            <b-icon slot="error" pack="mdi" icon="link" size="is-small"/>
          </vue-load-image>
        </a>
      </b-tooltip>
    </div>

    <!-- Info Section -->
    <div class="info-section" :class="{ 'is-preview': preview && item.has_preview }">
      <div class="scene-title">{{item.title}}</div>
      <div class="meta-row">
        <span class="site-link">
          <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer" @click.stop>
            <b-icon pack="mdi" icon="link-lock" custom-size="mdi-14px"/>
          </a>
          <a :href="item.scene_url" :class="{'site-subscribed': item.is_subscribed}" target="_blank" rel="noreferrer" @click.stop>{{item.site}}</a>
          <a v-if="stashLinkExists" :href="getStashdbUrl()" target="_blank" class="stashdb-link" @click.stop>
            <img src="https://guidelines.stashdb.org/favicon.ico" class="stashdb-icon" alt="StashDB"/>
          </a>
        </span>
        <span v-if="item.release_date !== '0001-01-01T00:00:00Z'" class="release-date">
          {{format(parseISO(item.release_date), "yyyy-MM-dd")}}
        </span>
      </div>
    </div>
  </div>
</template>

<script>
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

export default {
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
    hasStatusIcons () {
      return this.item.is_hidden || this.item.favourite || this.item.is_watched || 
             this.item.watchlist || this.item.wishlist || this.item.trailerlist
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
  }
}
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
  overflow: hidden;
}

.scene-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
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
  border-radius: 0;
  flex-shrink: 0;
}

.thumbnail-img {
  width: 100%;
  height: 100%;
  position: relative;
  background-color: transparent;
  transition: transform 0.3s ease;
  overflow: hidden;
}

.thumbnail-img .cover-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: filter 0.5s ease, transform 0.5s ease;
  filter: blur(15px);
  transform: scale(1.05);
}

.thumbnail-img .cover-image.is-loaded {
  filter: blur(0);
  transform: scale(1);
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

/* Hover Actions - positioned above info section */
.hover-actions {
  position: absolute;
  bottom: 56px;
  right: 6px;
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 4px;
  align-items: center;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease;
  z-index: 25;
}

.hover-actions.is-visible {
  opacity: 1;
  pointer-events: auto;
}



/* Kill ALL margin/padding on every element inside hover-actions */
.hover-actions :deep(*) {
  margin: 0 !important;
  margin-top: 0 !important;
  margin-right: 0 !important;
  margin-bottom: 0 !important;
  margin-left: 0 !important;
  padding: 0 !important;
}
.hover-actions > * {
  margin: 0 !important;
  margin-bottom: 0 !important;
}

/* Base button reset — fixed 28x28, no extra spacing */
.hover-actions :deep(.button) {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  min-height: 28px !important;
  max-width: 28px !important;
  max-height: 28px !important;
  border-radius: 5px !important;
  border: none !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  transition: all 0.15s ease !important;
  line-height: 1 !important;
  box-sizing: border-box !important;
  transform: none !important;
  box-shadow: none !important;
  /* Default: white bg */
  background: rgba(255,255,255,0.92) !important;
  color: #363636 !important;
}
.hover-actions :deep(.button:hover) {
  background: #fff !important;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15) !important;
}

/* Active buttons with their semantic colors */
.hover-actions :deep(.button.is-primary) {
  background: rgba(121, 87, 213, 0.95) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-primary:hover) {
  background: #7957d5 !important;
}
.hover-actions :deep(.button.is-success) {
  background: rgba(72, 199, 142, 0.95) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-success:hover) {
  background: #48c78e !important;
}
.hover-actions :deep(.button.is-danger) {
  background: rgba(241, 70, 104, 0.95) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-danger:hover) {
  background: #f14668 !important;
}
.hover-actions :deep(.button.is-warning) {
  background: rgba(255, 221, 87, 0.95) !important;
  color: rgba(0,0,0,0.7) !important;
}
.hover-actions :deep(.button.is-warning:hover) {
  background: #ffdd57 !important;
}
.hover-actions :deep(.button.is-info) {
  background: rgba(62, 142, 208, 0.95) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-info:hover) {
  background: #3e8ed0 !important;
}
.hover-actions :deep(.button.is-link) {
  background: rgba(72, 95, 199, 0.95) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-link:hover) {
  background: #485fc7 !important;
}
.hover-actions :deep(.button.is-dark) {
  background: rgba(54, 54, 54, 0.95) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-dark:hover) {
  background: #363636 !important;
}

/* Outlined (inactive) buttons — same color as active but more transparent */
/* Outlined (inactive) — same color, slightly transparent */
.hover-actions :deep(.button.is-primary.is-outlined) {
  background: rgba(121, 87, 213, 0.7) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-primary.is-outlined:hover) {
  background: #7957d5 !important;
}
.hover-actions :deep(.button.is-success.is-outlined) {
  background: rgba(72, 199, 142, 0.7) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-success.is-outlined:hover) {
  background: #48c78e !important;
}
.hover-actions :deep(.button.is-danger.is-outlined) {
  background: rgba(241, 70, 104, 0.7) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-danger.is-outlined:hover) {
  background: #f14668 !important;
}
.hover-actions :deep(.button.is-warning.is-outlined) {
  background: rgba(255, 221, 87, 0.7) !important;
  color: rgba(0,0,0,0.7) !important;
}
.hover-actions :deep(.button.is-warning.is-outlined:hover) {
  background: #ffdd57 !important;
}
.hover-actions :deep(.button.is-info.is-outlined) {
  background: rgba(62, 142, 208, 0.7) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-info.is-outlined:hover) {
  background: #3e8ed0 !important;
}
.hover-actions :deep(.button.is-link.is-outlined) {
  background: rgba(72, 95, 199, 0.7) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-link.is-outlined:hover) {
  background: #485fc7 !important;
}
.hover-actions :deep(.button.is-dark.is-outlined) {
  background: rgba(54, 54, 54, 0.7) !important;
  color: #fff !important;
}
.hover-actions :deep(.button.is-dark.is-outlined:hover) {
  background: #363636 !important;
}

/* Preview mode: frosted glass for ALL buttons */
.hover-actions.is-preview :deep(.button) {
  background: rgba(255,255,255,0.15) !important;
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  color: #fff !important;
  border: 1px solid rgba(255,255,255,0.2) !important;
}
.hover-actions.is-preview :deep(.button:hover) {
  background: rgba(255,255,255,0.3) !important;
  box-shadow: none !important;
}
.hover-actions.is-preview :deep(.button:not(.is-outlined)) {
  background: rgba(255,255,255,0.25) !important;
}

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
  background: rgba(255,255,255,0.95);
  border-radius: 5px;
  transition: all 0.15s ease;
  box-sizing: border-box;
}
.alt-link:hover {
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
}

/* Preview mode alt-links */
.hover-actions.is-preview .alt-link {
  background: rgba(255,255,255,0.15);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  border: 1px solid rgba(255,255,255,0.2);
}
.hover-actions.is-preview .alt-link:hover {
  background: rgba(255,255,255,0.3);
  box-shadow: none;
}

.alt-img {
  width: 18px;
  height: 18px;
  border-radius: 2px;
  object-fit: cover;
  display: block;
}

/* Info Section */
.info-section {
  position: relative;
  z-index: 21;
  padding: 7px 10px;
  transition: background 0.2s ease, color 0.2s ease;
  background: transparent;
}

/* Preview mode: frosted glass over video */
.info-section.is-preview {
  background: rgba(0,0,0,0.55);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}


.scene-title {
  font-size: 13px;
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
  color: #fff;
}


.meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-secondary, #666);
  min-width: 0;
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

.site-link a {
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.site-link a:hover {
  color: var(--primary, #7957d5);
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
  color: #8b7ff0;
}
html[data-theme="dark"] .release-date {
  color: #999;
}
</style>
