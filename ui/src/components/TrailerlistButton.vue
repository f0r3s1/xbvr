<template>
  <a :class="buttonClass"
   v-if="!item.is_available & item.trailer_source !=='' & this.$store.state.optionsWeb.web.sceneTrailerlist"
     @click="toggleState()"
     :data-tooltip="item.trailerlist ? 'Remove from Trailer List' : 'Add to Trailer List'">
    <b-icon pack="mdi" :icon="item.trailerlist ? 'movie-open-check' : 'movie-search-outline'" size="is-small"/>
  </a>
</template>

<script>
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'TrailerlistButton',
  props: { item: Object },

  computed: {
    buttonClass () {
      if (this.item.trailerlist) {
        return 'button is-primary is-small'
      }
      return 'button is-primary is-outlined is-small'
    }
  },

  methods: {
    toggleState() {
      let currentToggle=this.item.trailerlist
      this.$store.commit('sceneList/toggleSceneList', {scene_id: this.item.scene_id, list: 'trailerlist'})
      this.item.trailerlist=!currentToggle
    }
  },
});
</script>
