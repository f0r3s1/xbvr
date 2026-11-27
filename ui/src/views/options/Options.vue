<template>
  <div class="container">
    <div class="columns">
      <div class="column is-one-fifth custom-menu">
        <b-input v-model="search" icon="magnify" type="search" :placeholder="$t('Search settings')" class="menu-search" />
        <ul v-if="search.trim()" class="search-results">
          <li v-for="hit in hits" :key="hit.page + hit.text">
            <a @click="openHit(hit)"><span class="hit-page">{{ hit.pageLabel }}</span>{{ hit.text }}</a>
          </li>
          <li v-if="!hits.length" class="search-empty">{{ $t('No results') }}</li>
        </ul>
        <b-menu v-else :accordion="false">
          <b-menu-list v-for="group in groups" :key="group.label" :label="group.label">
            <b-menu-item v-for="page in group.pages" :key="page.id" :active="active === page.id" @click="setActive(page.id)">
              <template #label>
                <span class="menu-with-badge">{{ page.label }}
                  <span v-if="page.badge" class="menu-badge" :title="page.badgeTitle">{{ page.badge }}</span>
                </span>
              </template>
            </b-menu-item>
          </b-menu-list>
        </b-menu>
      </div>
      <div class="column">
        <div class="options-content" :class="{ settling }">
          <template v-for="page in pages" :key="page.id">
            <div v-if="visited.includes(page.id)" v-show="active === page.id" class="options-page" :data-page="page.id">
              <component v-for="(part, n) in page.parts" :key="n" :is="part[0]" v-bind="part[1] || {}"/>
            </div>
          </template>
          <SceneMatchParams v-if="showMatchParamsOverlay"/>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { defineComponent, markRaw } from 'vue';
import ky from 'ky'

import InterfaceWeb from './sections/InterfaceWeb'
import Storage from './sections/Storage'
import SceneDataScrapers from './sections/OptionsSceneDataScrapers'
import SceneCreate from './sections/OptionsSceneCreate'
import Funscripts from './sections/Funscripts'
import SceneDataImportExport from './sections/OptionsSceneDataImportExport'
import InterfaceDLNA from './sections/InterfaceDLNA.vue'
import Cache from './sections/Cache.vue'
import LibraryHealth from './sections/LibraryHealth.vue'
import Previews from './sections/Previews.vue'
import Schedules from './sections/Schedules.vue'
import InterfaceDeoVR from './sections/InterfaceDeoVR.vue'
import InterfaceAdvanced from './sections/InterfaceAdvanced.vue'
import SceneMatchParams from './overlays/SceneMatchParams.vue'

// tab ids from older links
const settingNodes = '.setting-item, .field, .switch, .b-checkbox, .b-radio, th, .card-header-title, .options-subtitle, .group-title, h4, h5'

// everyday words people type for settings that use other names
const synonyms = { mode: 'theme', dark: 'theme', light: 'theme', vpn: 'proxy', cloudflare: 'flaresolverr', login: 'authentication password', password: 'authentication', lang: 'language', translation: 'language', thumbnail: 'preview', trailer: 'preview', folder: 'storage path', delete: 'remove reset', backup: 'export' }

const normalize = text => (text || '').toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/\s+/g, ' ').trim()

function settingLabel (node) {
  const label = node.querySelector(':scope > .label, :scope > label, .setting-label, .control-label') || node.querySelector('.label, label')
  return (label || node).textContent.replace(/\s+/g, ' ').trim()
}

function settingHaystack (node) {
  const extra = [...node.querySelectorAll('option, [placeholder], .help, .language-hint')]
    .map(n => n.getAttribute('placeholder') || n.textContent).join(' ')
  return normalize(`${node.textContent} ${extra}`)
}

// how many of the typed words (or their synonyms) appear in the text
function score (words, text) {
  const hay = normalize(text)
  return words.filter(w => hay.includes(w) || (synonyms[w] || '').split(' ').some(s => s && hay.includes(s))).length
}

const pageAliases = { interface_advanced: 'interface_web', details: 'interface_web', 'alternate-sites': 'custom-sites', interface_dlna: 'interface_deovr', schedules: 'cache', 'create-scene': 'data-import-export' }

