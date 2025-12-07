<template>
  <div class="scene-card" 
       @mouseenter="hovering = true; startPreview()" 
       @mouseleave="hovering = false; stopPreview()"
       @click="showDetails(item)">
    
    <!-- Full Card Video Preview Overlay - Covers Everything -->
    <transition name="fade">
      <div class="preview-overlay" v-if="preview && item.has_preview" @click="showDetails(item)">
        <video ref="previewVideo" :src="`/api/dms/preview/${item.scene_id}`" autoplay loop muted></video>
        <!-- Hover Actions on top of preview -->
        <div class="hover-actions is-visible">
          <div class="actions-row" @click.stop>
            <hidden-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneHidden"/>
            <watchlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
            <trailerlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneTrailerlist"/>
            <favourite-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
            <wishlist-button v-if="this.$store.state.optionsWeb.web.sceneWishlist && !item.is_available" :item="item"/>
            <watched-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatched"/>
            <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit"/>
            <link-stashdb-button :item="item" v-if="!this.stashLinkExists" objectType="scene"/>
            <!-- Alt Sources inline -->
            <template v-for="(altsrc, idx) in alternateSources">
              <b-tooltip :key="idx" type="is-light" :label="altsrc.title" :delay="100">
                <a :href="altsrc.url" target="_blank" class="alt-link" @click.stop>
                  <vue-load-image>
                    <img slot="image" :src="getImageURL(altsrc.site_icon)" class="alt-img"/>
                    <b-icon slot="error" pack="mdi" icon="link" size="is-small"/>
                  </vue-load-image>
                </a>
              </b-tooltip>
            </template>
          </div>
        </div>
      </div>
    </transition>

    <!-- Main 16:9 Thumbnail -->
    <div class="thumbnail-wrapper">
      <div class="thumbnail-img"
           :style='{backgroundImage: `url("${getImageURL(item.cover_url)}")`, opacity: item.is_available ? 1.0 : isAvailOpactiy}'>
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
      
      <!-- Hover Actions Overlay - Only show if NO preview available -->
      <div class="hover-actions" :class="{ 'is-visible': hovering && !item.has_preview }">
        <div class="actions-row" @click.stop>
          <hidden-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneHidden"/>
          <watchlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatchlist"/>
          <trailerlist-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneTrailerlist"/>
          <favourite-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneFavourite"/>
          <wishlist-button v-if="this.$store.state.optionsWeb.web.sceneWishlist && !item.is_available" :item="item"/>
          <watched-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneWatched"/>
          <edit-button :item="item" v-if="this.$store.state.optionsWeb.web.sceneEdit"/>
          <link-stashdb-button :item="item" v-if="!this.stashLinkExists" objectType="scene"/>
          <!-- Alt Sources inline -->
          <template v-for="(altsrc, idx) in alternateSources">
            <b-tooltip :key="idx" type="is-light" :label="altsrc.title" :delay="100">
              <a :href="altsrc.url" target="_blank" class="alt-link" @click.stop>
                <vue-load-image>
                  <img slot="image" :src="getImageURL(altsrc.site_icon)" class="alt-img"/>
                  <b-icon slot="error" pack="mdi" icon="link" size="is-small"/>
                </vue-load-image>
              </a>
            </b-tooltip>
          </template>
        </div>
      </div>
    </div>

    <!-- Info Section with Status Icons -->
    <div class="info-section">
      <div class="title-row">
        <div class="scene-title">{{item.title}}</div>
      </div>
      <div class="meta-row">
        <span class="site-link">
          <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer" @click.stop>
            <b-icon pack="mdi" icon="link-lock" custom-size="mdi-14px"/>
          </a>
          <a :href="item.scene_url" :class="{'site-subscribed': item.is_subscribed}" target="_blank" rel="noreferrer" @click.stop>{{item.site}}</a>
          <!-- Stashdb link in info section -->
          <a v-if="stashLinkExists" :href="getStashdbUrl()" target="_blank" class="stashdb-link" @click.stop>
            <img src="https://guidelines.stashdb.org/favicon.ico" class="stashdb-icon" alt="StashDB"/>
          </a>
        </span>
        <span class="status-icons" @click.stop>
          <!-- Active state icons - matching button icons -->
          <b-icon v-if="item.is_hidden" pack="mdi" icon="eye-off" custom-size="mdi-14px" class="status-icon is-hidden" title="Hidden"/>
          <b-icon v-if="item.favourite" pack="mdi" icon="heart" custom-size="mdi-14px" class="status-icon is-favourite" title="Favourite"/>
          <b-icon v-if="item.is_watched" pack="mdi" icon="eye-check" custom-size="mdi-14px" class="status-icon is-watched" title="Watched"/>
          <b-icon v-if="item.watchlist" pack="mdi" icon="calendar-check" custom-size="mdi-14px" class="status-icon is-watchlist" title="Watchlist"/>
          <b-icon v-if="item.wishlist" pack="mdi" icon="oil-lamp" custom-size="mdi-14px" class="status-icon is-wishlist" title="Wishlist"/>
          <b-icon v-if="item.trailerlist" pack="mdi" icon="movie-search-outline" custom-size="mdi-14px" class="status-icon is-trailerlist" title="Trailerlist"/>
          <!-- Separator only if there are status icons -->
          <span v-if="hasStatusIcons" class="date-separator"></span>
          <span v-if="item.release_date !== '0001-01-01T00:00:00Z'" class="release-date">
            {{format(parseISO(item.release_date), "yyyy-MM-dd")}}
          </span>
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
    hasStatusIcons () {
      return this.item.is_hidden || this.item.favourite || this.item.is_watched || 
             this.item.watchlist || this.item.wishlist || this.item.trailerlist
    }
  },
  methods: {
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
        return '/img/700x/' + encodeURI(u)
      } else {
        return '/img/700x/' + encodeURI(decodeURI(u))
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
}

