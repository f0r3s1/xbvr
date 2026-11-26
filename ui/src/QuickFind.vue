<template>
  <b-modal :modelValue="isActive" @update:modelValue="isActive = $event"
           :destroy-on-hide="false"
           :render-on-mounted="true"
           has-modal-card
           trap-focus
           aria-role="dialog"
           aria-modal
           :can-cancel="['escape', 'outside']">
    <div class="quickfind">
    <b-field class="quickfind-input">
      <b-autocomplete
        ref="autocompleteInput"
        id="quickfind-input"
        :data="data"
        :placeholder="$t('Find scenes and people…')"
        field="query"
        group-field="group"
        group-options="items"
        :loading="isFetching"
        v-model="queryString"
        @typing="getAsyncData"
        @select="onSelect"
        :open-on-focus="true"
        custom-class="is-large"
        max-height="450">

        <template v-slot="props">
          <div v-if="props.option.kind === 'site'" class="qf-person">
            <span class="qf-site-logo">
              <img v-if="props.option.avatar_url" :src="getImageURL(props.option.avatar_url)" alt=""/>
              <span v-else>{{ props.option.name.slice(0, 2).toUpperCase() }}</span>
            </span>
            <span class="qf-person-name">{{ props.option.name }}</span>
            <span class="qf-person-meta">{{ props.option.scene_count }} scenes</span>
          </div>
          <div v-else-if="props.option.kind === 'actor'" class="qf-person">
            <img :src="props.option.image_url ? getImageURL(props.option.image_url) : '/ui/images/actor_placeholder.svg'" class="qf-person-img" alt=""/>
            <span class="qf-person-name">{{ props.option.name }}</span>
            <span class="qf-person-meta">{{ props.option.scene_count }} scenes</span>
          </div>
          <div v-else class="qf-scene">
            <img :src="getImageURL(props.option.cover_url)" class="qf-scene-img" alt="" @error="e => e.target.src = '/ui/images/blank.png'"/>
            <div class="qf-scene-body">
              <div class="truncate"><strong>{{ props.option.title }}</strong>
                <b-icon v-if="props.option.is_hidden" pack="mdi" icon="eye-off-outline" size="is-small"/></div>
              <div class="qf-scene-meta">{{ props.option.site }} · {{ format(parseISO(props.option.release_date), 'yyyy-MM-dd') }}</div>
              <div v-if="matchedCast(props.option).length" class="qf-scene-meta truncate">{{ matchedCast(props.option).join(', ') }}</div>
            </div>
          </div>
        </template>
      </b-autocomplete>
      <kbd class="quickfind-kbd">{{ shortcutLabel }}</kbd>
    </b-field>
    <div class="quickfind-fields">
      <span class="quickfind-label">{{ $t('Search in') }}</span>
      <button v-for="f in fieldChips" :key="f.label" type="button" class="qf-chip" :title="f.hint" @click="f.action">{{ f.label }}</button>
    </div>
    </div>
  </b-modal>
</template>

<script>
import { defineComponent, nextTick } from 'vue';

import ky from 'ky'
import { loadSites, siteFilterQuery } from './sites'
import { format, parseISO } from 'date-fns'

