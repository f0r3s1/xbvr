<template>
  <ToggleButton :item="actor" actor native-title field="watchlist" color="primary" icon="calendar-blank" active-icon="calendar-check"
    on-tooltip="Remove from watchlist" off-tooltip="Add to watchlist"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'ActorWatchlistButton',
  components: { ToggleButton },
  props: { actor: Object }
})
</script>
