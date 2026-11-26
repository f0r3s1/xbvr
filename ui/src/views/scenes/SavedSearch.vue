<template>
  <SharedSavedSearch/>
</template>

<script>
import { defineComponent } from 'vue'
import SharedSavedSearch from '../../components/SavedSearch.vue'

export default defineComponent({
  name: 'SceneSavedSearch',
  components: { SharedSavedSearch }
})
</script>
