<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
        <Filters ref="filters"/>
        <p class="fork-credit">{{ $t('This fork is maintained by') }} <a href="https://github.com/f0r3s1/xbvr" target="_blank" rel="noopener">f0r3s1</a></p>

        <a id="toTop" :class="{ 'is-visible': showScrollButtons }" @click="scrollToTop">
          <b-icon pack="mdi" icon="navigation" />
        </a>
      </div>

      <div class="column is-four-fifths">
        <List/>
      </div>

    </div>
    
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import Filters from './Filters'
import List from './List'

export default defineComponent({
  name: 'Actors',
  components: { Filters, List },

  data () {
    return {
      showScrollButtons: false
    }
  },

  methods: {
    handleScroll () {
      this.showScrollButtons = window.scrollY > 0 &&
        this.$refs.filters.$el.getBoundingClientRect().bottom < window.innerHeight - 90
    },
    scrollToTop () {
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  },

  mounted () {
    this._scrollHandler = this.handleScroll.bind(this)
    window.addEventListener('scroll', this._scrollHandler, { passive: true })
  },

  beforeUnmount () {
    window.removeEventListener('scroll', this._scrollHandler)
  },

  beforeRouteEnter (to) {
    return vm => {
      const store = vm.$store
      const before = JSON.stringify(store.state.actorList.filters)
      if (to.query !== undefined) {
        store.commit('actorList/stateFromQuery', to.query)
      }
      store.dispatch('optionsWeb/load')
      store.dispatch('optionsAdvanced/load')
      // coming back with the same filters keeps the list as it is
      if (store.state.actorList.actors.length === 0 || JSON.stringify(store.state.actorList.filters) !== before) {
        store.dispatch('actorList/load', { offset: 0 })
      }
    }
  },

  beforeRouteUpdate (to) {
    if (to.query !== undefined) {
      this.$store.commit('actorList/stateFromQuery', to.query)
    }
    this.$store.dispatch('actorList/load', { offset: 0 })
  },
});
</script>

<style scoped>
  #toTop {
    position: fixed;
    bottom: 20px;
    left: 30px;
    z-index: 100;
    background-color: #f0f0f0;
    color: #4a4a4a;
    padding: 15px;
    border-radius: 10px;
    font-size: 18px;
  }

  #toTop {
    opacity: 0;
    pointer-events: none;
  }
  #toTop.is-visible {
    opacity: 1;
    pointer-events: auto;
    transition: opacity 0.2s ease;
  }
  #toTop:hover {
    background-color: #BDBDBD;
  }

  html[data-theme="dark"] #toTop {
    background-color: #2a2a3a !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] #toTop:hover {
    background-color: #3a3a4a !important;
  }
</style>
