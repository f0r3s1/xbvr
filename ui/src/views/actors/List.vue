<template>
  <div style="position: relative; min-height: 200px;">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keydown.left="prevpage"
      @keydown.right="nextpage"
      @keydown.o="prevpage"
      @keydown.p="nextpage"
    />
    <b-loading :is-full-page="false" v-model="isLoading"></b-loading>

    <div class="list-toolbar">
      <strong class="toolbar-total">{{total}} results</strong>
      <div class="simple-pagination">
        <button class="button is-small" :disabled="current <= 1" @click="prevpage">&#8249;</button>
        <input type="number" class="pg-input" v-model.number="current" :min="1" :max="totalPages" @change="pageChanged" />
        <span class="pg-of">/ {{ totalPages }}</span>
        <button class="button is-small" :disabled="current >= totalPages" @click="nextpage">&#8250;</button>
      </div>
      <div class="toolbar-size">
        <b-field>
          <span class="list-header-label">{{$t('Card size')}}</span>
          <b-radio-button v-model="cardSize" native-value="1" size="is-small">XS</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="2" size="is-small">S</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="3" size="is-small">M</b-radio-button>
          <b-radio-button v-model="cardSize" native-value="4" size="is-small">L</b-radio-button>
        </b-field>
      </div>
    </div>
    <span v-show="show_actor_id==='never show, just need the computed show_actor_id to trigger '">{{show_actor_id}}</span>
        <div class="letter-jump" v-if="hideLetters">
          <b-radio-button v-model="jumpTo" native-value="" size="is-small"></b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="A" size="is-small">A</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="B" size="is-small">B</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="C" size="is-small">C</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="D" size="is-small">D</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="E" size="is-small">E</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="F" size="is-small">F</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="G" size="is-small">G</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="H" size="is-small">H</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="I" size="is-small">I</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="J" size="is-small">J</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="K" size="is-small">K</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="L" size="is-small">L</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="M" size="is-small">M</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="N" size="is-small">N</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="O" size="is-small">O</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="P" size="is-small">P</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="Q" size="is-small">Q/R</b-radio-button>          
          <b-radio-button v-model="jumpTo" native-value="S" size="is-small">S</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="T" size="is-small">T</b-radio-button>
          <b-radio-button v-model="jumpTo" native-value="U" size="is-small">U/V</b-radio-button>          
          <b-radio-button v-model="jumpTo" native-value="W" size="is-small">W/X/Y/Z</b-radio-button>
        </div>

    <div class="is-clearfix"></div>

    <div class="columns is-multiline">
      <div :class="['column', 'is-multiline', cardSizeClass, 'actor-col']"
           v-for="actor in actors" :key="actor.id">
        <ActorCard :actor="actor"/>
      </div>
    </div>
      <div class="letter-jump" v-if="hideLetters">
        <b-radio-button v-model="jumpTo" native-value="" size="is-small"></b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="A" size="is-small">A</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="B" size="is-small">B</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="C" size="is-small">C</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="D" size="is-small">D</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="E" size="is-small">E</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="F" size="is-small">F</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="G" size="is-small">G</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="H" size="is-small">H</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="I" size="is-small">I</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="J" size="is-small">J</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="K" size="is-small">K</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="L" size="is-small">L</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="M" size="is-small">M</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="N" size="is-small">N</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="O" size="is-small">O</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="P" size="is-small">P</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="Q" size="is-small">Q/R</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="S" size="is-small">S</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="T" size="is-small">T</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="U" size="is-small">U/V</b-radio-button>
        <b-radio-button v-model="jumpTo" native-value="W" size="is-small">W/X/Y/Z</b-radio-button>
      </div>
      <div class="pagination-bottom">
        <div class="simple-pagination">
          <button class="button is-small" :disabled="current <= 1" @click="prevpage">&#8249;</button>
          <input type="number" class="pg-input" v-model.number="current" :min="1" :max="totalPages" @change="pageChanged" />
          <span class="pg-of">/ {{ totalPages }}</span>
          <button class="button is-small" :disabled="current >= totalPages" @click="nextpage">&#8250;</button>
        </div>
      </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import ActorCard from './ActorCard'
import ky from 'ky'
import { GlobalEvents } from 'vue-global-events'

