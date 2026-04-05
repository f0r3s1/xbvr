<template>
  <a class="button is-dark is-outlined is-small"
    @click="editScene(item)"    
    :data-tooltip="item.id == 0 ? 'Display scene details (E)': 'Edit scene details (E)'">
    <b-icon pack="mdi" icon="lead-pencil" size="is-small" />
  </a>
</template>

<script>
export default {
  name: 'EditButton',
  props: { item: Object },
  methods: {
    editScene (scene) {
      this.$store.commit('overlay/editDetails', { scene })
    }
  }
}
</script>
