<template>
  <div style="position: relative; min-height: 200px;">
    <div class="list-toolbar" ref="toolbar">
      <strong class="toolbar-total">{{ showSkeleton ? $t('Loading…') : total + ' results' }}</strong>
      <nav class="letter-filter" aria-label="Filter by first letter">
        <button v-for="l in letters" :key="l.value" type="button" class="letter"
                :class="{ 'is-active': jumpTo === l.value, 'is-all': l.value === '' }"
                :aria-pressed="jumpTo === l.value" @click="jumpTo = l.value">{{ l.label }}</button>
      </nav>
      <div class="toolbar-size">
        <b-field>
          <span class="list-header-label">{{$t('Card size')}}</span>
          <b-radio-button name="actor-card-size" v-model="cardSize" native-value="1" size="is-small">XS</b-radio-button>
          <b-radio-button name="actor-card-size" v-model="cardSize" native-value="2" size="is-small">S</b-radio-button>
          <b-radio-button name="actor-card-size" v-model="cardSize" native-value="3" size="is-small">M</b-radio-button>
          <b-radio-button name="actor-card-size" v-model="cardSize" native-value="4" size="is-small">L</b-radio-button>
          <b-radio-button name="actor-card-size" v-model="cardSize" native-value="5" size="is-small">XL</b-radio-button>
        </b-field>
      </div>
    </div>
    <span v-show="show_actor_id==='never show, just need the computed show_actor_id to trigger '">{{show_actor_id}}</span>

    <div class="columns is-multiline list-grid">
      <div :class="['column', 'is-multiline', cardSizeClass, 'actor-col']"
           v-for="actor in (showSkeleton ? [] : actors)" :key="actor.id">
        <ActorCard :actor="actor"/>
      </div>
      <template v-if="showSkeleton">
        <div :class="['column', 'is-multiline', cardSizeClass, 'actor-col']" v-for="n in skeletonCount" :key="'sk' + n">
          <div class="skeleton-card" aria-hidden="true">
            <div class="skeleton-block skeleton-thumb" :style="{ aspectRatio: skeletonAspect }"></div>
            <div class="skeleton-info">
              <div class="skeleton-block skeleton-line"></div>
              <div class="skeleton-block skeleton-line is-short"></div>
            </div>
          </div>
        </div>
      </template>
      <div class="column is-full empty-state" v-if="!isLoading && !actors.length">
        {{ jumpTo ? `No actors starting with ${jumpTo}` : 'No actors match these filters' }}
      </div>
    </div>

    <div class="loading-more" v-if="isLoadingMore">
      <b-icon pack="mdi" icon="loading" custom-class="mdi-spin" size="is-medium"></b-icon>
      <span>Loading more...</span>
    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import ActorCard from './ActorCard'
import ky from 'ky'

