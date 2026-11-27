<template>
  <b-modal :modelValue="isModalActive" @update:modelValue="v => { if (!v) close() }"
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           :can-cancel="['escape', 'outside']">
    

    <div class="modal-card stashdb-search">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t('Search StashDB scenes') }}</p>
        <button class="delete" @click="close" :aria-label="$t('close')"></button>
      </header>

      <div class="modal-card-body">
                <div >
          <b-field>
            <b-input v-model="queryString" :placeholder="$t('Search StashDB by title, cast or studio')" @update:modelValue="debouncedSearch" :loading="isFetching" icon="magnify" expanded/>
          </b-field>
          <b-field>
            <b-input v-model="manualUrl" :placeholder="$t('…or paste a StashDB scene URL')" icon="link-variant" expanded/>
            <p class="control">
              <b-button :disabled="!manualUrl.includes('stashdb.org/scenes/')" @click="linktoStashdb({ Url: manualUrl.trim() })">{{ $t('Link') }}</b-button>
            </p>
          </b-field>
          <p v-if="isFetching" class="search-state">{{ $t('Searching StashDB…') }}</p>
          <p v-else-if="statusMessage" class="search-state">{{ statusMessage }}</p>

        <b-table v-if="searchResults.length" :data="searchResults" >
          <b-table-column field="ImageUrl" >
            <template v-slot="props">
              <div class="media">
                <div class="media-left">
                    <vue-load-image>
                        <template #image><img :src="getImageURL(props.row.ImageUrl)" width="150" @mouseover="showTooltipImage(props.row.ImageUrl)" @mouseout="showTooltipImage('')"/></template>
                        <template #preloader><img src="/ui/images/blank.png" height="150"/></template>
                        <template #error><img src="/ui/images/blank.png" height="150"/></template>
                    </vue-load-image>
                    <div v-if="tooltipImage!='' && tooltipImage==props.row.ImageUrl" class="tooltipimg">
                      <img :src="tooltipImage" :alt="$t('Tooltip Image')" width="400px" />
                    </div>
                  <div v-if="props.row.Date!=''"><small><strong>{{ $t('Released:') }}</strong> {{format(parseISO(props.row.Date), "yyyy-MM-dd")}}</small></div>
                  <div v-if="props.row.Duration!=''"><small><strong>{{ $t('Durn:') }}</strong> {{ props.row.Duration }}</small></div>
                  <div><small><strong>{{ $t('Score:') }}</strong> {{ props.row.Weight }}</small></div>
                  <div>
                    <a class="button is-primary is-small" @click="linktoStashdb(props.row)" :title="$t('Link scene with stashdb')">
                      <b-icon pack="mdi" :icon="'link-variant-plus'" size="is-small"/>
                    </a>
                  </div>
                </div>
                <div class="media-content">
                  <div class="truncate"><strong><a :href="props.row.Url"  target="_blank">{{ props.row.Studio }} - {{ props.row.Title }}</a></strong></div>
                  <div><small style="white-space: normal; display: block;">{{props.row.Description}}</small></div>
                  <div style="margin-top:0.5em">
                    <small style="white-space: normal; display: block;">
                      <span v-for="(c, idx) in props.row.Performers" :key="'Performers' + idx">{{c.Name}}<span v-if="idx < props.row.Performers.length-1">, </span></span>
                    </small>
                  </div>
                </div>            
              </div>
            </template>
          </b-table-column>
        </b-table>
        </div>
      </div>
      <footer class="modal-card-foot">
      </footer>
    </div>
  </b-modal>
</template>

<script>
import { defineComponent, nextTick } from 'vue';
import { imageURL } from '@/imageURL'

import ky from 'ky'
import VueLoadImage from 'vue-load-image'
import { format, parseISO } from 'date-fns'
import { debounce } from '../../debounce'


export default defineComponent({
  name: 'SearchStashdbScenes',
  components: { VueLoadImage },

  data () {
    return {
        isModalActive: true,
        stashdbUrl: "",
        manualUrl: "",
        statusMessage: "",
        searchResults: [],
        queryString: "",
        isFetching: false,
        tooltipImage: '',
        scene: "",
        }        
  },

  created() {
    this.debouncedSearch = debounce(this.searchStashdb, 750);
  },

  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchStashDbScenes.scene)    
    this.scene = item
    this.openDialog(item)    
  },

  methods: {
    format,
    parseISO,
    close () {
      this.$store.commit('overlay/hideSearchStashdbScenes')
    },
    searchStashdb () {
      this.isFetching = true
      this.statusMessage = ''
      ky.get('/api/extref/stashdb/search/' + this.scene.id, { searchParams: { q: this.queryString }, timeout: 120000 }).json()
        .then(data => {
          this.searchResults = Object.values(data.Results || {}).sort((a, b) => b.Weight - a.Weight)
          if (!this.searchResults.length) this.statusMessage = data.Status || this.$t('No StashDB scenes found. Try other words or paste the scene URL.')
        })
        .catch(() => { this.statusMessage = this.$t('StashDB search failed. Check the StashDB API key in Options → Advanced.') })
        .finally(() => { this.isFetching = false })
    },
    selectScene(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/scenes/","")
        nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
    },    
    linktoStashdb(option) {
        this.stashdbUrl=option.Url.replace("https://stashdb.org/scenes/","").split(/[?#/]/)[0]
        ky.get('/api/extref/stashdb/link2scene/' + this.scene.id +'/'+this.stashdbUrl ).json().then(data => {          
          this.$store.commit('sceneList/updateScene', data)
          this.$store.commit('overlay/showDetails', { scene: data })
          this.close()
        })
    },    
    getImageURL (u, size = '200x') {
      return imageURL(u, size)
    },
    openDialog(scene) {
        this.isModalActive = true
        this.searchStashdb()
        this.$store.commit('overlay/changeDetailsTab', { tab: 3 })
        nextTick(() => {
            if (this.$refs.autocompleteInput) {
                this.$refs.autocompleteInput.focus();
            }
        });
        this.scene = scene
    },
    showTooltipImage(val){
      this.tooltipImage=val
    },
  },

  computed: {

},
});
</script>

<style scoped>
.stashdb-search {
  width: min(760px, 94vw);
  height: 80vh;
}
.search-state {
  margin: 0.5rem 0 1rem;
  font-size: 0.85rem;
  opacity: 0.7;
}
.b-modal {
  left: -20%;
  width: 40%;
  height: 65%;
  overflow: auto;
}

.tab-item {
  height: 40vh;
}
.tooltipimg {
  position: absolute;
  z-index: 1;
  width: 350;
  background-color: rgba(20, 20, 28, 0.92);
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.3);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 10px;
  transform: translateX(60px) translateY(-50px);
}
.tooltipimg img {
  max-width: 100%;
  max-height: 100%;
}
</style>
