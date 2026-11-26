<template>
  <ToggleButton :item="item" field="is_hidden" color="danger" icon="eye-outline" active-icon="eye-off"
    on-tooltip="Unhide" off-tooltip="Mark as hidden"/>
</template>

<script>
import { defineComponent } from 'vue'
import ToggleButton from './ToggleButton.vue'

export default defineComponent({
  name: 'HiddenButton',
  components: { ToggleButton },
  props: { item: Object }
})
</script>
