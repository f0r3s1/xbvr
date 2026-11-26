<template>
  <div class="modal is-active">
    <div class="modal-background" @click="$emit('close')"></div>
    <div class="modal-card split-modal">
      <section class="modal-card-body" :style="bodyStyle">
        <div class="columns">
          <div class="column is-half">
            <slot name="media"/>
          </div>
          <div class="column is-half">
            <slot name="details"/>
          </div>
        </div>
      </section>
      <slot name="footer"/>
    </div>
    <slot/>
  </div>
</template>

<script>
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'SplitModal',
  props: {
    bodyStyle: [Object, String]
  },
  emits: ['close']
})
</script>

<style>
.split-modal {
  width: 85%;
  margin: 0 auto;
  border-radius: 8px;
  overflow: hidden;
}
@media (min-width: 769px) {
  .split-modal {
    height: calc(100vh - 80px);
    max-height: 1100px;
  }
  .split-modal > .modal-card-body {
    height: 100%;
    overflow: hidden !important;
    padding-bottom: 12px;
  }
  .split-modal > .modal-card-body > .columns {
    height: 100%;
    margin-bottom: 0;
  }
  .split-modal > .modal-card-body > .columns > .column {
    max-height: 100%;
    padding-bottom: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
  .split-modal .tab-content {
    padding-left: 0;
    padding-right: 0;
    padding-bottom: 0;
  }
}
@media (min-width: 769px) {
  .split-modal .column.is-half:first-child {
    display: flex;
    flex-direction: column;
  }
  .split-modal .media-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
  .split-modal .media-tabs .tab-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 12px 0 0;
  }
  .split-modal .media-tabs .tab-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    width: 100%;
  }
  .split-modal .media-tabs .tab-item > .carousel-wrapper,
  .split-modal .media-tabs .tab-item > .carousel-wrapper > .carousel {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
  .split-modal .media-tabs .carousel-items {
    flex: 1;
    display: flex;
    align-items: stretch;
    min-height: 0;
  }
  .split-modal .media-tabs .carousel-item {
    height: 100%;
  }
  .split-modal .media-tabs .carousel-image-container.is-1by1 {
    height: 100%;
    padding-top: 0 !important;
  }
  .split-modal .media-tabs .carousel-indicator {
    margin-top: auto;
    padding-bottom: 0;
  }
  .split-modal .media-tabs .tab-item > .player-wrapper {
    margin: auto 0;
  }
}
.split-modal .media-tabs .carousel {
  background: transparent !important;
}
.split-modal .media-tabs .carousel-items,
.split-modal .media-tabs .carousel-item {
  background: #f2f2f5 !important;
  border-radius: 0.5rem;
}
html[data-theme="dark"] .split-modal .media-tabs .carousel-items,
html[data-theme="dark"] .split-modal .media-tabs .carousel-item {
  background: #222230 !important;
}
.split-modal .media-tabs .carousel-image {
  border-radius: 8px;
}
.split-modal .media-tabs .carousel-indicator {
  flex-shrink: 0;
  scrollbar-width: none;
}
.split-modal .media-tabs .carousel-indicator::-webkit-scrollbar {
  display: none;
}
</style>
