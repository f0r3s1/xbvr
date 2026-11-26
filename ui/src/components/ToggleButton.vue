<template>
  <a :class="['button', 'is-small', `is-${color}`, { 'is-outlined': !active }]"
     v-bind="{ [nativeTitle ? 'title' : 'data-tooltip']: $t(active ? onTooltip : offTooltip) }"
     @click="toggle">
    <b-icon pack="mdi" :icon="active && activeIcon ? activeIcon : icon" size="is-small"/>
  </a>
</template>

<script>
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'ToggleButton',
  props: {
    item: { type: Object, required: true },
    field: { type: String, required: true },
    list: String,
    actor: Boolean,
    color: { type: String, default: 'dark' },
    icon: { type: String, required: true },
    activeIcon: String,
    onTooltip: { type: String, required: true },
    offTooltip: { type: String, required: true },
    nativeTitle: Boolean
  },

  computed: {
    active () {
      return !!this.item[this.field]
    }
  },

  methods: {
    toggle () {
      const next = !this.active
      const list = this.list || this.field
      if (this.actor) {
        this.$store.commit('actorList/toggleActorList', { actor_id: this.item.id, list })
      } else {
        this.$store.commit('sceneList/toggleSceneList', { scene_id: this.item.scene_id, list })
      }
      this.item[this.field] = next
    }
  }
})
</script>
