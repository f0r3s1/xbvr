<template>
  <OptionsSection :title="$t('Export funscripts')" :loading="isLoading" :value="options" @save="save">
    <div class="content">
      <p>{{ $t('Here you can download a ZIP file containing a funscript for each scripted scene. The file names include scene title and scene id, as expected by DeoVR. If a scene has multiple scripts you can choose a preferred script in the scene details view. Otherwise, the most recently added script is chosen.') }}</p>
      <p>{{ $t('Note that the filenames are not compatible with DLNA.') }}</p>
      <p>{{ $t('To use this export with DeoVR: Unzip and put the files in the Interactive folder on your device.') }}</p>
      <p>{{ $t('To use this export with ScriptPlayer: Unzip and put the files in a folder of your choice. In the ScriptPlayer settings, add this folder in the Paths section, then connect to DeoVR.') }}</p>
    </div>

    <div class="buttons">
      <b-button tag="a" href="/api/task/funscript/export-all" type="is-primary" icon-left="download" :disabled="countTotal === 0">
        {{ $t('Download all funscripts') }} ({{ countTotal }})
      </b-button>
      <b-button tag="a" href="/api/task/funscript/export-new" type="is-primary" outlined icon-left="download" :disabled="countUpdated === 0">
        {{ $t('Download changes since last export') }} ({{ countUpdated }})
      </b-button>
    </div>

    <b-field>
      <b-switch v-model="options.scrapeFunscripts">{{ $t('Scrape for available funscripts') }}</b-switch>
    </b-field>
  </OptionsSection>
</template>

<script>
import { defineComponent } from 'vue'
import OptionsSection from '../OptionsSection.vue'

export default defineComponent({
  name: 'Funscripts',
  components: { OptionsSection },

  data () {
    return { isLoading: true }
  },

  async mounted () {
    await this.$store.dispatch('optionsFunscripts/load')
    this.isLoading = false
  },

  computed: {
    options () {
      return this.$store.state.optionsFunscripts.optionsFunscripts
    },
    countTotal () {
      return this.$store.state.optionsFunscripts.countTotal
    },
    countUpdated () {
      return this.$store.state.optionsFunscripts.countUpdated
    }
  },

  methods: {
    save () {
      return this.$store.dispatch('optionsFunscripts/save')
    }
  }
})
</script>
