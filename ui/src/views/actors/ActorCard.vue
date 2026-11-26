<template>
  <div class="actor-card" @click="showDetails(actor)" @mouseenter="hovering = true" @mouseleave="hovering = false">
    <div class="thumbnail-wrapper" :style="{ aspectRatio: actorCardAspectRatio }">
      <div class="thumbnail-img" :class="{ 'is-contain': actorCardScale === 'contain' }"
           :style="{ opacity: isAvailable(actor) ? 1.0 : isAvailOpacity, '--actor-img': `url(${imageURL})` }">
        <img decoding="async" v-queued-src="imageURL" class="cover-image" :class="{ 'is-loaded': imageLoaded }"
             :style="{ objectFit: actorCardScale }" alt="" @load="imageLoaded = true" @error="imageLoaded = true"/>
      </div>

      <div class="top-right-tags">
        <b-tooltip :label="$t('Your rating')" :delay="500" v-if="actor.star_rating != 0">
          <b-tag class="mini-tag star-tag">
            <b-icon pack="mdi" icon="star" size="is-small"/>{{actor.star_rating}}
          </b-tag>
        </b-tooltip>
        <b-tooltip :label="$t('Average rating of scenes')" :delay="500" v-if="actor.scene_rating_average != 0">
          <b-tag class="mini-tag">
            <b-icon pack="mdi" icon="star-half-full" size="is-small"/>{{Math.round(actor.scene_rating_average * 4) / 4}}
          </b-tag>
        </b-tooltip>
      </div>

      <div v-if="hovering" class="hover-actions" :class="{ 'is-visible': hovering, 'is-glass': $store.state.optionsWeb.web.cardButtonStyle === 'glass' }" @click.stop>
        <a v-if="colleague != undefined" class="button is-info is-outlined is-small"
           @click="showColleague(actor.name, colleague)" :title="'Show Scenes with ' + actor.name">
          <b-icon pack="mdi" icon="movie-outline" size="is-small"/>
        </a>
        <actor-favourite-button :actor="actor" v-if="$store.state.optionsWeb.web.sceneFavourite"/>
        <actor-watchlist-button :actor="actor" v-if="$store.state.optionsWeb.web.sceneWatchlist"/>
        <actor-edit-button :actor="actor"/>
        <link-stashdb-button :item="actor" objectType="actor"/>
      </div>
    </div>

    <div class="info-section">
      <div class="title-row">
        <div class="actor-name">{{actor.name}}</div>
        <span class="status-icons">
          <b-icon v-if="actor.watchlist" pack="mdi" icon="calendar-check" size="is-small" class="status-primary" :title="$t('Watchlist')"/>
          <b-icon v-if="actor.favourite" pack="mdi" icon="heart" size="is-small" class="status-danger" :title="$t('Favourite')"/>
        </span>
      </div>
      <div class="meta-row">
        <span class="meta-left">
          <vue-load-image v-if="actor.nationality" class="flag-wrap">
            <template #image><img loading="lazy" decoding="async" :src="getImageURL('https://flagcdn.com/' + actor.nationality.toLowerCase() + '.svg')" class="flag" :title="actor.nationality"/></template>
          </vue-load-image>
          <span>{{sceneCount}} {{sceneCount == 1 ? 'scene' : 'scenes'}}<template v-if="actor.avail_count > 0"> · {{actor.avail_count}} available</template></span>
        </span>
        <span v-if="age !== null" class="actor-age" :title="actor.birth_date.slice(0, 10)">{{ age }} yrs</span>
      </div>
    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import ActorFavouriteButton from '../../components/ActorFavouriteButton'
import ActorWatchlistButton from '../../components/ActorWatchlistButton'
import ActorEditButton from '../../components/ActorEditButton'
import LinkStashdbButton from '../../components/LinkStashdbButton'
import VueLoadImage from 'vue-load-image'

export default defineComponent({
  name: 'ActorCard',
  props: { actor: Object, colleague: String },
  components: {ActorFavouriteButton, ActorWatchlistButton, VueLoadImage, ActorEditButton, LinkStashdbButton},

  data () {
    return {
      hovering: false,
      imageLoaded: false,
    }
  },

  watch: {
    imageURL () {
      this.imageLoaded = false
    },
  },

  computed: {
    sceneCount () {
      return this.actor.slim ? this.actor.scene_count : (this.actor.scenes || []).length
    },
    imageURL () {
      return this.getImageURL(this.actor.image_url)
    },
    age () {
      const b = this.actor.birth_date
      if (!b || b.startsWith('0001')) return null
      // birth dates are stored as UTC midnight; compare calendar dates, not local times
      const [y, m, day] = b.slice(0, 10).split('-').map(Number)
      const now = new Date()
      let a = now.getFullYear() - y
      if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < day)) a--
      return a > 0 && a < 120 ? a : null
    },
    isAvailOpacity () {      
      if (this.$store.state.optionsWeb.web.isAvailOpacity == undefined) {
        return .4
      }
      return this.$store.state.optionsWeb.web.isAvailOpacity / 100
    },
    actorCardAspectRatio () {
      if (this.$store.state.optionsWeb.web.actorCardAspectRatio == "2:3") {
        return 2 / 3
      } else if (this.$store.state.optionsWeb.web.actorCardAspectRatio == "9:16") {
        return 9 / 16
      } else {
        return 1
      }
    },
    actorCardScale () {
      if (this.$store.state.optionsWeb.web.actorCardScaleToFit) {
        return "contain"
      } else {
        return "cover"
      }
    },
  },

  methods: {
    getImageURL (u) {
      if (u=='' || u == undefined) {
        return "/ui/images/actor_placeholder.svg"
      }
      if (u.startsWith('http')) {
        return '/img/700x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    showDetails (actor) {
      this.$store.commit('overlay/showActorDetails', { actor: actor })
    },
    isAvailable(actor) {
      if (actor.slim) return actor.available_scenes > 0
      return (actor.scenes || []).some(scene => scene.is_available)
    },
    showColleague (main_actor, colleague) {      
      this.$store.state.sceneList.filters.cast = ["&" + main_actor , "&"+ colleague]
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$store.state.actorList.filters.dlState = "Any"
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.$store.commit('overlay/hideActorDetails')
    },

  },
});
</script>

