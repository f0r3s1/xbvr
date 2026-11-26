<template>
  <ToggleButton :item="actor" actor native-title field="favourite" color="danger" icon="heart-outline" active-icon="heart"
    on-tooltip="Remove from favourites" off-tooltip="Add to favourites"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'ActorFavouriteButton',
  components: { ToggleButton },
  props: { actor: Object }
})
</script>
