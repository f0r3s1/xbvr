<template>
  <SharedSavedSearch actors/>
</template>

<script>
import { defineComponent } from 'vue'
import SharedSavedSearch from '../../components/SavedSearch.vue'

export default defineComponent({
  name: 'ActorSavedSearch',
  components: { SharedSavedSearch }
})
</script>
