<template>
  <ToggleButton :item="item" field="needs_update" icon="refresh"
    on-tooltip="Do not refresh scene" off-tooltip="Refresh scene on next scrape"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'RefreshButton',
  components: { ToggleButton },
  props: { item: Object }
})
</script>
