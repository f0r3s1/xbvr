<template>
  <span></span>
</template>

<script>
import { defineComponent } from 'vue';

import { Wampy } from 'wampy'

export default defineComponent({
  name: 'Socket',

  data () {
    return {
      wsStatus: ''
    }
  },

  mounted () {
    this._connectWs()
  },

  methods: {
    async _connectWs () {
      const ws = new Wampy('/ws/', {
        realm: 'default',
        autoReconnect: true,
        reconnectInterval: 2000,
        maxRetries: 0,
        onClose: () => { this.wsStatus = 'disconnected' },
        onReconnect: () => { this.wsStatus = 'connecting' },
        onReconnectSuccess: async () => {
          this.wsStatus = 'connected'
          await this._subscribe(ws)
        },
      })

      // Retry initial connection until Go backend is ready
      while (true) {
        try {
          await ws.connect()
          this.wsStatus = 'connected'
          break
        } catch {
          this.wsStatus = 'disconnected'
          await new Promise(r => setTimeout(r, 2000))
        }
      }

      await this._subscribe(ws)
    },

    async _subscribe (ws) {
      await ws.subscribe('service.log', (eventData) => {
        if (eventData.argsDict.level == 'debug') {
          console.debug(eventData.argsDict.message)
        }
        if (eventData.argsDict.level == 'info') {
          console.info(eventData.argsDict.message)
        }
        if (eventData.argsDict.level == 'error') {
          console.error(eventData.argsDict.message)
        }

        if (eventData.argsDict.data.task === 'scrape') {
          this.$store.state.messages.lastScrapeMessage = eventData.argsDict
        }

        if (eventData.argsDict.data.task === 'scraperProgress') {
          if (eventData.argsDict.message === 'DONE') {
            this.$store.state.messages.runningScrapers = []
          }
          if (eventData.argsDict.data.started) {
            this.$store.state.messages.runningScrapers.push(eventData.argsDict.data.scraperID)
          }
          if (eventData.argsDict.data.completed) {
            this.$store.state.messages.runningScrapers.splice(this.$store.state.messages.runningScrapers.indexOf(eventData.argsDict.data.scraperID), 1)
          }
        }

        if (eventData.argsDict.data.task === 'rescan') {
          this.$store.state.messages.lastRescanMessage = eventData.argsDict
        }
      })

      await ws.subscribe('lock.change', (eventData) => {
        if (eventData.argsDict.name === 'scrape') {
          this.$store.state.messages.lockScrape = eventData.argsDict.locked
        }
        if (eventData.argsDict.name === 'rescan') {
          this.$store.state.messages.lockRescan = eventData.argsDict.locked
        }
      })

      await ws.subscribe('state.change.optionsStorage', () => {
        this.$store.dispatch('optionsStorage/load')
      })

      await ws.subscribe('options.previews.previewReady', (eventData) => {
        this.$store.commit('optionsPreviews/showPreview', { previewFn: eventData.argsDict.previewFn })
      })

      await ws.subscribe('health.progress', (eventData) => {
        this.$store.commit('health/setProgress', eventData.argsDict)
      })

      await ws.subscribe('remote.state', (eventData) => {
        this.$store.dispatch('remote/processMessage', eventData.argsDict)
      })
    }
  }
});
</script>

<style scoped>
</style>