export default defineComponent({
  name: 'List',
  components: { ActorCard },

  data () {
    return {
      isLoadingMore: false,
      lastScrollY: 0,
      isSlowLoad: false,
      slowLoadTimer: null,
      letters: [{ value: '', label: 'All' }]
        .concat('ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('').map(l => ({ value: l, label: l })))
        .concat([{ value: '#', label: '#' }])
    }
  },

  computed: {
    cardSize: {
      get () {
        return this.$store.state.actorList.filters.cardSize
      },
      set (value) {
        this.$store.state.actorList.filters.cardSize = value
        localStorage.setItem('actorCardSize', value)
        this.$nextTick(() => this.fillViewport())
      }
    },
    // first-letter filter (kept in the URL with the other filters)
    jumpTo: {
      get () {
        return this.$store.state.actorList.filters.jumpTo || ''
      },
      set (value) {
        this.$store.state.actorList.filters.jumpTo = value
        this.reloadList()
      }
    },
    cardSizeClass () {
      switch (this.$store.state.actorList.filters.cardSize) {
        case '1':
          return 'is-1'
        case '2':
          return 'is-2'
        case '3':
          return 'is-one-fifth'
        case '4':
          return 'is-one-quarter'
        case '5':
          return 'is-one-third'
        default:
          return 'is-2'
      }
    },
    isLoading () {
      return this.$store.state.actorList.isLoading
    },
    isListReloading () {
      return this.isLoading && !this.isLoadingMore
    },
    showSkeleton () {
      return this.isListReloading && this.isSlowLoad
    },
    skeletonCount () {
      const perRow = { 1: 8, 2: 6, 3: 5, 4: 4, 5: 3 }[this.cardSize] || 6
      return perRow * 2
    },
    skeletonAspect () {
      return (this.$store.state.optionsWeb.web.actorCardAspectRatio || '1:1').replace(':', ' / ')
    },
    actors () {
      return this.$store.state.actorList.actors
    },
    total () {
      return this.$store.state.actorList.total
    },
    show_actor_id() {
      return this.$store.state.actorList.show_actor_id
    },
  },

  methods: {
    reloadList () {
      this.$router.push({
        name: 'actors',
        query: {
          q: this.$store.getters['actorList/filterQueryParams']
        }
      })
    },
    async loadMore () {
      const list = this.$store.state.actorList
      if (this.isLoadingMore || list.isLoading || list.actors.length >= list.total) return
      this.isLoadingMore = true
      await this.$store.dispatch('actorList/load', { offset: list.actors.length })
      this.isLoadingMore = false
      this.$nextTick(() => this.fillViewport())
    },
    fillViewport () {
      if (document.documentElement.scrollHeight < window.innerHeight + 400) this.loadMore()
    },
    handleScroll () {
      if (this.scrollTimer) return
      this.scrollTimer = setTimeout(() => {
        this.scrollTimer = null
        const y = window.scrollY
        const movedDown = y > this.lastScrollY
        this.lastScrollY = y
        if (movedDown && y + window.innerHeight + 800 >= document.documentElement.scrollHeight) this.loadMore()
      }, 120)
    },
  },

  mounted () {
    this.toolbarObserver = new ResizeObserver(([entry]) => {
      document.documentElement.style.setProperty('--list-head-h', `${entry.target.offsetHeight}px`)
    })
    this.toolbarObserver.observe(this.$refs.toolbar)
    this.scrollHandler = () => this.handleScroll()
    window.addEventListener('scroll', this.scrollHandler, { passive: true })
  },

  beforeUnmount () {
    window.removeEventListener('scroll', this.scrollHandler)
    if (this.scrollTimer) clearTimeout(this.scrollTimer)
    clearTimeout(this.slowLoadTimer)
    this.toolbarObserver.disconnect()
    document.documentElement.style.removeProperty('--list-head-h')
  },

  watch: {
    isListReloading (loading) {
      clearTimeout(this.slowLoadTimer)
      this.isSlowLoad = false
      if (loading) this.slowLoadTimer = setTimeout(() => { this.isSlowLoad = true }, 350)
    },
    // after a (re)load finishes, keep filling until the screen is full
    '$store.state.actorList.isLoading' (loading) {
      if (!loading && !this.isLoadingMore) this.$nextTick(() => this.fillViewport())
    },
    '$store.state.actorList.show_actor_id' (id) {
      if (id && id !== '') {
        ky.get('/api/actor/' + id).json().then(data => {
          if (data.id != 0) {
            this.$store.commit('overlay/showActorDetails', { actor: data })
          }
        })
        this.$store.state.actorList.show_actor_id = ''
      }
    }
  },
});
</script>

<style scoped>
  .toolbar-total {
    flex: 1 1 0;
    white-space: nowrap;
  }
  .toolbar-size {
    flex: 1 1 0;
    display: flex;
    justify-content: flex-end;
  }
  .toolbar-size :deep(.field) {
    margin-bottom: 0 !important;
  }
  .letter-filter {
    flex: 0 1 auto;
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 1px;
  }
  .letter {
    min-width: 1.4rem;
    height: 1.6rem;
    padding: 0 0.3rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: inherit;
    font: inherit;
    font-size: 0.75rem;
    font-weight: 500;
    line-height: 1.6rem;
    text-align: center;
    opacity: 0.65;
    cursor: pointer;
    transition: background 0.12s, opacity 0.12s;
  }
  .letter.is-all {
    padding: 0 0.6rem;
  }
  .letter:hover {
    opacity: 1;
    background: rgba(127, 127, 127, 0.15);
  }
  .letter.is-active {
    opacity: 1;
    background: var(--bulma-primary, #7957d5);
    color: #fff;
  }
  .letter:focus-visible {
    outline: 2px solid var(--bulma-primary, #7957d5);
    outline-offset: 1px;
  }

  .empty-state {
    padding: 3rem 0;
    text-align: center;
    opacity: 0.6;
  }

  .loading-more {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 20px;
    color: #888;
  }

  /* Card columns */
  .actor-col {
    overflow: hidden;
    min-width: 0;
  }

</style>
