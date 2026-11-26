<template>
  <b-navbar :fixed-top="true" type="is-light">
    <template #brand>
      <b-navbar-item tag="a" href="#/" @click.prevent="goHome">
        <h1 class="title">XBVR <small>{{currentVersion}}</small></h1>
      </b-navbar-item>
    </template>
    <template #start>
      <b-navbar-item tag="router-link" :to="{ path: './' }">
        {{$t('Scenes')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './actors' }">
        {{$t('Actors')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './files' }">
        {{$t('Files')}}
      </b-navbar-item>
      <b-navbar-item tag="router-link" :to="{ path: './options' }">
        {{$t('Options')}}
      </b-navbar-item>
      <b-navbar-item @click="$store.commit('overlay/showQuickFind')">
        {{$t('Quick find')}}
        <kbd class="nav-kbd">{{ quickFindShortcut }}</kbd>
      </b-navbar-item>
    </template>
    <template #end>
      <!-- Library Health running (background checks are throttled to stay light) -->
      <b-navbar-item v-if="health.running" tag="router-link" :to="{ path: './options', query: { tab: 'health' } }"
                     class="health-indicator" :title="healthTitle">
        <span class="health-dot"></span>
        <span class="health-label">{{ health.background ? 'Background check' : 'Health check' }}</span>
      </b-navbar-item>
      <b-navbar-item v-if="hasStatusMessages">
        <table style="font-size:0.9em">
          <tr v-if="Object.keys(lastRescanMessage).length !== 0">
            <th><span :class="[lockRescan ? 'pulsate' : '']">{{$t('Files')}} →</span></th>
            <td>
              {{lastRescanMessage.message}}
              <b-button size="is-small" class="ml-2" @click="$store.state.messages.lastRescanMessage = {}">×</b-button>
            </td>
          </tr>
          <tr v-if="Object.keys(lastScrapeMessage).length !== 0">
            <th><span :class="[lockScrape ? 'pulsate' : '']">{{$t('Data')}} →</span></th>
            <td>
              {{lastScrapeMessage.message}}
              <b-button v-if="lockScrape" size="is-small" type="is-danger" class="ml-2" :disabled="stopping" @click="stopScrape">
                {{ stopping ? $t('Stopping…') : $t('Stop') }}
              </b-button>
              <b-button v-else size="is-small" class="ml-2" @click="$store.state.messages.lastScrapeMessage = {}">×</b-button>
            </td>
          </tr>
        </table>
      </b-navbar-item>
    </template>
  </b-navbar>
</template>

<script>
import { defineComponent } from 'vue';

import ky from 'ky'

export default defineComponent({
  data () {
    return {
      currentVersion: '',
      latestVersion: '',
      stopping: false
    }
  },

  computed: {
    quickFindShortcut () {
      return /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘K' : 'Ctrl K'
    },
    health () {
      return this.$store.state.health.progress
    },
    healthTitle () {
      const h = this.health
      const step = h.step && h.step !== 'done' ? ` · step ${h.step_num}/${h.total_steps}: ${h.step}` : ''
      return (h.background ? 'Library Health is checking in the background at low power (max ~20% of one core)' : 'Library Health check running') + step
    },
    lockRescan () {
      return this.$store.state.messages.lockRescan
    },
    lastRescanMessage () {
      return this.$store.state.messages.lastRescanMessage
    },
    lockScrape () {
      return this.$store.state.messages.lockScrape
    },
    lastScrapeMessage () {
      return this.$store.state.messages.lastScrapeMessage
    },
    hasStatusMessages () {
      return Object.keys(this.lastRescanMessage).length !== 0 ||
             Object.keys(this.lastScrapeMessage).length !== 0
    }
  },

  watch: {
    lockScrape (locked) {
      if (!locked) this.stopping = false
    }
  },

  methods: {
    stopScrape () {
      this.stopping = true
      ky.get('/api/task/scrape/cancel').catch(() => { this.stopping = false })
    },
    goHome () {
      this.$store.commit('sceneList/resetFilters')
      this.$store.commit('overlay/hideAll')
      if (this.$route.name === 'scenes' && !Object.keys(this.$route.query).length) {
        this.$store.dispatch('sceneList/load', { offset: 0 })
      } else {
        this.$router.push({ name: 'scenes' })
      }
      window.scrollTo({ top: 0 })
    }
  },

  mounted () {
    ky.get('/api/options/version-check').json().then(data => {
      this.currentVersion = data.current_version
      this.latestVersion = data.latest_version

      if (data.update_notify && this.currentVersion !== 'CURRENT') {
        this.$buefy.snackbar.open({
          message: `Version ${this.latestVersion} available!`,
          type: 'is-warning',
          position: 'is-bottom-right',
          actionText: this.$t('Download now'),
          indefinite: true,
          onAction: () => {
            window.location = 'https://github.com/f0r3s1/xbvr/releases'
          }
        })
      }
    }).catch(() => {
      // The update check is best-effort; never let it blank the version label.
    })
  },
});
</script>

<style>
.health-indicator {
  display: flex !important;
  align-items: center;
  gap: 6px;
  font-size: 0.78rem;
  opacity: 0.75;
}
.health-indicator:hover {
  opacity: 1;
}
.health-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #48c78e;
  animation: health-pulse 1.6s ease-in-out infinite;
}
@keyframes health-pulse {
  0%, 100% { opacity: 0.35; }
  50% { opacity: 1; }
}
@media (prefers-reduced-motion: reduce) {
  .health-dot { animation: none; }
}
</style>

<style scoped>
.nav-kbd {
  margin-left: 0.5rem;
  padding: 0.05rem 0.4rem;
  border: 1px solid rgba(127, 127, 127, 0.35);
  border-radius: 6px;
  background: transparent;
  color: inherit;
  font-size: 0.7rem;
  opacity: 0.6;
}
@media (max-width: 1023px) {
  .nav-kbd {
    display: none;
  }
}
  h1 {
    display: flex;
    align-items: center;
  }

  h1 small {
    font-size: 0.5em;
    margin-left: 0.5em;
    opacity: 0.5;
  }

  th {
    padding-right: 1em;
  }

  .pulsate {
    -webkit-animation: pulsate 0.5s linear;
    -webkit-animation-iteration-count: infinite;
    opacity: 0.5;
  }

  @-webkit-keyframes pulsate {
    0% {
      opacity: 0.5;
    }
    50% {
      opacity: 1.0;
    }
    100% {
      opacity: 0.5;
    }
  }
</style>
