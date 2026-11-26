<template>
  <div class="container">
    <div class="columns">

      <div class="column is-one-fifth custom-menu">
        <b-menu :accordion="false">
          <b-menu-list :label="$t('Options')">
            <b-menu-item :label="$t('Storage')" :active="active==='storage'" @click="setActive('storage')"/>
            <b-menu-item :label="$t('Previews')" :active="active==='previews'" @click="setActive('previews')"/>
            <b-menu-item :label="$t('Cache')" :active="active==='cache'" @click="setActive('cache')"></b-menu-item>
            <b-menu-item :active="active==='health'" @click="setActive('health')">
              <template #label>
                <span class="menu-with-badge">{{ $t('Library Health') }}
                  <span v-if="healthAttention" class="menu-badge" :title="healthAttention + ' issues need attention'">{{ healthAttention }}</span>
                </span>
              </template>
            </b-menu-item>
            <b-menu-item :label="$t('Task Schedules')" :active="active==='schedules'" @click="setActive('schedules')"></b-menu-item>
          </b-menu-list>
          <b-menu-list :label="$t('Scene data')">
            <b-menu-item :label="$t('Scrapers')" :active="active==='data-scrapers'"
                         @click="setActive('data-scrapers')"/>
            <b-menu-item :label="$t('Create/Import scene')" :active="active==='create-scene'"
                         @click="setActive('create-scene')"/>
            <b-menu-item :label="$t('Funscripts')" :active="active==='funscripts'"
                         @click="setActive('funscripts')"/>
            <b-menu-item :label="$t('Data import/export')" :active="active==='data-import-export'"
                         @click="setActive('data-import-export')"/>
          </b-menu-list>
          <b-menu-list :label="$t('Interfaces')">
            <b-menu-item :label="$t('Players')" :active="active==='interface_deovr'" @click="setActive('interface_deovr')"/>
            <b-menu-item :label="$t('DLNA')" :active="active==='interface_dlna'" @click="setActive('interface_dlna')"/>
            <b-menu-item :label="$t('Web UI')" :active="active==='interface_web'" @click="setActive('interface_web')"/>
            <b-menu-item :label="$t('Advanced')" :active="active==='interface_advanced'" @click="setActive('interface_advanced')"/>
          </b-menu-list>
        </b-menu>
      </div>

      <div class="column">
        <div class="options-content" :class="{ settling }">
          <Storage v-show="active==='storage'"/>
          <Cache v-show="active==='cache'"/>
          <LibraryHealth v-show="active==='health'"/>
          <Previews v-show="active==='previews'"/>
          <Schedules v-show="active==='schedules'"/>
          <SceneDataScrapers v-show="active==='data-scrapers'"/>
          <SceneCreate v-show="active==='create-scene'"/>
          <Funscripts v-show="active==='funscripts'"/>
          <SceneDataImportExport v-show="active==='data-import-export'"/>
          <InterfaceWeb v-show="active==='interface_web'"/>
          <InterfaceDLNA v-show="active==='interface_dlna'"/>
          <InterfaceDeoVR v-show="active==='interface_deovr'"/>
          <InterfaceAdvanced v-show="active==='interface_advanced'"/>
          <SceneMatchParams v-if="showMatchParamsOverlay"/>
        </div>
      </div>

    </div>
  </div>
</template>

<script>
import { defineComponent } from 'vue';
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

export default defineComponent({
  components: { Storage, SceneDataScrapers, SceneCreate, Funscripts, SceneDataImportExport, InterfaceWeb, InterfaceDLNA, InterfaceDeoVR, Cache, Previews, Schedules, InterfaceAdvanced, SceneMatchParams, LibraryHealth },

  data: function () {
    return {
      active: this.$route.query.tab || localStorage.getItem('optionsActiveTab') || 'storage',
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
    setActive: function (e) {
      this.active = e
      localStorage.setItem('optionsActiveTab', e)
      if (this.$route.query.tab !== e) {
        this.$router.replace({ query: { ...this.$route.query, tab: e } }).catch(() => {})
      }
    }
  },

  mounted () {
    this.loadHealthSummary()
    setTimeout(() => { this.settling = false }, 600)
    if (this.$route.query.tab) {
      this.active = this.$route.query.tab
    }
  },

  computed: {
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