export default defineComponent({
  components: { Storage, SceneDataScrapers, SceneCreate, Funscripts, SceneDataImportExport, InterfaceWeb, InterfaceDLNA, InterfaceDeoVR, Cache, Previews, Schedules, InterfaceAdvanced, SceneMatchParams, LibraryHealth },

  data: function () {
    return {
      active: 'storage',
      visited: [],
      search: '',
      hits: [],
      // no transitions while settings load, so toggles snap to their saved value
      settling: true
    }
  },

  methods: {
    async loadHealthSummary () {
      try {
        const data = await ky.get('/api/health/report', { timeout: 10000 }).json()
        if (data.report) this.$store.commit('health/setReport', data.report)
      } catch { /* the badge is optional */ }
    },
    // searches every setting on every page by its label, dropdown options, placeholder and help text
    async findSettings () {
      const words = normalize(this.search).split(' ').filter(w => w.length > 1)
      if (!words.length) { this.hits = []; return }
      for (const p of this.pages) if (!this.visited.includes(p.id)) this.visited.push(p.id)
      await this.$nextTick()
      const hits = []
      for (const p of this.pages) {
        const el = this.$el.querySelector(`.options-page[data-page="${p.id}"]`)
        if (!el) continue
        const seen = new Set()
        const pageScore = score(words, `${p.label} ${p.keywords || ''}`)
        if (pageScore) hits.push({ page: p.id, pageLabel: p.label, text: '', score: pageScore - 0.5 })
        for (const node of el.querySelectorAll(settingNodes)) {
          const text = settingLabel(node)
          if (!text || text.length > 80 || seen.has(text)) continue
          const s = score(words, settingHaystack(node))
          if (!s) continue
          const ranked = s + score(words, text) / 2 // words in the setting's own name rank first
          seen.add(text)
          hits.push(markRaw({ page: p.id, pageLabel: p.label, text, node, score: ranked }))
        }
      }
      this.hits = hits.sort((a, b) => b.score - a.score).slice(0, 40)
    },
    async openHit (hit) {
      this.setActive(hit.page)
      if (!hit.node) return
      await this.$nextTick()
      hit.node.scrollIntoView({ block: 'center' })
      hit.node.classList.remove('search-flash')
      void hit.node.offsetWidth
      hit.node.classList.add('search-flash')
    },
    setActive: function (e) {
      e = pageAliases[e] || e
      if (!this.pages.some(p => p.id === e)) e = 'storage'
      if (this.active !== e) window.scrollTo(0, 0)
      this.active = e
      if (!this.visited.includes(e)) this.visited.push(e)
      localStorage.setItem('optionsActiveTab', e)
      if (this.$route.query.tab !== e) {
        this.$router.replace({ query: { ...this.$route.query, tab: e } }).catch(() => {})
      }
    }
  },

  mounted () {
    this.loadHealthSummary()
    setTimeout(() => { this.settling = false }, 600)
    this.setActive(this.$route.query.tab || localStorage.getItem('optionsActiveTab') || 'storage')
  },

  computed: {
    // pages are grouped by task; each shows one or more existing sections, so upstream section files stay unchanged
    groups () {
      const advanced = (tab, title) => ['InterfaceAdvanced', { tab, title }]
      return [
        { label: this.$t('Library'), pages: [
          { id: 'storage', label: this.$t('Storage'), parts: [['Storage']] },
          { id: 'previews', label: this.$t('Previews'), parts: [['Previews']] },
          { id: 'funscripts', label: this.$t('Funscripts'), parts: [['Funscripts']] }
        ] },
        { label: this.$t('Scrapers'), pages: [
          { id: 'data-scrapers', label: this.$t('Sites'), parts: [['SceneDataScrapers']] },
          { id: 'custom-sites', label: this.$t('Custom sites'), keywords: 'alternate', parts: [advanced(2, this.$t('Custom sites')), advanced(3, this.$t('Alternate sites'))] },
          { id: 'actor-data', label: this.$t('Actor data'), keywords: 'stashdb imperial', parts: [advanced(1, this.$t('Actor data'))] },
          { id: 'connection', label: this.$t('Connection'), keywords: 'flaresolverr proxy cookies headers', parts: [advanced(4, this.$t('Proxy')), advanced(5, this.$t('Cookies/Headers'))] }
        ] },
        { label: this.$t('Appearance'), pages: [
          { id: 'interface_web', label: this.$t('Interface'), keywords: 'web theme language cards scene details', parts: [['InterfaceWeb'], advanced(0, this.$t('Scene Details'))] }
        ] },
        { label: this.$t('Players'), pages: [
          { id: 'interface_deovr', label: this.$t('Players'), keywords: 'deovr heresphere dlna', parts: [['InterfaceDeoVR'], ['InterfaceDLNA']] },
          { id: 'api-key', label: this.$t('API & access'), keywords: 'login password token key', parts: [advanced(6, this.$t('API key'))] }
        ] },
        { label: this.$t('Maintenance'), pages: [
          { id: 'health', label: this.$t('Library Health'), badge: this.healthAttention, badgeTitle: this.$t('{n} issues need attention', { n: this.healthAttention }), parts: [['LibraryHealth']] },
          { id: 'cache', label: this.$t('Cache & tasks'), keywords: 'schedules', parts: [['Cache'], ['Schedules']] },
          { id: 'data-import-export', label: this.$t('Import & export'), keywords: 'backup restore create scene', parts: [['SceneDataImportExport'], ['SceneCreate']] }
        ] }
      ]
    },
    pages () {
      return this.groups.flatMap(g => g.pages)
    },
    healthAttention () {
      return this.$store.state.health.attention
    },
    showMatchParamsOverlay () {      
      return this.$store.state.overlay.sceneMatchParams.show
    },
    showSceneCreate() {
      if (this.$store.state.optionsSceneCreate.showSceneCreate){
        this.setActive('create-scene')
        this.$store.commit('optionsSceneCreate/showSceneCreate', false )
      }
      return this.$store.state.optionsSceneCreate.showSceneCreate;
    },
  },

  watch: {
    search () {
      clearTimeout(this._searchTimer)
      this._searchTimer = setTimeout(() => this.findSettings(), 150)
    },
    // Accessing showSceneCreate here forces re-evaluation of the computed when the store changes.
    showSceneCreate() {},
    // a (background or manual) check finished: refresh the badge
    '$store.state.health.progress.running' (running, was) {
      if (was && !running) this.loadHealthSummary()
    },
  },
});
</script>

