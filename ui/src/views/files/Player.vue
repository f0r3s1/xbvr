<template>
  <div class="modal is-active">
    <GlobalEvents @keyup.esc="close"/>
    <div class="modal-background" @click="close"></div>
    <div class="modal-content">
      <VrPlayer ref="player" aspect-ratio="1 / 1" :loop="false"/>
    </div>
    <button class="modal-close is-large" :aria-label="$t('close')"
            @click="close()"></button>
  </div>
</template>

<script>
import { defineComponent } from 'vue';
import { GlobalEvents } from 'vue-global-events'
import VrPlayer from '../../components/VrPlayer.vue'

export default defineComponent({
  name: 'FilePlayer',
  components: { VrPlayer, GlobalEvents },

  mounted () {
    const file = this.$store.state.overlay.player.file
    if (!file) return
    this.$refs.player.load('/api/dms/file/' + file.id + '?dnt=true', file.projection)
    this.$refs.player.play()
  },

  methods: {
    close () {
      this.$store.commit('overlay/hidePlayer')
    }
  },
});
</script>

<style scoped>
  .modal-content {
    display: flex;
    justify-content: center;
    align-items: center;
  }
  .modal-content :deep(.xbvr-skin) {
    width: min(640px, 90vw);
  }
</style>
