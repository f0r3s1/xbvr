<template>
  <div ref="scrollContainer" style="position: relative; min-height: 200px;">
    <!-- Reloads keep the current cards on screen (dimmed) under a thin progress
         bar; nothing blocks the grid. A first load shows skeleton cards. -->
    <div class="list-progress" v-if="isListReloading" role="progressbar" aria-label="Loading scenes"></div>

    <div class="list-toolbar">
      <strong class="toolbar-total">{{ isListReloading && !items.length ? $t('Loading…') : total + ' results' }}</strong>
      <div class="toolbar-segment">
        <b-radio-button v-model="dlState" native-value="any" size="is-small">
          {{$t("Any")}} ({{counts.any}})
        </b-radio-button>
        <b-radio-button v-model="dlState" native-value="available" size="is-small">
          {{$t("Available right now")}} ({{counts.available}})
        </b-radio-button>
        <b-radio-button v-model="dlState" native-value="downloaded" size="is-small">
          {{$t("Downloaded")}} ({{counts.downloaded}})
        </b-radio-button>
        <b-radio-button v-model="dlState" native-value="missing" size="is-small">
          {{$t("Not downloaded")}} ({{counts.not_downloaded}})
        </b-radio-button>
        <b-radio-button v-model="dlState" native-value="hidden" size="is-small">
          {{$t("Hidden")}} ({{counts.hidden}})
        </b-radio-button>
      </div>
      <div class="toolbar-size">
        <b-field>
          <span class="list-header-label">{{$t('Card size')}}</span>
          <b-radio-button v-model="cardSize" native-value="1" size="is-small">XS</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="2" size="is-small">S</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="3" size="is-small">M</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="4" size="is-small">L</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="5" size="is-small">XL</b-radio-button>
        </b-field>
      </div>
    </div>

    <div class="columns is-multiline" :class="{ 'is-reloading': isListReloading && items.length }">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in items" :key="item.id">
        <SceneCard :item="item"/>
      </div>
      <template v-if="isListReloading && !items.length">
        <div :class="['column', 'is-multiline', cardSizeClass]" v-for="n in skeletonCount" :key="'sk' + n">
          <div class="skeleton-card" aria-hidden="true">
            <div class="skeleton-block skeleton-thumb" :style="{ aspectRatio: skeletonAspect }"></div>
            <div class="skeleton-block skeleton-line"></div>
            <div class="skeleton-block skeleton-line is-short"></div>
          </div>
        </div>
      </template>
    </div>

    <div class="column is-full loading-more" v-if="isLoadingMore">
      <b-icon pack="mdi" icon="loading" custom-class="mdi-spin" size="is-medium"></b-icon>
      <span>Loading more...</span>
    </div>
    <div class="column is-full" v-if="!infiniteScrollEnabled && items.length < total">
      <b-button type="is-primary" @click="loadMore" :loading="isLoadingMore" expanded>{{$t('Load More')}}</b-button>
    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import SceneCard from './SceneCard'
import ky from 'ky'

