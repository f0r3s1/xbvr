<template>
  <a class="button is-dark is-outlined is-small"
    @click="editScene(item)"    
    :data-tooltip="item.id == 0 ? 'Display scene details': 'Edit scene details'">
    <b-icon pack="mdi" icon="lead-pencil" size="is-small" />
  </a>
</template>

<script>
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'EditButton',
  props: { item: Object },

  methods: {
    editScene (scene) {
      this.$store.commit('overlay/editDetails', { scene })
    }
  },
});
</script>