<style scoped>
/* Same card language as SceneCard: rounded surface, image fills the frame,
   chips on the photo, actions on hover, fixed two-line info block. */
.actor-card {
  position: relative;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  transition: box-shadow 0.2s ease;
  cursor: pointer;
  isolation: isolate;
}
.actor-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 5;
}

.thumbnail-wrapper {
  position: relative;
  width: 100%;
  border-radius: 8px 8px 0 0;
}
.thumbnail-img {
  position: absolute;
  inset: 0;
  overflow: hidden;
  border-radius: inherit;
  background: var(--card-placeholder);
}
/* scale-to-fit: fill the letterbox with a soft blurred copy instead of a flat colour */
.thumbnail-img.is-contain::before {
  content: '';
  position: absolute;
  inset: -20px;
  background: var(--actor-img) center / cover no-repeat;
  filter: blur(18px) brightness(0.7);
}
.cover-image {
  position: relative;
  display: block;
  width: 100%;
  height: 100%;
  object-position: center top;
  filter: blur(15px);
  transition: filter 0.5s ease;
}
.thumbnail-img.is-contain .cover-image {
  object-position: center;
}
.cover-image.is-loaded {
  filter: blur(0);
}

.top-right-tags {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 3px;
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
  border-radius: 4px !important;
  font-weight: 500 !important;
  gap: 3px;
}
.mini-tag.star-tag {
  background: rgba(255, 221, 87, 0.95) !important;
  color: rgba(0, 0, 0, 0.7) !important;
}
.mini-tag :deep(.icon) {
  margin: 0 2px 0 0 !important;
  width: 12px !important;
  height: 12px !important;
}
.mini-tag :deep(.mdi) {
  font-size: 12px !important;
}

.hover-actions {
  position: absolute;
  bottom: 6px;
  left: 6px;
  right: 6px;
  display: flex;
  flex-wrap: wrap-reverse;
  justify-content: flex-end;
  gap: 4px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease;
  z-index: 25;
}
.hover-actions.is-visible {
  opacity: 1;
  pointer-events: auto;
}
.hover-actions > *,
.hover-actions :deep(.button) {
  margin: 0 !important;
}
.hover-actions :deep(.button) {
  width: 28px !important;
  height: 28px !important;
  padding: 0 !important;
  border: none !important;
  border-radius: 5px !important;
  background: #fff !important;
  color: #363636 !important;
}
.hover-actions:not(.is-glass) :deep(.button.is-outlined) { opacity: 0.7; }
.hover-actions:not(.is-glass) :deep(.button.is-outlined:hover) { opacity: 1; }
.hover-actions:not(.is-glass) :deep(.button.is-primary),
.hover-actions:not(.is-glass) :deep(.button.is-primary.is-outlined) { background: #7957d5 !important; color: #fff !important; }
.hover-actions:not(.is-glass) :deep(.button.is-danger),
.hover-actions:not(.is-glass) :deep(.button.is-danger.is-outlined) { background: #f14668 !important; color: #fff !important; }
.hover-actions:not(.is-glass) :deep(.button.is-info),
.hover-actions:not(.is-glass) :deep(.button.is-info.is-outlined) { background: #3e8ed0 !important; color: #fff !important; }
.hover-actions:not(.is-glass) :deep(.button.is-dark),
.hover-actions:not(.is-glass) :deep(.button.is-dark.is-outlined) { background: #363636 !important; color: #fff !important; }
.hover-actions :deep(.button .icon) { margin: 0 !important; }
.hover-actions :deep(.button .mdi) { font-size: 16px !important; }
.hover-actions :deep(.button span:not(.icon)) { display: none !important; }

.info-section {
  padding: 7px 10px;
  height: 52px;
  box-sizing: border-box;
}
.title-row {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 20px;
  min-width: 0;
}
.actor-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.3;
  color: var(--text-primary, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.status-icons {
  display: inline-flex;
  gap: 3px;
  flex-shrink: 0;
}
.status-icons :deep(.mdi) { font-size: 14px !important; }
.status-primary { color: #7957d5; }
.status-danger { color: #f14668; }
.meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 18px;
  font-size: 11px;
  color: var(--text-secondary, #666);
  min-width: 0;
}
.meta-left {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.flag-wrap {
  display: inline-flex;
  flex-shrink: 0;
}
.flag {
  display: block;
  height: 10px;
  border-radius: 2px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.15);
}
.actor-age {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  opacity: 0.75;
}

html[data-theme="dark"] .actor-card {
  background: #1c1c26;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
html[data-theme="dark"] .actor-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
}

html[data-theme="dark"] .actor-name {
  color: #e0e0e4;
}
html[data-theme="dark"] .meta-row {
  color: #888;
}
</style>
