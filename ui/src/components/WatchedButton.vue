<template>
  <ToggleButton :item="item" field="is_watched" list="watched" icon="eye" active-icon="eye-check"
    on-tooltip="Mark as unwatched" off-tooltip="Mark as watched"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'WatchedButton',
  components: { ToggleButton },
  props: { item: Object }
})
</script>
