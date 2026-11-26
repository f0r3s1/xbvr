<template>
  <ToggleButton v-if="!item.is_available" :item="item" field="wishlist" color="info" icon="oil-lamp"
    on-tooltip="Remove from wishlist" off-tooltip="Add to wishlist"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'WishlistButton',
  components: { ToggleButton },
  props: { item: Object }
})
</script>