export default defineComponent({
  name: 'List',
  components: { SceneCard },

  props: {
    infiniteScrollEnabled: {
      type: Boolean,
      default: true
    }
  },

  data() {
    return {
      isLoadingMore: false,
      scrollHandler: null,
      debounceTimeout: null
    }
  },

  computed: {
    cardSize: {
      get () {
        return this.$store.state.sceneList.filters.cardSize
      },
      set (value) {
        this.$store.state.sceneList.filters.cardSize = value
        localStorage.setItem('sceneCardSize', value)
      }
    },
    cardSizeClass () {
      switch (this.$store.state.sceneList.filters.cardSize) {
        case '1':
          return 'is-2'
        case '2':
          return 'is-one-fifth'
        case '3':
          return 'is-one-quarter'
        case '4':
          return 'is-one-third'
        case '5':
          return 'is-half'
        default:
          return 'is-one-fifth'
      }
    },
    dlState: {
      get () {
        return this.$store.state.sceneList.filters.dlState
      },
      set (value) {
        this.$store.state.sceneList.filters.dlState = value

        switch (this.$store.state.sceneList.filters.dlState) {
          case 'any':
            this.$store.state.sceneList.filters.isAvailable = null
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'available':
            this.$store.state.sceneList.filters.isAvailable = true
            this.$store.state.sceneList.filters.isAccessible = true
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'downloaded':
            this.$store.state.sceneList.filters.isAvailable = true
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'missing':
            this.$store.state.sceneList.filters.isAvailable = false
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = false
            break
          case 'hidden':
            this.$store.state.sceneList.filters.isAvailable = null
            this.$store.state.sceneList.filters.isAccessible = null
            this.$store.state.sceneList.filters.isHidden = true
            break
        }

        this.reloadList()
      }
    },
    isLoading () {
      return this.$store.state.sceneList.isLoading
    },
    isListReloading () {
      return this.isLoading && !this.isLoadingMore
    },
    // Two rows of placeholders for the current card size.
    skeletonCount () {
      const perRow = { 1: 6, 2: 5, 3: 4, 4: 3, 5: 2 }[this.$store.state.sceneList.filters.cardSize] || 5
      return perRow * 2
    },
    skeletonAspect () {
      return (this.$store.state.optionsWeb.web.sceneCardAspectRatio || '16:9').replace(':', ' / ')
    },
    items () {
      return this.$store.state.sceneList.items
    },
    total () {
      return this.$store.state.sceneList.total
    },
    counts () {
      return this.$store.state.sceneList.counts
    }
  },

  watch: {
    '$store.state.sceneList.show_scene_id' (id) {
      if (id && id !== '') {
        ky.get('/api/scene/' + id).json().then(data => {
          if (data.id != 0) {
            this.$store.commit('overlay/showDetails', { scene: data })
          }
        })
        this.$store.state.sceneList.show_scene_id = ''
      }
    },
    infiniteScrollEnabled(newVal) {
      if (newVal) {
        window.addEventListener('scroll', this.scrollHandler, { passive: true })
      } else {
        window.removeEventListener('scroll', this.scrollHandler)
      }
    }
  },

  methods: {
    reloadList () {
      this.$router.push({
        name: 'scenes',
        query: {
          q: this.$store.getters['sceneList/filterQueryParams']
        }
      })
    },
    async loadMore () {
      if (this.isLoadingMore || this.items.length >= this.total) return
      this.isLoadingMore = true
      await this.$store.dispatch('sceneList/load', { offset: this.$store.state.sceneList.offset })
      this.isLoadingMore = false
    },
    handleScroll () {
      if (this.debounceTimeout) return
      this.debounceTimeout = setTimeout(() => {
        this.debounceTimeout = null
        const scrollY = window.scrollY || window.pageYOffset
        const viewportHeight = window.innerHeight
        const fullHeight = document.documentElement.scrollHeight
        if (scrollY + viewportHeight + 600 >= fullHeight) {
          this.loadMore()
        }
      }, 150)
    }
  },

  mounted () {
    this.scrollHandler = this.handleScroll.bind(this)
    if (this.infiniteScrollEnabled) {
      window.addEventListener('scroll', this.scrollHandler, { passive: true })
    }
  },

  beforeUnmount () {
    if (this.scrollHandler) {
      window.removeEventListener('scroll', this.scrollHandler)
    }
    if (this.debounceTimeout) clearTimeout(this.debounceTimeout)
  },
});
</script>

<style scoped>
  .list-progress {
    position: sticky;
    top: 3.25rem;
    z-index: 7;
    height: 2px;
    margin-bottom: -2px;
    overflow: hidden;
    background: rgba(127, 127, 127, 0.15);
  }
  .list-progress::after {
    content: '';
    position: absolute;
    inset: 0 auto 0 0;
    width: 35%;
    background: var(--bulma-primary, #7957d5);
    animation: list-progress-slide 1.1s ease-in-out infinite;
  }
  @keyframes list-progress-slide {
    from { transform: translateX(-100%); }
    to { transform: translateX(290%); }
  }
  .is-reloading {
    opacity: 0.6;
    transition: opacity 150ms ease 150ms;
  }
  .skeleton-card {
    border-radius: 6px;
    overflow: hidden;
  }
  .skeleton-block {
    background: linear-gradient(90deg, rgba(127, 127, 127, 0.12) 25%, rgba(127, 127, 127, 0.22) 50%, rgba(127, 127, 127, 0.12) 75%);
    background-size: 200% 100%;
    animation: skeleton-shimmer 1.4s ease-in-out infinite;
    border-radius: 4px;
  }
  .skeleton-thumb {
    width: 100%;
    border-radius: 6px 6px 0 0;
  }
  .skeleton-line {
    height: 0.8em;
    margin: 0.6em 0.6em 0;
  }
  .skeleton-line.is-short {
    width: 55%;
    margin-bottom: 0.6em;
  }
  @keyframes skeleton-shimmer {
    from { background-position: 200% 0; }
    to { background-position: -200% 0; }
  }
  @media (prefers-reduced-motion: reduce) {
    .skeleton-block, .list-progress::after { animation: none; }
  }
  .toolbar-total {
    flex: 1 1 auto;
    white-space: nowrap;
  }
  .toolbar-segment {
    display: flex;
    flex-wrap: wrap;
    flex: 0 1 auto;
  }
  .toolbar-size {
    flex: 0 0 auto;
  }
  .toolbar-size :deep(.field) {
    margin-bottom: 0 !important;
  }
  .list-header-label {
    padding-right: 1em;
    white-space: nowrap;
    align-self: center;
  }
  .loading-more {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 20px;
    color: #888;
  }
</style>
