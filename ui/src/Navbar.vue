<template>
  <b-navbar :fixed-top="true" type="is-light">
    <template #brand>
      <b-navbar-item>
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
            <td>{{lastRescanMessage.message}}</td>
          </tr>
          <tr v-if="Object.keys(lastScrapeMessage).length !== 0">
            <th><span :class="[lockScrape ? 'pulsate' : '']">{{$t('Data')}} →</span></th>
            <td>{{lastScrapeMessage.message}}</td>
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
      latestVersion: ''
    }
  },

  computed: {
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
