<template>
  <ToggleButton :item="item" field="favourite" color="danger" icon="heart-outline" active-icon="heart"
    on-tooltip="Remove from favourites" off-tooltip="Add to favourites"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'FavouriteButton',
  components: { ToggleButton },
  props: { item: Object }
})
</script>
