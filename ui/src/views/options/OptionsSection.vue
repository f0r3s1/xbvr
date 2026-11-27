<template>
  <section class="options-section">
    <b-loading :is-full-page="false" :model-value="loading && !loadedOnce"/>
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
    return { snapshot: null, saving: false, savedAt: null, loadedOnce: false }
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
      handler (loading, was) {
        if (loading) return
        this.snapshot = this.current
        if (was) this.loadedOnce = true // later saves reload quietly instead of covering the form
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

/* one look for every settings page, matching the Web UI page */
.options-section {
  font-size: 0.875rem;
}
.options-section .options-subtitle,
.options-section .group-title,
.options-section .title,
.options-section .subtitle {
  font-size: 0.95rem !important;
  font-weight: 600 !important;
}
.options-section .label {
  font-size: 0.8rem;
  font-weight: 600;
}
.options-section .input,
.options-section .textarea,
.options-section .select select,
.options-section .taginput .taginput-container {
  font-size: 0.875rem;
  border-radius: 8px;
}
.options-section .input,
.options-section .select select {
  height: 2.4em;
}
.options-section .switch {
  font-size: 0.875rem;
}
.options-section .button:not(.hover-actions *) {
  border-radius: 8px;
  font-size: 0.85rem;
}
.options-section .message,
.options-section .notification {
  font-size: 0.85rem;
  border-radius: 10px;
}
.options-section .tag {
  border-radius: 6px;
}
.options-section .tabs {
  font-size: 0.875rem;
}
.options-section .field.has-addons:has(.button):not(:has(.input)) {
  gap: 0.5rem;
  flex-wrap: wrap;
}
.options-section .field.has-addons:has(.button):not(:has(.input)) .button {
  border-radius: 8px !important;
}
.options-section .b-tooltip:has(> .tooltip-trigger .field) {
  display: block;
}
.options-section .b-tooltip:has(> .tooltip-trigger .field) + .b-tooltip {
  margin-top: 1.25rem;
}
.options-section .table th .input {
  padding-left: 2em;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2.2'%3E%3Ccircle cx='11' cy='11' r='7'/%3E%3Cpath d='m20 20-3.5-3.5'/%3E%3C/svg%3E") !important;
  background-repeat: no-repeat !important;
  background-position: 0.65em center !important;
}
.options-section .table .tag.is-medium {
  font-size: 0.8rem;
  min-width: 2.2em;
  background: rgba(128, 128, 160, 0.14) !important;
  color: inherit !important;
}
/* buttons: solid accent for the main action, tinted for colored ones, soft fill for the rest; the repeated class outranks the global theme, table row actions (App.vue) outrank these */
.options-section .button:not(.hover-actions *) {
  font-weight: 500;
  transition: background-color 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}
.options-section.options-section.options-section .button:not(.hover-actions *):where(:not([class*="is-primary"], .is-danger, .is-info, .is-success, .is-warning, .is-text)) {
  background-color: rgba(128, 128, 160, 0.1) !important;
  border: 1px solid rgba(128, 128, 160, 0.18) !important;
  color: inherit !important;
}
.options-section.options-section.options-section .button:not(.hover-actions *):where(:not([class*="is-primary"], .is-danger, .is-info, .is-success, .is-warning, .is-text)):hover {
  background-color: rgba(128, 128, 160, 0.18) !important;
}
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary:not(.is-light):not(.is-outlined):hover {
  filter: brightness(1.12);
}
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary.is-light,
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary.is-outlined {
  --tint: 124, 108, 240;
}
.options-section.options-section.options-section .button:not(.hover-actions *).is-danger { --tint: 241, 70, 104; }
.options-section.options-section.options-section .button:not(.hover-actions *).is-info { --tint: 96, 165, 250; }
.options-section.options-section.options-section .button:not(.hover-actions *).is-success { --tint: 52, 211, 153; }
.options-section.options-section.options-section .button:not(.hover-actions *).is-warning { --tint: 234, 179, 8; }
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary.is-light,
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary.is-outlined,
.options-section.options-section.options-section .button:not(.hover-actions *).is-danger,
.options-section.options-section.options-section .button:not(.hover-actions *).is-info,
.options-section.options-section.options-section .button:not(.hover-actions *).is-success,
.options-section.options-section.options-section .button:not(.hover-actions *).is-warning {
  background-color: rgba(var(--tint), 0.12) !important;
  border: 1px solid rgba(var(--tint), 0.45) !important;
  color: rgb(var(--tint)) !important;
}
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary.is-light:hover,
.options-section.options-section.options-section .button:not(.hover-actions *).is-primary.is-outlined:hover,
.options-section.options-section.options-section .button:not(.hover-actions *).is-danger:hover,
.options-section.options-section.options-section .button:not(.hover-actions *).is-info:hover,
.options-section.options-section.options-section .button:not(.hover-actions *).is-success:hover,
.options-section.options-section.options-section .button:not(.hover-actions *).is-warning:hover {
  background-color: rgba(var(--tint), 0.22) !important;
}
.options-section .button[disabled] {
  opacity: 0.45;
}
.options-section .b-tabs {
  margin-bottom: 1rem !important;
}
.options-section .b-tabs .tabs {
  margin-bottom: 0 !important;
}
.options-section .b-tabs .tab-content {
  padding: 0;
}
.options-section .field.is-floating-label:not(:last-child) {
  margin-bottom: 1.4rem;
}
.options-section .field.is-grouped {
  gap: 0.5rem;
  flex-wrap: wrap;
}
/* file pickers look like the other buttons */
.options-section .file-cta,
.options-section .file-name {
  height: 2.4em;
  padding: 0 1em;
  font-size: 0.85rem;
  border-radius: 8px;
}
.options-section .file.has-name .file-cta {
  border-radius: 8px 0 0 8px;
}
.options-section .file.has-name .file-name {
  border-radius: 0 8px 8px 0;
}
.options-section .b-tooltip .field {
  margin-bottom: 0;
}
.options-section .field:has(> .b-tooltip .file) {
  align-items: center;
  gap: 0.5rem;
}
.options-section .field:has(> .b-tooltip .file) > .b-tooltip {
  margin: 0 !important;
}
</style>
