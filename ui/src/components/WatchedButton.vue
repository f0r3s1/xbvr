<template>
  <a :class="buttonClass"
     @click="toggleState()"
     :data-tooltip="item.is_watched ? 'Mark as unwatched' : 'Mark as watched'">
    <b-icon pack="mdi" :icon="item.is_watched ? 'eye-check' : 'eye'" size="is-small"/>
  </a>
</template>

<script>
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'WatchedButton',
  props: { item: Object },

  computed: {
    buttonClass () {
      if (this.item.is_watched) {
        return 'button is-dark is-small'
      }
      return 'button is-dark is-outlined is-small'
    }
  },

  methods: {
    toggleState() {
      let currentToggle=this.item.is_watched
      console.log("watched toggleState", this.item.is_watched)
      this.$store.commit('sceneList/toggleSceneList', {scene_id: this.item.scene_id, list: 'watched'})
      this.item.is_watched=!currentToggle
      console.log("watched toggleState", this.item.is_watched)
    }
  },
});
</script>
