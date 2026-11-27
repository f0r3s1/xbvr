<template>
  <OptionsSection :title="$t('Previews')" :loading="isLoading" :value="preview" :save-label="$t('Save settings')" @save="saveSettings">
    <div class="method-row">
      <b-field :label="$t('Method for new previews')">
        <b-select v-model="preview.method" size="is-small">
          <option value="classic">{{ $t('Classic: one eye, cropped') }}</option>
          <option value="flat">{{ $t('Flat: undistorted trailer, true colours') }}</option>
          <option value="smart">{{ $t('Smart: flat trailer framed on the people') }}</option>
        </b-select>
      </b-field>
      <b-field v-if="preview.method !== 'classic'" :label="$t('Video format')">
        <b-select v-model="preview.codec" size="is-small">
          <option value="h265">{{ $t('H.265 (smaller)') }}</option>
          <option value="h264">{{ $t('H.264 (plays everywhere)') }}</option>
        </b-select>
      </b-field>
    </div>
    <p class="method-hint">{{ $t('Existing previews are kept. To remake one with another method, use Regenerate preview in the scene details.') }}</p>
    <div class="columns">
      <div class="column">
        <b-field v-for="s in visibleSliders" :key="s.key" :label="s.label" class="slider-row">
          <b-slider :min="s.min" :max="s.max" :step="s.step" :tooltip="false" v-model="preview[s.key]"></b-slider>
          <span class="slider-value">{{ preview[s.key] }}{{ s.unit }}</span>
        </b-field>
        <b-field v-if="preview.method === 'classic'">
          <b-checkbox v-model="preview.extraSnippet">{{ $t('Grab extra snippet from the end of video') }}</b-checkbox>
        </b-field>
      </div>
      <div class="column">
        <video v-if="isPreviewReady" :src="`/api/dms/preview/${previewFn}`" autoplay loop muted></video>
        <div v-else-if="generatingPreview" class="preview-pending">
          <b-icon pack="mdi" icon="sync" custom-class="mdi-spin"></b-icon>
        </div>
      </div>
    </div>

    <template #footer>
      <b-button @click="testSettings">{{ $t('Test settings') }}</b-button>
    </template>

    <div class="generate">
      <p>{{ $t('Once you picked preview settings, start generating them. This is a CPU-heavy process; once started it can only be stopped by closing the app.') }}</p>
      <b-button type="is-primary" outlined @click="startGenerating">{{ $t('Start generating previews') }}</b-button>
    </div>
  </OptionsSection>
</template>

<script>
import { defineComponent } from 'vue'
import ky from 'ky'
import { getOptionsState } from '@/api'
import OptionsSection from '../OptionsSection.vue'

export default defineComponent({
  name: 'Previews',
  components: { OptionsSection },

  data () {
    return {
      isLoading: true,
      preview: { startTime: 5, snippetLength: 0.2, snippetAmount: 2, resolution: 300, extraSnippet: false, method: 'classic', codec: 'h265' },
      sliders: [
        { key: 'startTime', label: this.$t('Start time'), min: 5, max: 60, step: 5, unit: 's' },
        { key: 'snippetLength', label: this.$t('Snippet length'), min: 0.2, max: 5, step: 0.2, unit: 's' },
        { key: 'snippetAmount', label: this.$t('Number of snippets'), min: 2, max: 40, step: 1, unit: '' },
        { key: 'resolution', label: this.$t('Preview resolution'), min: 300, max: 800, step: 20, unit: 'px' }
      ]
    }
  },

  async mounted () {
    const data = await getOptionsState()
    const p = data.config.library.preview
    this.preview = { startTime: p.startTime, snippetLength: p.snippetLength, snippetAmount: p.snippetAmount, resolution: p.resolution, extraSnippet: p.extraSnippet, method: p.method || 'classic', codec: p.codec || 'h265' }
    this.isLoading = false
  },

  computed: {
    // the flat trailer has its own fixed moments, so only the start time and resolution apply to it
    visibleSliders () {
      return this.preview.method !== 'classic' ? this.sliders.filter(s => s.key === 'startTime' || s.key === 'resolution') : this.sliders
    },
    generatingPreview () {
      return this.$store.state.optionsPreviews.generatingPreview
    },
    isPreviewReady () {
      return this.$store.state.optionsPreviews.isPreviewReady
    },
    previewFn () {
      return this.$store.state.optionsPreviews.previewFn
    }
  },

  methods: {
    saveSettings () {
      return ky.put('/api/options/previews', { json: this.preview }).json()
    },
    testSettings () {
      this.$store.commit('optionsPreviews/hidePreview')
      return ky.post('/api/options/previews/test', { json: this.preview })
    },
    startGenerating () {
      return ky.get('/api/task/preview/generate')
    }
  }
})
</script>

<style scoped>
.slider-row :deep(.field-body > .field),
.slider-row :deep(> .control) {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.slider-row :deep(.b-slider) {
  flex: 1;
}
.slider-value {
  min-width: 4em;
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}
video {
  width: 100%;
  border-radius: 8px;
}
.preview-pending {
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 1;
  max-width: 240px;
  border-radius: 8px;
  background: rgba(127, 127, 127, 0.12);
}
.generate {
  margin-top: 1.5rem;
}
.method-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1.5rem;
}
.method-hint {
  margin: 0.25rem 0 1.25rem;
  font-size: 0.8rem;
  opacity: 0.65;
}
.generate p {
  margin-bottom: 0.5rem;
  font-size: 0.85rem;
  opacity: 0.75;
}
</style>
