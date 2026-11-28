<template>
  <b-dropdown position="is-bottom-left" append-to-body aria-role="list">
    <template #trigger>
      <a class="button is-dark is-outlined is-small" :data-tooltip="$t('Regenerate preview')">
        <b-icon pack="mdi" icon="movie-open-cog-outline" size="is-small"/>
      </a>
    </template>
    <b-dropdown-item v-for="m in methods" :key="m.id" aria-role="listitem" @click="regenerate(m.id)">
      {{ m.label }}<span v-if="item.preview_method === m.id" class="current"> · {{ $t('current') }}</span>
    </b-dropdown-item>
  </b-dropdown>
</template>

<script>
import ky from 'ky'

export default {
  name: 'RegeneratePreviewButton',
  props: { item: { type: Object, required: true } },
  computed: {
    methods () {
      return [
        { id: 'classic', label: this.$t('Classic: one eye, cropped') },
        { id: 'flat', label: this.$t('Flat: undistorted trailer, true colours') },
        { id: 'smart', label: this.$t('Smart: flat trailer framed on the people') }
      ]
    }
  },
  methods: {
    async regenerate (method) {
      await ky.post('/api/task/preview/regenerate', { json: { scene_id: this.item.id, method } })
    }
  }
}
</script>

<style scoped>
.current {
  opacity: 0.6;
}
</style>
