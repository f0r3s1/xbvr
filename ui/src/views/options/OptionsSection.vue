<template>
  <section class="options-section">
    <b-loading :is-full-page="false" :model-value="loading"/>
    <header class="options-section-head">
      <h3 class="options-section-title">{{ title }}</h3>
      <div class="options-section-actions"><slot name="actions"/></div>
    </header>
    <p v-if="description" class="options-section-desc">{{ description }}</p>
    <slot :dirty="dirty"/>
    <footer v-if="onSave" class="options-section-save">
      <b-button type="is-primary" :disabled="!dirty" :loading="saving" @click="save">{{ saveLabel || $t('Save') }}</b-button>
      <slot name="footer"/>
      <span v-if="savedAt && !dirty" class="options-section-note">{{ savedNote || $t('Saved') }}</span>
    </footer>
  </section>
</template>

<script>
import { defineComponent } from 'vue'

// Save is enabled only while `value` differs from what was last loaded or saved.
export default defineComponent({
  name: 'OptionsSection',
  props: {
    title: { type: String, required: true },
    description: { type: String, default: '' },
    loading: { type: Boolean, default: false },
    value: { type: [Object, Array], default: null },
    saveLabel: { type: String, default: '' },
    savedNote: { type: String, default: '' },
    onSave: { type: Function, default: null }
  },

  data () {
    return { snapshot: null, saving: false, savedAt: null }
  },

  computed: {
    current () {
      return JSON.stringify(this.value)
    },
    dirty () {
      return this.value === null || (this.snapshot !== null && this.current !== this.snapshot)
    }
  },

  watch: {
    loading: {
      handler (loading) {
        if (!loading) this.snapshot = this.current
      },
      immediate: true
    }
  },

  methods: {
    async save () {
      this.saving = true
      try {
        await this.onSave()
        this.snapshot = this.current
        this.savedAt = Date.now()
      } finally {
        this.saving = false
      }
    }
  }
})
</script>

<style>
.options-section {
  position: relative;
  min-height: 120px;
}
.options-section-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  padding-bottom: 0.6rem;
  margin-bottom: 1rem;
  border-bottom: 1px solid rgba(127, 127, 127, 0.2);
}
.options-section-title {
  flex: 1;
  margin: 0 !important;
  font-size: 1.25rem;
  font-weight: 600;
}
.options-section-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}
.options-section-desc {
  margin: -0.4rem 0 1rem;
  font-size: 0.85rem;
  opacity: 0.7;
}
.options-section-save {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1.25rem;
  padding-top: 0.75rem;
  border-top: 1px solid rgba(127, 127, 127, 0.15);
}
.options-section .options-subtitle {
  margin: 1.5rem 0 0.6rem !important;
  font-size: 1rem;
  font-weight: 600;
}
.options-section .options-subtitle:first-child {
  margin-top: 0 !important;
}
.options-section-note {
  font-size: 0.8rem;
  opacity: 0.6;
}
</style>