.scene-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* Full Card Preview Overlay - Covers EVERYTHING */
.preview-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 20;
  background: #000;
  border-radius: 8px;
}

.preview-overlay video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 8px;
}

/* Fade transition */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.25s ease;
}
.fade-enter, .fade-leave-to {
  opacity: 0;
}

/* 16:9 Thumbnail - Always */
.thumbnail-wrapper {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: 8px 8px 0 0;
}

.thumbnail-img {
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  transition: transform 0.3s ease;
  border-radius: 8px 8px 0 0;
}

.scene-card:hover .thumbnail-img {
  transform: scale(1.02);
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

/* Hover Actions Overlay */
.hover-actions {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 8px;
  background: linear-gradient(transparent, rgba(0,0,0,0.75));
  display: flex;
  justify-content: flex-end;
  align-items: center;
  opacity: 0;
  transition: opacity 0.2s ease;
  z-index: 15;
  pointer-events: none;
}

.hover-actions.is-visible {
  opacity: 1;
  pointer-events: auto;
}

/* Also show on preview overlay */
.preview-overlay .hover-actions {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  opacity: 1;
  pointer-events: auto;
  border-radius: 0 0 8px 8px;
}

.actions-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

/* Square icon-only buttons with better hover */
.actions-row :deep(.button) {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  min-height: 28px !important;
  padding: 0 !important;
  margin: 0 !important;
  border-radius: 5px !important;
  background: rgba(255,255,255,0.92) !important;
  border: none !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  transition: all 0.15s ease !important;
  color: #363636 !important;
  line-height: 1 !important;
}

.actions-row :deep(.button:hover) {
  background: #fff !important;
  transform: scale(1.05);
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

/* Primary - Purple */
.actions-row :deep(.button.is-primary) {
  background: rgba(121, 87, 213, 0.95) !important;
  color: #fff !important;
}
.actions-row :deep(.button.is-primary:hover) {
  background: #7957d5 !important;
}

/* Success - Green */
.actions-row :deep(.button.is-success) {
  background: rgba(72, 199, 142, 0.95) !important;
  color: #fff !important;
}
.actions-row :deep(.button.is-success:hover) {
  background: #48c78e !important;
}

/* Danger - Red */
.actions-row :deep(.button.is-danger) {
  background: rgba(241, 70, 104, 0.95) !important;
  color: #fff !important;
}
.actions-row :deep(.button.is-danger:hover) {
  background: #f14668 !important;
}

/* Warning - Yellow */
.actions-row :deep(.button.is-warning) {
  background: rgba(255, 221, 87, 0.95) !important;
  color: rgba(0,0,0,0.7) !important;
}
.actions-row :deep(.button.is-warning:hover) {
  background: #ffdd57 !important;
}

/* Info - Blue */
.actions-row :deep(.button.is-info) {
  background: rgba(62, 142, 208, 0.95) !important;
  color: #fff !important;
}
.actions-row :deep(.button.is-info:hover) {
  background: #3e8ed0 !important;
}

/* Link - Blue text */
.actions-row :deep(.button.is-link) {
  background: rgba(72, 95, 199, 0.95) !important;
  color: #fff !important;
}
.actions-row :deep(.button.is-link:hover) {
  background: #485fc7 !important;
}

/* Light */
.actions-row :deep(.button.is-light) {
  background: rgba(245, 245, 245, 0.95) !important;
  color: #363636 !important;
}
.actions-row :deep(.button.is-light:hover) {
  background: #f5f5f5 !important;
}

/* Dark */
.actions-row :deep(.button.is-dark) {
  background: rgba(54, 54, 54, 0.95) !important;
  color: #fff !important;
}
.actions-row :deep(.button.is-dark:hover) {
  background: #363636 !important;
}

.actions-row :deep(.button .icon) {
  margin: 0 !important;
  width: 16px !important;
  height: 16px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

.actions-row :deep(.button .icon .mdi) {
  font-size: 16px !important;
  line-height: 1 !important;
}

.actions-row :deep(.button span:not(.icon)) {
  display: none !important;
}

/* Alt Source Images - Inline with buttons, vertically centered */
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
}

.alt-link:hover {
  transform: scale(1.08);
  background: #fff;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
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
  padding: 8px 10px;
  min-height: 56px;
}

.title-row {
  margin-bottom: 4px;
}

.scene-title {
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary, #333);
  line-height: 1.3;
}

.meta-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 10px;
  color: var(--text-secondary, #666);
  gap: 4px;
  min-width: 0;
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
  transition: color 0.15s;
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

/* Status icons row */
.status-icons {
  display: inline-flex;
  align-items: center;
  flex-shrink: 1;
  gap: 4px;
  overflow: hidden;
  min-width: 0;
}

.status-icons :deep(.status-icon) {
  width: 12px !important;
  height: 12px !important;
  min-width: 12px !important;
  min-height: 12px !important;
  max-width: 12px !important;
  max-height: 12px !important;
  font-size: 12px !important;
  flex-shrink: 0;
}

.status-icons :deep(.status-icon .mdi),
.status-icons :deep(.status-icon .mdi::before) {
  font-size: 12px !important;
  line-height: 1 !important;
  width: 12px !important;
  height: 12px !important;
}

/* Colors matching button types */
.status-icons :deep(.is-hidden) {
  color: #f14668 !important; /* red - danger color */
}

.status-icons :deep(.is-favourite) {
  color: #f14668 !important;
}

.status-icons :deep(.is-watched) {
  color: #363636 !important;
}

.status-icons :deep(.is-watchlist) {
  color: #7957d5 !important;
}

.status-icons :deep(.is-wishlist) {
  color: #3e8ed0 !important;
}

.status-icons :deep(.is-trailerlist) {
  color: #7957d5 !important;
}

.date-separator {
  width: 1px;
  height: 10px;
  background: rgba(0,0,0,0.15);
}

.release-date {
  opacity: 0.6;
  font-size: 10px;
  white-space: nowrap;
}
</style>
