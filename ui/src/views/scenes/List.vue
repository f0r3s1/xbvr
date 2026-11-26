<template>
  <div ref="scrollContainer" style="position: relative; min-height: 200px;">
    <div class="list-toolbar" ref="toolbar">
      <strong class="toolbar-total">{{ showStudios ? studioCount + ' studios' : showSkeleton ? $t('Loading…') : total + ' results' }}</strong>
      <div class="toolbar-segment">
        <b-radio-button name="scene-view" v-model="tab" native-value="any" size="is-small">
          {{$t("Any")}} ({{counts.any}})
        </b-radio-button>
        <b-radio-button name="scene-view" v-model="tab" native-value="available" size="is-small">
          {{$t("Available right now")}} ({{counts.available}})
        </b-radio-button>
        <b-radio-button name="scene-view" v-model="tab" native-value="downloaded" size="is-small">
          {{$t("Downloaded")}} ({{counts.downloaded}})
        </b-radio-button>
        <b-radio-button name="scene-view" v-model="tab" native-value="missing" size="is-small">
          {{$t("Not downloaded")}} ({{counts.not_downloaded}})
        </b-radio-button>
        <b-radio-button name="scene-view" v-model="tab" native-value="hidden" size="is-small">
          {{$t("Hidden")}} ({{counts.hidden}})
        </b-radio-button>
        <b-radio-button name="scene-view" v-model="tab" native-value="studios" size="is-small">
          {{$t("Studios")}} ({{ studioCount }})
        </b-radio-button>
      </div>
      <div class="toolbar-size" :style="{ visibility: showStudios ? 'hidden' : null }">
        <b-field>
          <span class="list-header-label">{{$t('Card size')}}</span>
          <b-radio-button name="scene-card-size" v-model="cardSize" native-value="1" size="is-small">XS</b-radio-button>
          <b-radio-button name="scene-card-size" v-model="cardSize" native-value="2" size="is-small">S</b-radio-button>
          <b-radio-button name="scene-card-size" v-model="cardSize" native-value="3" size="is-small">M</b-radio-button>
          <b-radio-button name="scene-card-size" v-model="cardSize" native-value="4" size="is-small">L</b-radio-button>
          <b-radio-button name="scene-card-size" v-model="cardSize" native-value="5" size="is-small">XL</b-radio-button>
        </b-field>
      </div>
    </div>

    <StudioGrid v-if="showStudios"/>
    <template v-else>
    <SiteHeader/>

    <div class="columns is-multiline list-grid">
      <div :class="['column', 'is-multiline', cardSizeClass]"
           v-for="item in (showSkeleton ? [] : items)" :key="item.id">
        <SceneCard :item="item"/>
      </div>
      <template v-if="showSkeleton">
        <div :class="['column', 'is-multiline', cardSizeClass]" v-for="n in skeletonCount" :key="'sk' + n">
          <div class="skeleton-card" aria-hidden="true">
            <div class="skeleton-block skeleton-thumb" :style="{ aspectRatio: skeletonAspect }"></div>
            <div class="skeleton-info">
              <div class="skeleton-block skeleton-line"></div>
              <div class="skeleton-block skeleton-line is-short"></div>
            </div>
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
    </template>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import SceneCard from './SceneCard'
import SiteHeader from './SiteHeader.vue'
import StudioGrid from './StudioGrid.vue'
import { loadSites } from '../../sites'
import ky from 'ky'

export default defineComponent({
  name: 'List',
  components: { SceneCard, SiteHeader, StudioGrid },

  props: {
    infiniteScrollEnabled: {
      type: Boolean,
      default: true
    }
  },

  data() {
    return {
      isLoadingMore: false,
      lastScrollY: 0,
      studioCount: 0,
      isSlowLoad: false,
      slowLoadTimer: null,
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
    showStudios () {
      return this.$route.query.view === 'studios'
    },
    tab: {
      get () {
        return this.showStudios ? 'studios' : this.dlState
      },
      set (value) {
        if (value === 'studios') {
          this.$router.push({ query: { ...this.$route.query, view: 'studios' } })
        } else {
          this.dlState = value
        }
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
    showSkeleton () {
      return this.isListReloading && this.isSlowLoad
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
    isListReloading (loading) {
      clearTimeout(this.slowLoadTimer)
      this.isSlowLoad = false
      if (loading) this.slowLoadTimer = setTimeout(() => { this.isSlowLoad = true }, 350)
    },
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
        const y = window.scrollY
        const movedDown = y > this.lastScrollY
        this.lastScrollY = y
        if (movedDown && y + window.innerHeight + 600 >= document.documentElement.scrollHeight) {
          this.loadMore()
        }
      }, 150)
    }
  },

  mounted () {
    loadSites().then(sites => { this.studioCount = sites.filter(s => s.scene_count > 0).length })
    this.toolbarObserver = new ResizeObserver(([entry]) => {
      document.documentElement.style.setProperty('--list-head-h', `${entry.target.offsetHeight}px`)
    })
    this.toolbarObserver.observe(this.$refs.toolbar)
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
    clearTimeout(this.slowLoadTimer)
    this.toolbarObserver.disconnect()
    document.documentElement.style.removeProperty('--list-head-h')
  },
});
</script>

<style scoped>
  /* results | filter tabs (centred) | card size: both sides grow equally so the
     tabs sit in the middle of the toolbar */
  .toolbar-total {
    flex: 1 1 0;
    white-space: nowrap;
  }
  .toolbar-segment {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    flex: 0 1 auto;
  }
  .toolbar-size {
    flex: 1 1 0;
    display: flex;
    justify-content: flex-end;
  }
  .toolbar-size :deep(.field) {
    margin-bottom: 0 !important;
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
