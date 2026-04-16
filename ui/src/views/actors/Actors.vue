<template>
  <div class="container is-fluid">
    <div class="columns">

      <div class="column is-one-fifth">
         <Filters/> 

        <a id="toTop">
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
  components: { Filters, List},

  mounted () {
    const toTop = document.getElementById('toTop')
    addEventListener('scroll', function () {
      toTop.style.display = document.body.scrollTop > 20 || document.documentElement.scrollTop > 20
        ? 'block'
        : 'none'
    })
    toTop.addEventListener('click', function () {
      scrollToTop()
    })

    const scrollToTop = () => {
      const c = document.documentElement.scrollTop || document.body.scrollTop
      if (c > 0) {
        window.requestAnimationFrame(scrollToTop)
        window.scrollTo(0, c - c / 16)
      }
    }
  },

  beforeRouteEnter (to) {
    return vm => {
      if (to.query !== undefined) {
        vm.$store.commit('actorList/stateFromQuery', to.query)
      }
      vm.$store.dispatch('optionsWeb/load')
      vm.$store.dispatch('actorList/load', { offset: 0 })
      vm.$store.dispatch('optionsAdvanced/load')
    }
  },

  beforeRouteUpdate (to) {
    if (to.query !== undefined) {
      this.$store.commit('actorList/stateFromQuery', to.query)
    }
    this.$store.dispatch('actorList/load', { offset: 0 })
  },

  computed: {
  },
});
</script>

<style scoped>
  #toTop {
    display: none;
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