export default defineComponent({
  name: 'ModalNewTag',

  props: {
    active: Boolean,
    sceneId: String
  },



  computed: {
    fieldChips () {
      return [
        { label: 'title', hint: 'Search titles', action: () => this.searchPrefix('+title:') },
        { label: 'cast', hint: 'Search cast names', action: () => this.searchPrefix('cast:') },
        { label: 'site', hint: 'Search sites', action: () => this.searchPrefix('+site:') },
        { label: 'id', hint: 'Search scene ids', action: () => this.searchPrefix('+id:') },
        { label: 'duration', hint: 'Filter by file duration in minutes', action: () => this.searchDurationPrefix('duration:') },
        { label: 'released', hint: 'Released in the last week (yyyy-mm-dd)', action: () => this.searchDatePrefix('released:') },
        { label: 'added', hint: 'Added in the last week (yyyy-mm-dd)', action: () => this.searchDatePrefix('added:') }
      ]
    },
    shortcutLabel () {
      return /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘K' : 'Ctrl K'
    },
    isActive: {
      get () {
        return this.$store.state.overlay.quickFind.show
      },
      set (value) {
        this.$store.state.overlay.quickFind.show = value
      }
    }
  },

  watch: {
    isActive (val) {
      if (val) {
        if (this.queryString != null && this.queryString != "") {
          this.getAsyncData(this.queryString)
        }
        nextTick(() => {
          const ac = this.$refs && this.$refs.autocompleteInput
          const input = ac && ac.$refs && ac.$refs.input
          if (input && input.focus) input.focus()
          const qs = this.$store.state.overlay.quickFind.searchString
          if (qs != null && qs != "") {
            this.queryString = qs
            this.$store.state.overlay.quickFind.searchString = null
          }
        })
      }
    }
  },

  data () {
    return {
      data: [],
      dataNumRequests: 0,
      dataNumResponses: 0,
      selected: null,
      isFetching: false,
      queryString: ""
    }
  },

  methods: {
    format,
    parseISO,
    getAsyncData (query) {
      clearTimeout(this.typingTimer)
      if (!query.length) return this.search(query)
      this.typingTimer = setTimeout(() => this.search(query), 150)
    },
    async search (query) {
      const requestIndex = this.dataNumRequests
      this.dataNumRequests = this.dataNumRequests + 1

      if (!query.length) {
        this.data = []
        this.dataNumResponses = requestIndex + 1
        this.isFetching = false
        return
      }

      this.isFetching = true

      const name = query.replace(/\+?\w+:("[^"]*"|\S*)/g, '').trim() || (query.match(/cast:"?([^"]+)"?/) || [])[1] || ''
      const [resp, people] = await Promise.all([
        ky.get('/api/scene/search', { searchParams: { q: query, fuzzy: 1 } }).json(),
        name.length > 1
          ? ky.post('/api/actor/list', { json: { name, limit: 5, offset: 0, sort: 'scene_count_desc' } }).json().catch(() => ({ actors: [] }))
          : Promise.resolve({ actors: [] })
      ])

      if (requestIndex >= this.dataNumResponses) {
        this.dataNumResponses = requestIndex + 1
        if (this.dataNumResponses === this.dataNumRequests) {
          this.isFetching = false
        }

        const groups = []
        const sites = await loadSites()
        const siteHits = name.length > 1 ? sites.filter(s => s.name.toLowerCase().includes(name.toLowerCase())).slice(0, 4) : []
        if (siteHits.length) groups.push({ group: 'Studios', items: siteHits.map(s => ({ ...s, kind: 'site' })) })
        if (people.actors && people.actors.length) {
          groups.push({ group: 'People', items: people.actors.map(a => ({ ...a, kind: 'actor' })) })
        }
        if (resp.results > 0) groups.push({ group: 'Scenes', items: resp.scenes })
        this.data = groups
      }
    },
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/200x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    matchedCast (scene) {
      const words = (this.queryString || '').toLowerCase().replace(/\+?\w+:/g, ' ').split(/[\s"]+/).filter(w => w.length > 1)
      if (!words.length || !scene.cast) return []
      return scene.cast.map(c => c.name).filter(n => words.some(w => n.toLowerCase().includes(w)))
    },
    onSelect (option) {
      if (!option) return
      if (option.kind === 'site') {
        this.$store.commit('overlay/hideQuickFind')
        this.data = []
        this.$router.push(siteFilterQuery(this.$store, option.name))
        return
      }
      if (option.kind === 'actor') {
        this.$store.commit('overlay/hideQuickFind')
        this.data = []
        ky.get('/api/actor/' + option.id).json().then(actor => {
          this.$store.commit('overlay/showActorDetails', { actor })
        })
        return
      }
      this.showSceneDetails(option)
    },
    showSceneDetails (scene) {
      this.$store.commit('overlay/hideQuickFind')
      if (this.$store.state.overlay.quickFind.displaySelectedScene) {
        if (this.$router.currentRoute.name !== 'scenes') {
            this.$router.push({ name: 'scenes' })
          }
          this.$store.commit('overlay/hideQuickFind')
          this.data = []
          this.$store.commit('overlay/showDetails', { scene })
        } else {
          // don't display the scene, just pass the selected scene back in the $store.state and close
          this.$store.state.overlay.quickFind.selectedScene = scene          
          this.$store.commit('overlay/hideQuickFind')
          this.data = []
      }
    },
    searchPrefix(prefix) {      
      let textbox = this.$refs.autocompleteInput.$refs.input.$refs.input
      if (textbox.selectionStart != textbox.selectionEnd) {
        let selected = textbox.value.substring(textbox.selectionStart, textbox.selectionEnd)
        selected=selected.replace(/_/g," ").replace(/-/g," ").trim()
        if (selected.indexOf(' ') >= 0) {
          selected='"' + selected + '"'
        }
        this.queryString = textbox.value.substring(0,textbox.selectionStart) + " " + prefix + selected + " " + textbox.value.substr(textbox.selectionEnd)
      } else {
        const value = this.queryString || ''
        this.queryString = (value.trim() ? value.trim() + ' ' : '') + prefix
      }
      this.getAsyncData(this.queryString)
      this.$refs.autocompleteInput.focus()
    },
    searchDatePrefix(prefix) {      
        let today = new Date().toISOString().slice(0, 10)
        let weekago = new Date(Date.now() - 604800000).toISOString().slice(0, 10)
        if (this.queryString == undefined) {
          this.queryString = prefix + '>="' + weekago + '" ' +  prefix + '<="' + today + '"'          
        } else {
          this.queryString = this.queryString.trim() + ' ' + prefix + '>="' + weekago + '" ' +  prefix + '<="' + today + '"'        
        }
        this.getAsyncData(this.queryString)
        this.$refs.autocompleteInput.focus()
    },
    searchDurationPrefix(prefix) {
      if (this.queryString == undefined) {
        this.queryString = prefix + '>=0'
      } else {
        this.queryString = this.queryString.trim() + ' ' + prefix + '>=0'
      }
      this.getAsyncData(this.queryString)
      this.$refs.autocompleteInput.focus()
    }
  },
});
</script>

<style scoped>
  .modal {
    justify-content: normal;
    padding-top: 9vh;
  }
  .modal :deep(.animation-content),
  .modal :deep(.modal-content) {
    pointer-events: none;
  }
  .quickfind {
    width: min(640px, calc(100vw - 32px));
    margin: 0 auto;
    position: relative;
    z-index: 1;
    pointer-events: auto;
  }
  .quickfind-input {
    position: relative;
    margin-bottom: 0.6rem;
  }
  .quickfind-input :deep(.autocomplete) {
    width: 100%;
  }
  .quickfind-input :deep(input) {
    padding-right: 4.5rem;
  }
  .quickfind-kbd {
    position: absolute;
    right: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    padding: 0.15rem 0.45rem;
    border: 1px solid rgba(127, 127, 127, 0.35);
    border-radius: 6px;
    background: transparent;
    color: inherit;
    font-size: 0.75rem;
    opacity: 0.6;
    pointer-events: none;
  }
  .quickfind-fields {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
  }
  .qf-chip {
    height: 1.6rem;
    padding: 0 0.6rem;
    border: 1px solid rgba(255, 255, 255, 0.22);
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.06);
    color: #fff;
    font: inherit;
    font-size: 0.75rem;
    cursor: pointer;
    transition: background 0.15s ease, border-color 0.15s ease;
  }
  .qf-chip:hover {
    background: rgba(255, 255, 255, 0.14);
    border-color: rgba(255, 255, 255, 0.4);
  }
  .qf-scene {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
  }
  .qf-scene-img {
    flex: 0 0 72px;
    width: 72px;
    height: 41px;
    border-radius: 6px;
    object-fit: cover;
  }
  .qf-scene-body {
    min-width: 0;
    flex: 1;
  }
  .qf-scene-meta {
    font-size: 0.8rem;
    opacity: 0.65;
  }
  .qf-person {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  .qf-site-logo {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    overflow: hidden;
    display: grid;
    place-items: center;
    background: rgba(127, 127, 127, 0.18);
    font-size: 0.7rem;
    font-weight: 700;
  }
  .qf-site-logo img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .qf-person-img {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    object-fit: cover;
  }
  .qf-person-name {
    flex: 1;
    font-weight: 600;
  }
  .qf-person-meta {
    font-size: 0.8rem;
    opacity: 0.6;
  }
  .quickfind-label {
    margin-right: 4px;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    opacity: 0.6;
    color: #fff;
  }
  .truncate {
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .quickfind :deep(.media-content) {
    min-width: 0;
  }
</style>
