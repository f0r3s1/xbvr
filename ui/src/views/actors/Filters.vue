<template>
  <div class="filter-panel">
    <div class="is-divider" data-content="Saved searches"></div>

     <SavedSearch/> 

    <div class="is-divider" data-content="Properties"></div>

    <div class="columns is-mobile is-gapless property-filters">
      <div class="column" :title="$t('Watchlist')">
        <b-checkbox-button name="actor-lists" v-model="lists" native-value="watchlist" type="is-primary">
          <b-icon pack="mdi" icon="calendar-check"/>
          <span>{{ $t('Watchlist') }}</span>
        </b-checkbox-button>
      </div>
      <div class="column" :title="$t('Favourite')">
        <b-checkbox-button name="actor-lists" v-model="lists" native-value="favourite" type="is-danger">
          <b-icon pack="mdi" icon="heart"/>
          <span>{{ $t('Favourite') }}</span>
        </b-checkbox-button>
      </div>
      <div class="column" :title="$t('Actors with scenes in your library')">
        <b-checkbox-button name="actor-lists" v-model="collection" native-value="collection" type="is-success">
          <b-icon pack="mdi" icon="folder-check"/>
          <span>{{ $t('Collection') }}</span>
        </b-checkbox-button>
      </div>
    </div>

    <div class="is-divider" data-content="Sorting"></div>

    <b-field :label="$t('Sort by')" label-for="actor-filter-sort" label-position="on-border" :addons="true" class="field-extra">
      <div class="control is-expanded">
        <div class="select is-fullwidth">
          <select id="actor-filter-sort" v-model="sort">
            <option value="name_asc">↑ {{ $t("Name") }}</option>
            <option value="name_desc">↓ {{ $t("Name") }}</option>
            <option value="birthday_desc">↓ {{ $t("Birthdate") }}</option>
            <option value="birthday_asc">↑ {{ $t("Birthdate") }}</option>
            <option value="rating_desc">↓ {{ $t("Rating") }}</option>
            <option value="rating_asc">↑ {{ $t("Rating") }}</option>
            <option value="scene_rating_desc">↓ {{ $t("Average Scene Rating") }}</option>
            <option value="added_desc">↓ {{ $t("Added date") }}</option>
            <option value="added_asc">↑ {{ $t("Added date") }}</option>
            <option value="modified_desc">↓ {{ $t("Modified date") }}</option>
            <option value="modified_asc">↑ {{ $t("Modified date") }}</option>
            <option value="scene_release_desc">↓ {{ $t("Scene Released Date") }}</option>
            <option value="scene_added_desc">↓ {{ $t("Scene Added Date") }}</option>
            <option value="file_added_desc">↓ {{ $t("File Added Date") }}</option>
            <option value="scene_available_desc">↓ {{ $t("Available Scene Count") }}</option>
            <option value="scene_count_desc">↓ {{ $t("Scene Count") }}</option>            
            <option value="random">↯ {{ $t("Random") }}</option>
          </select>
        </div>
      </div>
    </b-field>

    <div class="is-divider" data-content="Actor Filters"></div>

    <div v-if="Object.keys(filters).length !== 0">
      <b-field :label="$t('Cast')" label-for="actor-filter-cast" label-position="on-border" class="field-extra">
        <b-taginput v-field-id="'actor-filter-cast'" v-model="cast" autocomplete :data="filteredCast" @typing="getFilteredCast">
          <template v-slot="props">{{ props.option }}</template>
          <template #empty>{{ $t("No matching cast") }}</template>
          <template #selected="props">
              <b-tag v-for="(tag, index) in props.tags"
                :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
                :key="tag+index" :tabstop="false" closable  @close="cast=cast.filter(e => e !== tag)" @click="toggle2Way(tag,index,'cast')">              
                <span class="chip-label" :title="tag.charAt(0)=='!' ? 'Exclude ' + removeConditionPrefix(tag) : tag.charAt(0)=='&' ? 'Must Have ' + removeConditionPrefix(tag) : 'Include ' + removeConditionPrefix(tag)">
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='&'" icon="plus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
                </span>
              </b-tag>
          </template>
        </b-taginput>
      </b-field>

      <b-field :label="$t('Site')" label-for="actor-filter-sites" label-position="on-border" class="field-extra">
        <b-taginput v-field-id="'actor-filter-sites'" v-model="sites" autocomplete :data="filteredSites" @typing="getFilteredSites">
          <template v-slot="props">{{ props.option }}</template>
          <template #empty>{{ $t("No matching sites") }}No matching sites</template>
          <template #selected="props">
            <b-tag v-for="(tag, index) in props.tags"
              :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
              :key="tag+index" :tabstop="false" closable  @close="sites=sites.filter(e => e !== tag)" @click="toggle3way(tag,index,'sites')">
                <span class="chip-label" :title="tag.charAt(0)=='!' ? 'Exclude ' + removeConditionPrefix(tag) : 'Include ' + removeConditionPrefix(tag)">
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
                </span>
            </b-tag>
          </template>
        </b-taginput>
      </b-field>

      <b-tooltip position="is-top" :label="$t('Allows searching a variety of attributes such as: Possible Aka actors, Cup Size, Eye/Hair Color, Has Tattoo, Has Piercing, Breast Type, Nationailty, Ethnicity, Aka, Has Images')" multilined :delay="1000" style="width:100%">
        <b-field :label="$t('Attributes')" label-for="actor-filter-attributes" label-position="on-border" class="field-extra">
          <b-taginput v-field-id="'actor-filter-attributes'" v-model="attributes" autocomplete :data="filteredAttributes" @typing="getFilteredAttributes">
            <template v-slot="props">{{ props.option }}</template>
            <template #empty>{{ $t("No matching attributes") }}</template>
            <template #selected="props">
              <b-tag v-for="(tag, index) in props.tags"
                :type="tag.charAt(0)=='!' ? 'is-danger': (tag.charAt(0)=='&' ? 'is-success' : '')"
                :key="tag+index" :tabstop="false" closable  @close="attributes=attributes.filter(e => e !== tag)" @click="toggle3way(tag,index,'attributes')"> 
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='!'" icon="minus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  <b-icon pack="mdi" v-if="tag.charAt(0)=='&'" icon="plus-circle-outline" size="is-small" class="tagicon"></b-icon>
                  {{removeConditionPrefix(tag)}}
              </b-tag>
            </template>          
          </b-taginput>
        </b-field>
      </b-tooltip>

      <div class="range-list">
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Age") }}</span>
            <span class="range-value">{{ rangeText(ages, 18, 100, '') }}</span>
          </div>
          <b-slider :min="18" :max="100" :step="1" :tooltip="true" v-model="ages" lazy size="is-small"></b-slider>
        </div>
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Height") }}</span>
            <span class="range-value">{{ rangeText(heights, 120, 220, ' cm') }}</span>
          </div>
          <b-slider :min="120" :max="220" :step="1" :tooltip="true" v-model="heights" lazy size="is-small"></b-slider>
        </div>
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Weight") }}</span>
            <span class="range-value">{{ rangeText(weights, 25, 150, ' kg') }}</span>
          </div>
          <b-slider :min="25" :max="150" :step="1" :tooltip="true" v-model="weights" lazy size="is-small"></b-slider>
        </div>
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Scenes") }}</span>
            <span class="range-value">{{ rangeText(scenecounts, 0, 150, '') }}</span>
          </div>
          <b-slider :min="0" :max="150" :step="1" :tooltip="true" v-model="scenecounts" lazy size="is-small"></b-slider>
        </div>
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Available") }}</span>
            <span class="range-value">{{ rangeText(avails, 0, 150, '') }}</span>
          </div>
          <b-slider :min="0" :max="150" :step="1" :tooltip="true" v-model="avails" lazy size="is-small"></b-slider>
        </div>
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Rating") }}</span>
            <span class="range-value">{{ rangeText(ratings, 0, 5, '') }}</span>
          </div>
          <b-slider :min="0" :max="5" :step=".5" :tooltip="true" v-model="ratings" lazy size="is-small"></b-slider>
        </div>
        <div class="range">
          <div class="range-head">
            <span class="range-label">{{ $t("Scene Rating") }}</span>
            <span class="range-value">{{ rangeText(sceneratings, 0, 5, '') }}</span>
          </div>
          <b-slider :min="0" :max="5" :step=".25" :tooltip="true" v-model="sceneratings" lazy size="is-small"></b-slider>
        </div>
      </div>
    </div>
    <div class="is-divider" data-content="Actor Also Known As groups"></div>
    <b-field class="group-actions">
      <b-tooltip position="is-right" :label="$t('New Aka Group. Select 2 or more actors in the Cast filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="createAkaGroup" :disabled="disableNewAkaGroup">
          <b-icon pack="mdi" icon="account-multiple-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-right" :label="$t('Select the Aka Group to delete in the Cast Filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="deleteAkaGroup" :disabled="disableDeleteAkaGroup">
          <b-icon pack="mdi" icon="delete-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" :label="$t('Add Cast to Aka Group. Select the Aka group and Actors to add in the Cast Filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="addToAkaGroup" :disabled="disableAddToAkaGroup">
          <b-icon pack="mdi" icon="account-plus-outline"></b-icon>
        </button>
      </b-tooltip>
      <b-tooltip position="is-bottom" :label="$t('Remove Cast from Aka Group. Select the Aka group and Actors to remove in the Cast Filter')" multilined :delay="200">
        <button class="button is-small is-outlined" @click="removeFromAkaGroup" :disabled="disableRemoveFromAkaGroup">
          <b-icon pack="mdi" icon="account-minus-outline"></b-icon>
        </button>
      </b-tooltip>

    </b-field>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import SavedSearch from './SavedSearch'