export default defineComponent({
  name: 'List',
  components: { ActorCard, GlobalEvents },

  data () {
    return {      
      current: 1,      
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
        switch (value){
          case "1":
            this.limit=36
            break
          case "2":
            this.limit=18
            break
          case "3":
            this.limit=10
            break
          case "4":
            this.limit=8
            break
            }            
        }      
    },
    limit: {
      get(){
        return this.$store.state.actorList.limit
      },
      set(newLimit){
        // find the position of the first actor
        let currentOffset = this.$store.state.actorList.offset - this.$store.state.actorList.limit + 1
        // what is the new page number, based on the new limit
        this.current = Math.floor(currentOffset / newLimit) + 1
        if (this.current<1)
          this.current=1
        this.$store.state.actorList.limit = newLimit
        // what is the the first actor based on the new page size
        this.$store.state.actorList.offset = (this.current -1) * this.$store.state.actorList.limit          
        this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset })
      }
    },
    jumpTo: {
      get () {
        return this.$store.state.actorList.filters.jumpTo
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
        default:
          return 'is-2'
      }
    },
    isLoading () {
      this.current = this.$store.state.actorList.offset / this.$store.state.actorList.limit
      return this.$store.state.actorList.isLoading
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
    hideLetters: {
      get () {
        switch (this.$store.state.actorList.filters.sort) {
          case "":
            return true
          case "name_asc":
            return true
          case "name_desc":
            return true
        }
        return false
        },
    },
    totalPages () {
      return Math.max(1, Math.ceil(this.total / this.limit))
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
    async pageChanged () {      
      this.$store.state.actorList.offset = (this.current -1) * this.$store.state.actorList.limit
      this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset })
    },
    nextpage () {
      if (this.$store.state.overlay.actordetails.show){
        return 
      }
      if (this.$store.state.overlay.details.show){
        return 
      }
      if (this.current * this.limit >= this.total) {
        this.current = 1
      } else {
        this.current += 1
      }      
      this.pageChanged()
    },
    prevpage () {
      if (this.$store.state.overlay.actordetails.show){
        return
      }
      if (this.$store.state.overlay.details.show){
        return
      }
      if (this.current > 1) {
        this.current -= 1
      } else {
        this.current = Math.floor(this.total / this.limit) + 1
      }
      this.pageChanged()
    },
  },

  watch: {
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
  /* Toolbar row: results — pagination — card size */
  .list-toolbar {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.5rem;
    padding: 0.4rem 0 0;
    margin-bottom: 0.75rem;
  }
  /* Bulma .columns has margin-top: -0.75rem by default, which eats the toolbar's margin-bottom */
  .columns.is-multiline {
    margin-top: 0 !important;
  }
  .toolbar-total {
    flex: 1 1 auto;
    white-space: nowrap;
  }
  .toolbar-size {
    flex: 0 0 auto;
  }
  /* Remove buefy field's default bottom margin inside toolbar so it doesn't add double space */
  .toolbar-size :deep(.field) {
    margin-bottom: 0 !important;
  }

  /* Compact custom pagination */
  .simple-pagination {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    flex: 0 0 auto;
  }
  .pg-input {
    width: 2.5rem;
    height: 1.75rem;
    text-align: center;
    border: 1px solid #dbdbdb;
    border-radius: 4px;
    font-size: 0.75rem;
    padding: 0 0.2rem;
    color: inherit;
    background: transparent;
    -moz-appearance: textfield;
  }
  html[data-theme="dark"] .pg-input {
    border-color: #4a4a5a;
  }
  .pg-input::-webkit-inner-spin-button,
  .pg-input::-webkit-outer-spin-button { -webkit-appearance: none; }
  .pg-of {
    font-size: 0.75rem;
    white-space: nowrap;
    opacity: 0.7;
  }

  /* Bottom pagination centred */
  .pagination-bottom {
    display: flex;
    justify-content: center;
    padding: 0.5rem 0 0.75rem;
  }

  /* Letter jump bar */
  .letter-jump {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 3px;
    margin: 0.25rem 0 0.5rem;
    padding: 0 0.25rem;
  }

  /* Card columns */
  .actor-col {
    overflow: hidden;
    min-width: 0;
  }

  .list-header-label {
    padding-right: 0.5em;
  }
</style>
