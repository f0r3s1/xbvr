<template>
  <OptionsSection :title="$t('Cache')" :loading="isLoading">
    <div class="settings-list" v-if="!isLoading">
      <div class="settings-row">
        <div class="row-text">
          <strong>Images</strong>
          <p>Cache of remote images that were requested at least once. Least recently used images are removed above the maximum size (0 = unlimited).</p>
        </div>
        <span class="row-size">{{ prettyBytes(sizes.images) }}</span>
        <div class="row-actions">
          <b-field class="max-size" :title="'Max size in GB'">
            <b-input v-model.number="imageMaxGB" type="number" min="0" size="is-small" @blur="saveImageMaxGB" @keyup.enter="saveImageMaxGB" />
            <p class="control"><span class="button is-static is-small">GB</span></p>
          </b-field>
          <b-button size="is-small" @click="resetCache('images')">Reset</b-button>
        </div>
      </div>
      <div class="settings-row">
        <div class="row-text">
          <strong>Video previews</strong>
          <p>Generated on demand for local files. Remove them to regenerate previews with new settings.</p>
        </div>
        <span class="row-size">{{ prettyBytes(sizes.previews) }}</span>
        <div class="row-actions">
          <b-button size="is-small" @click="resetCache('previews')">Reset</b-button>
        </div>
      </div>
      <div class="settings-row">
        <div class="row-text">
          <strong>Search index</strong>
          <span class="row-note">{{ searchInprogress ? 'Indexing…' : indexSceneCount + ' scenes indexed' }}</span>
          <p>Remove the search index when finding or matching files misbehaves.</p>
        </div>
        <span class="row-size">{{ prettyBytes(sizes.searchIndex) }}</span>
        <div class="row-actions">
          <b-button size="is-small" @click="resetCache('searchIndex')">Reset</b-button>
          <b-button size="is-small" @click="indexRescan">Rescan</b-button>
        </div>
      </div>
      <div class="settings-row">
        <div class="row-text">
          <strong>Scene status</strong>
          <p>Refresh scenes that aren't marked available or scripted despite having such files.</p>
        </div>
        <span class="row-size"></span>
        <div class="row-actions">
          <b-button size="is-small" @click="taskRefresh">Refresh</b-button>
        </div>
      </div>
      <div class="settings-row">
        <div class="row-text">
          <strong>Fix inconsistencies</strong>
          <p>Repairs files matched to deleted or missing scenes, and scenes whose size or availability no longer matches their files.</p>
        </div>
        <span class="row-size">{{ fixStatusText }}</span>
        <div class="row-actions">
          <b-button size="is-small" :loading="fixRunning" :disabled="fixRunning" @click="fixInconsistencies">Fix all</b-button>
        </div>
      </div>
    </div>
  </OptionsSection>
</template>

<script>
import { defineComponent } from 'vue';

import ky from 'ky'
import { getOptionsState } from '@/api'
import prettyBytes from 'pretty-bytes'
import OptionsSection from '../OptionsSection.vue'

export default defineComponent({
  name: 'Cache',
  components: { OptionsSection },

  data () {
    return {
      isLoading: true,
      sizes: {},
      imageMaxGB: 10,
      indexSceneCount: 0,
      searchInprogress: false,
      fixRunning: false,
      fixPhase: 'idle',
      fixDone: 0,
      fixTotal: 0,
      fixResult: null,
      fixPoll: null,
    }
  },
  computed: {
    fixStatusText () {
      if (this.fixRunning && this.fixPhase === 'scanning') return 'Scanning…'
      if (this.fixRunning && this.fixPhase === 'fixing') return `Fixing ${this.fixDone}/${this.fixTotal}…`
      if (this.fixResult) {
        let t = `Fixed ${this.fixResult.fixed} of ${this.fixResult.scanned}`
        if (this.fixResult.failed) t += `, ${this.fixResult.failed} failed`
        return t
      }
      return ''
    }
  },

  async mounted () {
    await this.loadState()
    this.loadSearchState()
    this.pollFixStatus()
  },
  beforeUnmount () {
    if (this.fixPoll) clearInterval(this.fixPoll)
  },

  methods: {
    async loadState () {
      this.isLoading = true
      await getOptionsState()
        .then(data => {
          this.sizes = data.currentState.cacheSize
          this.imageMaxGB = data.config.library.imageCacheMaxGB
          this.isLoading = false
        })
    },
    async saveImageMaxGB () {
      const maxGB = Math.max(0, Math.floor(Number(this.imageMaxGB) || 0))
      this.imageMaxGB = maxGB
      await ky.put('/api/options/cache/image-max-gb', { json: { maxGB } })
    },
    async resetCache (kind) {
      this.isLoading = true
      await ky.delete(`/api/options/cache/reset/${kind}`, { timeout: 30000 })
      await this.loadState()
      await this.loadSearchState()
    },
    taskRefresh: function () {
      ky.get('/api/task/scene-refresh')
    },
    fixInconsistencies () {
      this.$buefy.dialog.confirm({
        title: 'Fix inconsistencies',
        message: 'Scan the library and apply all fixes (re-match/unmatch orphaned files, reset stale scene status)?',
        confirmText: 'Fix All',
        type: 'is-warning',
        onConfirm: async () => {
          await ky.post('/api/inconsistencies/fixall', { json: {} })
          this.fixRunning = true
          this.fixPhase = 'scanning'
          this.fixResult = null
          this.pollFixStatus()
        }
      })
    },
    async pollFixStatus () {
      const apply = st => {
        this.fixRunning = st.running
        this.fixPhase = st.phase
        this.fixDone = st.done
        this.fixTotal = st.total
        if (st.result) this.fixResult = st.result
      }
      apply(await ky.get('/api/inconsistencies/status').json())
      if (this.fixPoll) { clearInterval(this.fixPoll); this.fixPoll = null }
      if (this.fixRunning) {
        this.fixPoll = setInterval(async () => {
          const st = await ky.get('/api/inconsistencies/status').json()
          apply(st)
          if (!st.running) {
            clearInterval(this.fixPoll); this.fixPoll = null
            await this.loadState()
          }
        }, 1000)
      }
    },
    async loadSearchState () {
      this.isLoading = true
      await ky.get('/api/options/state/search')
        .json()
        .then(data => {
          this.indexSceneCount = data.documentCount
          this.searchInprogress = data.inProgress
          this.isLoading = false
        })
    },
    async indexRescan () {
      this.isLoading = true
      await ky.get('/api/task/index')
      this.searchInprogress = true
      this.isLoading = false
    },
    prettyBytes
  },
});
</script>

<style scoped>
.settings-list {
  border: 1px solid rgba(127, 127, 127, 0.25);
  border-radius: 8px;
  overflow: hidden;
}
.settings-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 6rem auto;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
}
.settings-row + .settings-row {
  border-top: 1px solid rgba(127, 127, 127, 0.25);
}
.row-text p {
  margin: 0.15rem 0 0;
  font-size: 0.8rem;
  opacity: 0.65;
}
.row-note {
  margin-left: 0.5rem;
  font-size: 0.75rem;
  opacity: 0.6;
}
.row-size {
  text-align: right;
  font-size: 0.85rem;
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}
.max-size {
  width: 6rem;
  margin: 0;
}
.row-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.4rem;
}
@media (max-width: 600px) {
  .settings-row {
    grid-template-columns: minmax(0, 1fr) auto;
  }
  .row-size {
    display: none;
  }
}
</style>