import ky from 'ky'

export default defineComponent({
  name: 'Filters',
  components: { SavedSearch },

  mounted () {
    this.$store.dispatch('actorList/filters')
    this.fetchFilters()
  },

  data () {
    return {
      filteredCast: [],
      filteredSites: [],
      filteredTags: [],
      filteredAttributes: [],
    }
  },

  methods: {
    // "Any" when the range is untouched, else "min-max" (open ends shown as "+")
    rangeText (v, min, max, unit) {
      if (!v || (v[0] <= min && v[1] >= max)) return 'Any'
      const hi = v[1] >= max ? v[1] + '+' : v[1]
      return `${v[0]}-${hi}${unit}`
    },
    reloadList () {
      this.$router.push({
        name: 'actors',
        query: {
          q: this.$store.getters['actorList/filterQueryParams']
        }
      })
    },
    // capped: thousands of actors must not render thousands of dropdown rows
    matchOptions (options, text, selected) {
      const q = (text || '').toLowerCase()
      const taken = new Set(selected.map(e => this.removeConditionPrefix(e.toString())))
      const starts = []
      const contains = []
      for (const option of options) {
        const name = option.toString()
        if (taken.has(name)) continue
        const lower = name.toLowerCase()
        const at = lower.indexOf(q)
        if (at === 0 || lower.startsWith('aka:' + q)) {
          starts.push(option)
          if (starts.length >= 50) break
        } else if (at > 0 && contains.length < 50) {
          contains.push(option)
        }
      }
      return starts.concat(contains).slice(0, 50)
    },
    getFilteredCast (text) {
      this.filteredCast = this.matchOptions(this.filters.cast, text, this.cast)
    },
    getFilteredSites (text) {
      this.filteredSites = this.matchOptions(this.filters.sites, text, this.sites)
    },
    getFilteredAttributes (text) {
      this.filteredAttributes = this.matchOptions(this.filters.attributes, text, this.attributes)
    },
    createAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/create', {json: {actorList: this.cast}}).json().then(data => {
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('actorList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.actorList.isLoading = false
      })
    },
    deleteAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/delete', {json: {name: this.cast[0]}}).json().then(() => {
        this.cast = []
        this.$store.dispatch('actorList/filters')
        this.reloadList()
        this.$store.state.actorList.isLoading = false
      })
    },
    addToAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/add', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name) 
        this.$store.dispatch('actorList/filters')       
        this.reloadList()        
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.actorList.isLoading = false
      })
      
    },
    removeFromAkaGroup () {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/aka/remove', {json: {actorList: this.cast}}).json().then(data => {        
        // delete old aka & add new name
        this.cast = this.cast.filter(e => !e.startsWith("aka:")) 
        this.cast.push(data.akas.aka_actor.name)
        this.$store.dispatch('actorList/filters')
        this.reloadList()
        if (data.status != '') {
          this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
        }
        this.$store.state.actorList.isLoading = false
      })
    },
    toggle3way (text, idx, list) {      
      let tags = []
      switch (list) {
        case 'cast':
          tags=this.cast 
          break
        case 'sites':
          tags=this.sites
          break
        case 'attributes':
          tags=this.attributes
          break
      }      
      switch(tags[idx].charAt(0)) {
        case '!':
          tags[idx]=this.removeConditionPrefix(tags[idx])
          break
        case '&':
          tags[idx]='!' + this.removeConditionPrefix(tags[idx])
        break
        default:
        tags[idx]='&'+text        
      }      
      switch (list) {
        case 'cast':
          this.cast=tags
          break
        case 'sites':
          this.sites=tags
          break
        case 'attributes':
          this.attributes=tags
          break
      }
    },    
    toggle2Way (text, idx, list) {      
      let tags = []
      switch (list) {
        case 'sites':
          tags=this.sites
          break
        case 'cast':
          tags=this.cast
          break
      }      
      switch(tags[idx].charAt(0)) {
        case '!':
          tags[idx]=this.removeConditionPrefix(tags[idx])
          break
        default:
        tags[idx]='!'+text        
      }      
      switch (list) {
        case 'sites':
          this.sites=tags
          break
        case 'cast':
          this.cast=tags
          break
      }      
    },    
    removeConditionPrefix(txt) {
      if (txt.charAt(0)=='!' || txt.charAt(0)=='&') {
        return txt.substring(1) 
      }
      return txt
    },
    async fetchFilters() {
        this.filteredAttributes=['Loading attributes']
        ky.get('/api/actor/filters').json().then(data => {
          this.filteredAttributes=data.attributes          
      })      
    }
  },

  computed: {
    filters () {
      return this.$store.state.actorList.filterOpts
    },
    lists: {
      get () {
        return this.$store.state.actorList.filters.lists
      },
      set (value) {
        this.$store.state.actorList.filters.lists = value
        this.reloadList()
      }
    },
    cast: {
      get () {        
        return this.$store.state.actorList.filters.cast
      },
      set (value) {
        this.$store.state.actorList.filters.cast = value
        this.reloadList()
      }
    },
    sites: {
      get () {
        return this.$store.state.actorList.filters.sites
      },
      set (value) {
        this.$store.state.actorList.filters.sites = value
        this.reloadList()
      }
    },
    ages: {
      get () {
        return [this.$store.state.actorList.filters.min_age, this.$store.state.actorList.filters.max_age]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_age != value[0] || this.$store.state.actorList.filters.max_age != value[1]) {
          this.$store.state.actorList.filters.min_age = value[0]
          this.$store.state.actorList.filters.max_age = value[1]
          this.reloadList()
        }
      }
    },
    heights: {
      get () {
        return [this.$store.state.actorList.filters.min_height, this.$store.state.actorList.filters.max_height]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_height != value[0] || this.$store.state.actorList.filters.max_height != value[1]){
          this.$store.state.actorList.filters.min_height = value[0]
          this.$store.state.actorList.filters.max_height = value[1]
          this.reloadList()
        }
      }
    },
    weights: {
      get () {
        return [this.$store.state.actorList.filters.min_weight, this.$store.state.actorList.filters.max_weight]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_weight != value[0] || this.$store.state.actorList.filters.max_weight != value[1]){
          this.$store.state.actorList.filters.min_weight = value[0]
          this.$store.state.actorList.filters.max_weight = value[1]
          this.reloadList()
        }
      }
    },
    scenecounts: {
      get () {
        return [this.$store.state.actorList.filters.min_count, this.$store.state.actorList.filters.max_count]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_count != value[0] || this.$store.state.actorList.filters.max_count != value[1]){
          this.$store.state.actorList.filters.min_count = value[0]
          this.$store.state.actorList.filters.max_count = value[1]
          this.reloadList()
        }
      }
    },
    collection: {
      get () {
        return this.$store.state.actorList.filters.min_avail > 0 ? ['collection'] : []
      },
      set (value) {
        this.avails = [value.length ? 1 : 0, this.$store.state.actorList.filters.max_avail]
      }
    },
    avails: {
      get () {
        return [this.$store.state.actorList.filters.min_avail, this.$store.state.actorList.filters.max_avail]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_avail != value[0] || this.$store.state.actorList.filters.max_avail != value[1]){
          this.$store.state.actorList.filters.min_avail = value[0]
          this.$store.state.actorList.filters.max_avail = value[1]
          this.reloadList()
        }
      }
    },
    ratings: {
      get () {
        return [this.$store.state.actorList.filters.min_rating, this.$store.state.actorList.filters.max_rating]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_rating != value[0] || this.$store.state.actorList.filters.max_rating != value[1]){
          this.$store.state.actorList.filters.min_rating = value[0]
          this.$store.state.actorList.filters.max_rating = value[1]
          this.reloadList()
        }
      }
    },
    sceneratings: {
      get () {
        return [this.$store.state.actorList.filters.min_scene_rating, this.$store.state.actorList.filters.max_scene_rating]
      },
      set (value) {
        if (this.$store.state.actorList.filters.min_scene_rating != value[0] || this.$store.state.actorList.filters.max_scene_rating != value[1]){
          this.$store.state.actorList.filters.min_scene_rating = value[0]
          this.$store.state.actorList.filters.max_scene_rating = value[1]
          this.reloadList()
        }
      }
    },
    attributes: {
      get () {
        return this.$store.state.actorList.filters.attributes
      },
      set (value) {
        this.$store.state.actorList.filters.attributes = value
        this.reloadList()        
      }
    },
    sort: {
      get () {
        return this.$store.state.actorList.filters.sort
      },
      set (value) {
        this.$store.state.actorList.filters.sort = value
        this.reloadList()
      }
    },
    disableNewAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }
      // you can create a new group from a list of actors (more than one)
      return akaCastCnt == 0 && actorCnt > 1 ? false : true
    },
    disableDeleteAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can only delete a group when it is the only thing selected      
      return akaCastCnt == 1 && actorCnt == 0 > 1 ? false : true
    },
    disableAddToAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can add to a group if you select one group and one or more actors
      return akaCastCnt == 1 && actorCnt > 0 ? false : true
    },
    disableRemoveFromAkaGroup() {
      let akaCastCnt = 0
      let actorCnt = 0       
 
      for (var i = 0; i < this.cast.length; i++) {        
        if (this.cast[i].startsWith("aka:")) {
          akaCastCnt++
        } else {
          actorCnt++
        }
      }

      // you can remove from a group if you select one group and one or more actors
      return akaCastCnt == 1 && actorCnt > 0 ? false : true

    },
},
});
</script>

<style lang="scss" scoped>
.is-divider {
  position: relative;
  display: block;
  border-top: 0.1rem solid rgba(127, 127, 127, 0.25);
  height: 0.1rem;
  margin: 1.5rem 0;
  text-align: center;
}
.is-divider[data-content]::after {
  background: #fff;
  color: #b5b5b5;
  content: attr(data-content);
  display: inline-block;
  font-size: 0.75rem;
  padding: 0.4rem 0.8rem;
  transform: translateY(-1.1rem);
  text-align: center;
}

.is-gapless div.control {
  margin: 0.1rem;
}

.field-extra {
  margin-bottom: 1.1em !important;
}

.tagicon {
  margin-right: -0.2em !important;
}
.range-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 4px;
}
.range-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 8px;
}
.range-label {
  font-size: 0.8rem;
  font-weight: 500;
}
.range-value {
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  opacity: 0.6;
}
/* inset the track by half a thumb so the thumbs line up with the labels */
.range :deep(.b-slider) {
  margin: 0 !important;
  padding: 0 7px;
  box-sizing: border-box;
}

</style>
