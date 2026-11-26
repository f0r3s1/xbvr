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
      <b-navbar-item v-if="hasStatusMessages" tag="div" class="task-status">
        <div v-if="Object.keys(lastRescanMessage).length !== 0" class="task-row">
          <span class="task-text">
            <span class="task-label" :class="{ 'is-running': lockRescan }">{{ $t('Files') }}</span>
            <span class="task-message" :title="lastRescanMessage.message">{{ lastRescanMessage.message }}</span>
          </span>
          <button class="task-dismiss" :aria-label="$t('Dismiss')" @click="$store.state.messages.lastRescanMessage = {}">
            <b-icon pack="mdi" icon="close" size="is-small"/>
          </button>
        </div>
        <div v-if="Object.keys(lastScrapeMessage).length !== 0" class="task-row">
          <span class="task-text">
            <span class="task-label" :class="{ 'is-running': lockScrape }">{{ $t('Data') }}</span>
            <span class="task-message" :title="lastScrapeMessage.message">{{ lastScrapeMessage.message }}</span>
          </span>
          <button v-if="lockScrape" class="task-stop" :disabled="stopping" @click="stopScrape">
            {{ stopping ? $t('Stopping…') : $t('Stop') }}
          </button>
          <button v-else class="task-dismiss" :aria-label="$t('Dismiss')" @click="$store.state.messages.lastScrapeMessage = {}">
            <b-icon pack="mdi" icon="close" size="is-small"/>
          </button>
        </div>
      </b-navbar-item>
      <b-navbar-item v-if="loginRequired" href="/logout" :title="$t('Log out')" class="logout-link">
        <b-icon pack="mdi" icon="logout" size="is-small"/>
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
      stopping: false,
      loginRequired: false
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
    ky.get('/api/options/apikey').json().then(d => { this.loginRequired = d.loginRequired }).catch(() => {})
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
/* stays within the navbar's height so the sticky list header below keeps its place */
.task-status {
  display: grid !important;
  grid-template-columns: minmax(0, 1fr) auto;
  align-content: center;
  align-items: center;
  column-gap: 8px;
  row-gap: 2px;
  width: min(440px, 38vw);
  max-height: 3.25rem;
  padding-top: 0.25rem !important;
  padding-bottom: 0.25rem !important;
  font-size: 0.75rem;
  line-height: 1.2;
}
.task-text {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.task-row {
  display: contents;
}
.task-label {
  flex-shrink: 0;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.task-label.is-running::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #7957d5;
  animation: task-pulse 1s ease-in-out infinite;
}
@keyframes task-pulse { 50% { opacity: 0.3; } }
.task-message {
  min-width: 0;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  opacity: 0.75;
}
.task-dismiss, .task-stop {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font: inherit;
}
.task-dismiss {
  width: 18px;
  height: 18px;
  opacity: 0.55;
}
.task-dismiss:hover {
  opacity: 1;
  background: rgba(127, 127, 127, 0.18);
}
.task-stop {
  padding: 0 7px;
  line-height: 1.35;
  color: #d93f5a;
  border: 1px solid currentColor;
}
.task-stop:hover:not(:disabled) {
  background: rgba(217, 63, 90, 0.12);
}
.logout-link {
  opacity: 0.6;
}
.logout-link:hover {
  opacity: 1;
}
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

</style>
