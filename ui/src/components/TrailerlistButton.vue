<template>
  <ToggleButton v-if="!item.is_available && item.trailer_source !== '' && $store.state.optionsWeb.web.sceneTrailerlist" :item="item" field="trailerlist" color="primary" icon="movie-search-outline" active-icon="movie-open-check"
    on-tooltip="Remove from Trailer List" off-tooltip="Add to Trailer List"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'TrailerlistButton',
  components: { ToggleButton },
  props: { item: Object }
})
</script>
