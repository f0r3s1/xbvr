<template>
  <div class="modal is-active">
    <div class="modal-background" @click="close"></div>
    <div class="modal-content">
      <video ref="player"
             width="640" height="640" class="video-js vjs-default-skin"
             controls playsinline autoplay>
        <source :src="sourceUrl" type="video/mp4">
      </video>
    </div>
    <button class="modal-close is-large" aria-label="close"
            @click="close()"></button>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

import videojs from 'video.js'
import 'videojs-vr'
import 'videojs-hotkeys'

export default defineComponent({
  name: 'FilePlayer',

  data () {
    return {
      player: {}
    }
  },

  computed: {
    sourceUrl () {
      if (this.$store.state.overlay.player.file) {
        return '/api/dms/file/' + this.$store.state.overlay.player.file.id + '?dnt=true'
      }
      return ''
    }
  },

  mounted () {
    this.player = videojs(this.$refs.player)
    const vr = this.player.vr({
      projection: this.$store.state.overlay.player.file.projection == 'flat' ? 'NONE' : '180',
      forceCardboard: false
    })

    this.player.hotkeys({
      alwaysCaptureHotkeys: true,
      volumeStep: 0.1,
      seekStep: 5,
      enableModifiersForNumbers: false,
      customKeys: {
        closeModal: {
          key: function (event) {
            return event.which === 27
          },
          handler: () => {
            this.player.dispose()
            this.$store.commit('overlay/hidePlayer')
          }
        }
      }
    })

    this.player.on('loadedmetadata', function () {
      if (vr && vr.camera) vr.camera.position.set(-1, 0, -1)
    })
  },

  methods: {
    close () {
      this.player.dispose()
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
  .video-js {
    display: block;
    margin: 0 auto;
  }
</style>