<style scoped>
.menu-with-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.menu-badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: #f14668;
  color: #fff;
  font-size: 0.68rem;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
}
.options-content.settling :deep(*),
.options-content.settling :deep(*::before),
.options-content.settling :deep(*::after) {
  transition: none !important;
}

/* the menu stays in view while the page scrolls */
.custom-menu {
  position: sticky;
  top: 52px;
  align-self: flex-start;
}
.custom-menu :deep(.menu-label) {
  margin: 0.9rem 0 0.3rem;
  font-size: 0.68rem;
}
.custom-menu :deep(.menu-list a) {
  padding: 0.35rem 0.75rem;
  border-radius: 6px;
  font-size: 0.9rem;
}
@media screen and (max-width: 768px) {
  .custom-menu {
    position: static;
  }
}
.menu-search {
  margin-bottom: 1rem;
}
.search-results {
  list-style: none;
  margin: 0;
}
.search-results a {
  display: block;
  padding: 0.4rem 0.75rem;
  border-radius: 6px;
  font-size: 0.875rem;
  color: inherit;
}
.search-results a:hover {
  background: rgba(128, 128, 160, 0.14);
}
.hit-page {
  display: block;
  font-size: 0.7rem;
  opacity: 0.6;
}
.search-empty {
  padding: 0.4rem 0.75rem;
  font-size: 0.875rem;
  opacity: 0.6;
}
.options-content :deep(.search-flash) {
  animation: search-flash 1.6s ease;
  border-radius: 6px;
}
@keyframes search-flash {
  0%, 40% { background: rgba(124, 108, 240, 0.35); }
  100% { background: transparent; }
}
.menu-search :deep(.input) {
  height: 36px;
  border-radius: 8px;
  border-color: transparent !important;
  background: rgba(0, 0, 0, 0.04) !important;
  box-shadow: none !important;
  font-size: 0.875rem;
}
.menu-search :deep(.input:focus) {
  border-color: rgba(128, 128, 160, 0.45) !important;
}
.menu-search :deep(.icon) {
  height: 36px;
  color: #888 !important;
}
html[data-theme="dark"] .menu-search :deep(.input) {
  background: #222230 !important;
}
.options-page > * + * {
  margin-top: 1.5rem;
}
.options-content {
  padding-top: 0.5rem;
  position: relative;
  min-height: 200px;
}
@media screen and (max-width: 768px) {
  .container {
    padding: 0 16px;
  }
}
</style>
