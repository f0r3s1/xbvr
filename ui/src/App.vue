<template>
  <div>
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keypress="onGlobalKeypress"
    />
    <GlobalEvents @keydown="onGlobalKeydown"/>
    <svg class="glass-defs" aria-hidden="true" width="0" height="0">
      <filter id="glass-refract" x="0" y="0" width="100%" height="100%" color-interpolation-filters="sRGB">
        <feImage href="data:image/svg+xml,%3Csvg%20xmlns='http://www.w3.org/2000/svg'%20viewBox='0%200%20100%20100'%3E%3Cdefs%3E%3ClinearGradient%20id='r'%3E%3Cstop%20offset='0'%20stop-color='%23000'/%3E%3Cstop%20offset='1'%20stop-color='%23f00'/%3E%3C/linearGradient%3E%3ClinearGradient%20id='g'%20x2='0'%20y2='1'%3E%3Cstop%20offset='0'%20stop-color='%23000'/%3E%3Cstop%20offset='1'%20stop-color='%230f0'/%3E%3C/linearGradient%3E%3Cfilter%20id='b'%3E%3CfeGaussianBlur%20stdDeviation='7'/%3E%3C/filter%3E%3C/defs%3E%3Crect%20width='100'%20height='100'%20fill='url%28%23r%29'/%3E%3Crect%20width='100'%20height='100'%20fill='url%28%23g%29'%20style='mix-blend-mode:screen'/%3E%3Crect%20x='20'%20y='20'%20width='60'%20height='60'%20rx='18'%20fill='%23808000'%20filter='url%28%23b%29'/%3E%3C/svg%3E" preserveAspectRatio="none" result="map"/>
        <feDisplacementMap in="SourceGraphic" in2="map" scale="30" xChannelSelector="R" yChannelSelector="G"/>
      </filter>
    </svg>
    <Navbar/>
    <div class="navbar-pad">
      <router-view/>
    </div>

    <SceneDetails v-if="showOverlay"/>
    <EditScene v-if="showEdit" />
    <ActorDetails v-if="showActorDetails"/>
    <EditActor v-if="showActorEdit" />
    <SearchStashdbScenes v-if="showSearchStashdbScenes" />
    <SearchStashdbActors v-if="showSearchStashdbActors" />

    <QuickFind/>
    <MigrationOverlay/>

    <Socket/>
  </div>
</template>

<script>
import { defineComponent, defineAsyncComponent } from 'vue';

import { GlobalEvents } from 'vue-global-events'

import Navbar from './Navbar.vue'
import Socket from './Socket.vue'
import QuickFind from './QuickFind'
import MigrationOverlay from './components/MigrationOverlay'

const SceneDetails = defineAsyncComponent(() => import('./views/scenes/Details.vue'))
const EditScene = defineAsyncComponent(() => import('./views/scenes/EditScene.vue'))
const ActorDetails = defineAsyncComponent(() => import('./views/actors/ActorDetails.vue'))
const EditActor = defineAsyncComponent(() => import('./views/actors/EditActor.vue'))
const SearchStashdbScenes = defineAsyncComponent(() => import('./views/scenes/SearchStashdbScenes.vue'))
const SearchStashdbActors = defineAsyncComponent(() => import('./views/actors/SearchStashdbActors.vue'))

export default defineComponent({
  components: { Navbar, Socket, QuickFind, GlobalEvents, SceneDetails, EditScene, ActorDetails, EditActor, SearchStashdbScenes, SearchStashdbActors, MigrationOverlay },

  mounted () {
    // apply the saved theme before the server answers to avoid a flash
    const saved = localStorage.getItem('xbvr-theme') || 'auto'
    this.applyTheme(saved)

    // scene cards need the site list for logos, so load it at startup
    if (!this.$store.state.optionsSites.items.length) this.$store.dispatch('optionsSites/load')

    this._darkMQ = window.matchMedia('(prefers-color-scheme: dark)')
    this._darkMQHandler = () => {
      if ((this.$store.state.optionsWeb.web.theme || 'auto') === 'auto') {
        this.applyTheme('auto')
      }
    }
    this._darkMQ.addEventListener('change', this._darkMQHandler)

    // a navbar click while a scene or actor is open hits the backdrop; forward it to the link
    this._modalNavClick = e => {
      const backdrop = e.target.closest && e.target.closest('.modal-background')
      if (!backdrop || backdrop.parentElement.querySelector('.quickfind')) return // quick find just closes
      const link = document.elementsFromPoint(e.clientX, e.clientY)
        .map(el => el.closest && el.closest('.navbar a'))
        .find(Boolean)
      if (!link || link.classList.contains('router-link-exact-active')) return
      e.preventDefault()
      e.stopPropagation()
      link.click()
    }
    document.addEventListener('click', this._modalNavClick, true)

    this.$store.dispatch('optionsWeb/load')
  },

  beforeUnmount () {
    if (this._darkMQ) this._darkMQ.removeEventListener('change', this._darkMQHandler)
    document.removeEventListener('click', this._modalNavClick, true)
  },

  watch: {
    '$store.state.optionsWeb.web.theme' (theme) {
      this.applyTheme(theme || 'auto')
    },
    // Overlays are global; leaving the page they were opened from closes them.
    '$route.path' () {
      this.$store.commit('overlay/hideAll')
    }
  },

  methods: {
    onGlobalKeydown (e) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        this.$store.commit('overlay/showQuickFind')
      }
    },
    onGlobalKeypress (e) {
      if (e.key === '?') {
        e.preventDefault()
        this.$store.commit('overlay/showQuickFind')
      }
    },
    applyTheme (theme) {
      const html = document.documentElement
      if (theme === 'dark') {
        html.setAttribute('data-theme', 'dark')
      } else if (theme === 'light') {
        html.setAttribute('data-theme', 'light')
      } else {
        html.setAttribute('data-theme', window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      }
    }
  },

  computed: {
    showOverlay () {
      return this.$store.state.overlay.details.show
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    },
    showActorDetails() {
      return this.$store.state.overlay.actordetails.show
    },
    showActorEdit() {
      return this.$store.state.overlay.actoredit.show
    },
    showSearchStashdbScenes() {
      return this.$store.state.overlay.searchStashDbScenes.show
    },
    showSearchStashdbActors() {
      return this.$store.state.overlay.searchStashDbActors.show
    },
  },
});
</script>

<style>
  /* Loading overlays only appear when a load is actually slow (>300ms); fast
     loads no longer flash a spinner over content that is about to render. */
  .loading-overlay.is-active {
    animation: xbvr-delayed-fade-in 150ms ease-out 300ms both;
  }
  @keyframes xbvr-delayed-fade-in {
    from { opacity: 0; visibility: hidden; }
    to { opacity: 1; visibility: visible; }
  }

  .navbar-pad {
    margin-top: 1rem;
    padding-bottom: 1.5rem;
  }
  .navbar-pad:has(.list-toolbar) {
    margin-top: 0;
  }

  :root {
    --list-row-h: 49px;
    --list-head-gap: 0.75rem;
  }
  .list-toolbar {
    min-height: var(--list-row-h);
    box-sizing: border-box;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.5rem 0.75rem;
    position: sticky;
    top: 3.25rem;
    z-index: 6;
    padding: 0.5rem 0;
    margin-bottom: var(--list-head-gap);
    background: #fff;
    box-shadow: -12px 0 0 #fff, 12px 0 0 #fff;
    border-bottom: 1px solid rgba(127, 127, 127, 0.15);
    font-variant-numeric: tabular-nums;
  }
  .fork-credit {
    margin-top: 1.5rem;
    font-size: 0.7rem;
    opacity: 0.4;
  }
  .fork-credit a {
    color: inherit;
    text-decoration: underline;
  }
  .property-filters {
    display: grid !important;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
    margin: 0 !important;
  }
  .property-filters > .column {
    width: auto !important;
    padding: 0 !important;
  }
  .property-filters > .column:last-child:nth-child(odd) {
    grid-column: 1 / -1;
  }
  .property-filters .control,
  html[data-theme] .property-filters .button {
    width: 100%;
  }
  html[data-theme] .property-filters .button {
    height: 36px;
    margin: 0 !important;
    border-radius: 8px !important;
  }
  html .hover-actions.is-glass .alt-link {
    position: relative;
  }
  .detail-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
  }
  .detail-actions .button {
    width: 28px !important;
    height: 28px !important;
    min-width: 28px !important;
    min-height: 28px !important;
    padding: 0 !important;
    margin: 0 !important;
    border-radius: 5px !important;
    display: inline-flex !important;
    align-items: center !important;
    justify-content: center !important;
    line-height: 1 !important;
  }
  .detail-actions .button .icon {
    margin: 0 !important;
    width: 16px !important;
    height: 16px !important;
  }
  .detail-actions .button .icon .mdi {
    font-size: 16px !important;
    line-height: 1 !important;
  }
  .detail-actions .button span:not(.icon) {
    display: none !important;
  }
  .detail-actions > :nth-child(-n+3) [data-tooltip]::after,
  .detail-actions > [data-tooltip]:nth-child(-n+3)::after {
    left: 0;
    right: auto;
  }

  .glass-defs {
    position: absolute;
  }
  .list-toolbar .control,
  .segmented .control {
    margin: 0 !important;
  }
  html[data-theme] .list-toolbar .b-radio.button,
  html[data-theme] .segmented .b-radio.button {
    height: 32px;
    margin: 0 0 0 -1px !important;
    padding: 0 0.8em;
    font-size: 0.8rem;
    border-radius: 0 !important;
    border: 1px solid rgba(127, 127, 127, 0.3) !important;
    background: transparent !important;
    color: inherit !important;
    opacity: 0.75;
    box-shadow: none !important;
  }
  html[data-theme] .list-toolbar .b-radio.button:hover,
  html[data-theme] .segmented .b-radio.button:hover {
    background: rgba(127, 127, 127, 0.12) !important;
    opacity: 1;
  }
  html[data-theme] .list-toolbar .b-radio.button.is-selected,
  html[data-theme] .segmented .b-radio.button.is-selected {
    background: rgba(121, 87, 213, 0.22) !important;
    opacity: 1;
  }
  .list-toolbar .control:first-of-type .b-radio.button,
  .segmented .control:first-of-type .b-radio.button {
    margin-left: 0 !important;
    border-radius: 8px 0 0 8px !important;
  }
  .list-toolbar .control:last-of-type .b-radio.button,
  .segmented .control:last-of-type .b-radio.button {
    border-radius: 0 8px 8px 0 !important;
  }
  .list-toolbar .list-header-label {
    padding-right: 0.6em;
    white-space: nowrap;
    align-self: center;
    font-size: 0.75rem;
    opacity: 0.6;
  }
  html[data-theme="dark"] .list-toolbar {
    background: #121218;
    box-shadow: -12px 0 0 #121218, 12px 0 0 #121218;
  }
  /* ── Filter sidebars (scenes + actors) ──
     Section headers on the left with a hairline, labels above their inputs
     (not cut into the border), and more air between groups. */
  html[data-theme] .filter-panel .is-divider {
    display: flex !important;
    align-items: center;
    gap: 8px;
    height: auto !important;
    margin: 1.4rem 0 0.7rem !important;
    border: none !important;
    background: none !important;
    text-align: left !important;
  }
  @media (min-width: 769px) {
    html[data-theme] .filter-panel > .is-divider:first-child {
      height: var(--list-row-h) !important;
      margin: calc(var(--list-head-h, var(--list-row-h)) - var(--list-row-h)) 0 calc(var(--list-head-gap) - 6px) !important;
    }
  }
  @media (max-width: 768px) {
    html[data-theme] .filter-panel > .is-divider:first-child {
      margin-top: 0.25rem !important;
    }
    html .list-toolbar .toolbar-segment {
      order: 3;
      flex: 1 0 100%;
      flex-wrap: nowrap;
      overflow-x: auto;
      scrollbar-width: none;
      justify-content: flex-start;
    }
    .list-toolbar .toolbar-segment .b-radio.button {
      white-space: nowrap;
    }
    html .list-toolbar .toolbar-total {
      flex: 1 1 auto;
    }
    .list-toolbar .list-header-label {
      display: none;
    }
  }
  html[data-theme] .filter-panel .is-divider::before {
    content: '';
    flex: 1;
    height: 1px;
    background: rgba(127, 127, 127, 0.22);
    order: 2;
  }
  html[data-theme] .filter-panel .is-divider[data-content]::after {
    position: static;
    order: 1;
    transform: none !important;
    padding: 0 !important;
    background: none !important;
    font-size: 0.68rem !important;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: inherit !important;
    opacity: 0.55;
  }
  :root {
    --card-bg: #fff;
    --card-placeholder: #e9eaef;
  }
  html[data-theme="dark"] {
    --card-bg: #1c1c26;
    --card-placeholder: #262633;
  }
  .skeleton-card {
    border-radius: 8px;
    overflow: hidden;
    background: var(--card-bg);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
    animation: fade-in 0.2s ease-out both;
  }
  .skeleton-block {
    background: rgba(127, 127, 127, 0.16);
    border-radius: 4px;
    animation: skeleton-pulse 1.6s ease-in-out infinite alternate;
  }
  .skeleton-thumb {
    width: 100%;
    border-radius: 0;
    background: var(--card-placeholder);
  }
  .skeleton-info {
    height: 52px;
    box-sizing: border-box;
    padding: 9px 10px;
    display: flex;
    flex-direction: column;
    gap: 9px;
  }
  .skeleton-line {
    height: 11px;
    width: 72%;
  }
  .skeleton-line.is-short {
    height: 9px;
    width: 40%;
  }
  @keyframes skeleton-pulse {
    from { opacity: 1; }
    to { opacity: 0.6; }
  }
  @keyframes fade-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }
  @media (prefers-reduced-motion: reduce) {
    .skeleton-card, .skeleton-block { animation: none; }
  }
  /* labels sit inside their field; the column fits the longest label of the panel (v-fit-labels) */
  .filter-panel .field.is-floating-label {
    position: relative;
    --label-col: var(--panel-label-col, 84px);
  }
  .filter-panel .field.is-floating-label .label {
    position: absolute !important;
    top: 20px !important;
    left: 12px !important;
    z-index: 5;
    width: calc(var(--label-col) - 16px);
    margin: 0 !important;
    padding: 0 !important;
    transform: translateY(-50%) !important;
    background: none !important;
    font-size: 0.7rem !important;
    font-weight: 500;
    line-height: 1.2;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    opacity: 0.55;
    pointer-events: none;
  }
  .filter-panel .field.is-floating-label .label::before {
    display: none !important;
  }
  .filter-panel .field.is-floating-label .select,
  .filter-panel .field.is-floating-label .select select,
  .filter-panel .field.is-floating-label .control > .input {
    height: 40px !important;
  }
  .filter-panel .field.is-floating-label .select select,
  .filter-panel .field.is-floating-label .control > .input {
    padding-top: 0 !important;
    padding-bottom: 0 !important;
    padding-left: var(--label-col) !important;
    font-size: 0.875rem;
  }
  .filter-panel .field.is-floating-label .select:not(.is-multiple):not(.is-loading)::after {
    top: 50% !important;
    margin-top: -0.375em !important;
  }
  .filter-panel .field.is-floating-label .taginput .taginput-container {
    min-height: 40px !important;
    padding: 4px 8px 4px var(--label-col) !important;
    align-items: center;
  }
  /* typing slot: takes the rest of the row, wraps to its own line below 80px;
     Buefy's 8em min-width on the input made it overflow the field after a chip */
  .filter-panel .field.is-floating-label .taginput-container .autocomplete,
  .filter-panel .field.is-floating-label .taginput-container .control {
    flex: 1 1 80px;
    min-width: 80px;
  }
  .filter-panel .taginput .taginput-container .autocomplete input {
    min-width: 0 !important;
    max-width: 100% !important;
  }
  /* with chips and not focused, the empty typing slot takes no room (no extra
     line); clicking the field focuses it and the slot opens again */
  .filter-panel .field.is-floating-label .taginput-container:has(.tag):not(:focus-within) .autocomplete {
    flex: 0 0 0;
    min-width: 0;
    width: 0;
    overflow: hidden;
  }
  .filter-panel .field.is-floating-label .taginput-container .input,
  .filter-panel .field.is-floating-label .taginput-container input {
    width: 100% !important;
    height: 30px !important;
    min-height: 0 !important;
    padding: 0 4px !important;
    margin: 0 !important;
    font-size: 0.875rem;
  }
  .filter-panel .field.is-floating-label .taginput-container .tag {
    height: 24px;
    margin: 2px 4px 2px 0 !important;
  }
  html[data-theme] .filter-panel .field.is-floating-label .control .button {
    height: 40px !important;
    background: transparent !important;
    border: 1px solid rgba(127, 127, 127, 0.3) !important;
  }
  html[data-theme] .filter-panel .field.is-floating-label .control .button:hover {
    background: rgba(127, 127, 127, 0.12) !important;
  }

  /* Saved searches: one segmented control (select + save/edit/delete) */
  .filter-panel .saved-search .field.is-grouped {
    gap: 0;
    margin-bottom: 0;
  }
  .filter-panel .saved-search .field.is-grouped > .control,
  .filter-panel .saved-search .field.is-grouped > .b-tooltip {
    margin: 0 !important;
  }
  .filter-panel .saved-search .select,
  .filter-panel .saved-search .select select {
    height: 40px !important;
  }
  .filter-panel .saved-search .select select {
    border-radius: 8px 0 0 8px !important;
    font-size: 0.85rem;
  }
  .filter-panel .saved-search .button {
    height: 40px !important;
    width: 40px;
    margin-left: -1px !important;
    border-radius: 0 !important;
    border-color: rgba(127, 127, 127, 0.3) !important;
    background: transparent !important;
    opacity: 0.85;
  }
  .filter-panel .saved-search .button:hover:not([disabled]) {
    opacity: 1;
    background: rgba(127, 127, 127, 0.12) !important;
  }
  /* disabled: fade the icon only, so every segment keeps the same border */
  .filter-panel .saved-search .button[disabled] {
    opacity: 1;
  }
  .filter-panel .saved-search .button[disabled] .icon {
    opacity: 0.35;
  }
  .filter-panel .saved-search .b-tooltip:last-child .button {
    border-radius: 0 8px 8px 0 !important;
  }
  .filter-panel .field-extra {
    margin-bottom: 0.9rem !important;
  }
  /* Watchlist / Favourite / ... toggles as filter chips: neutral when off, a soft
     tint of their colour when on (no heavy solid buttons) */
  .filter-panel .columns.is-gapless {
    display: grid !important;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
    margin: 0 !important;
  }
  .filter-panel .columns.is-gapless > .column {
    width: auto !important;
    padding: 0 !important;
  }
  .filter-panel .columns.is-gapless .control,
  .filter-panel .columns.is-gapless .b-checkbox.button {
    width: 100%;
  }
  html[data-theme] .filter-panel .columns.is-gapless label.button {
    justify-content: flex-start;
    gap: 6px;
    height: 34px;
    padding: 0 10px !important;
    border-radius: 8px !important;
    border: 1px solid rgba(127, 127, 127, 0.22) !important;
    background: rgba(127, 127, 127, 0.06) !important;
    color: inherit !important;
    font-size: 0.8rem;
    font-weight: 500;
    box-shadow: none !important;
    transition: background 0.12s, border-color 0.12s, color 0.12s;
  }
  html[data-theme] .filter-panel .columns.is-gapless label.button:hover {
    background: rgba(127, 127, 127, 0.12) !important;
  }
  .filter-panel .columns.is-gapless label.button .icon {
    margin: 0 !important;
    width: 16px;
    height: 16px;
    opacity: 0.75;
  }
  .filter-panel .columns.is-gapless label.button .icon .mdi::before {
    font-size: 16px;
  }
  html[data-theme] .filter-panel .columns.is-gapless label.button.is-primary {
    background: rgba(121, 87, 213, 0.16) !important;
    border-color: rgba(121, 87, 213, 0.55) !important;
    color: #9d84ef !important;
  }
  html[data-theme] .filter-panel .columns.is-gapless label.button.is-danger {
    background: rgba(241, 70, 104, 0.14) !important;
    border-color: rgba(241, 70, 104, 0.5) !important;
    color: #f2708b !important;
  }
  html[data-theme] .filter-panel .columns.is-gapless label.button.is-info {
    background: rgba(62, 142, 208, 0.15) !important;
    border-color: rgba(62, 142, 208, 0.5) !important;
    color: #5aa5e6 !important;
  }
  .filter-panel .columns.is-gapless label.button[class*="is-"]:not(.is-small) .icon {
    opacity: 1;
  }

  /* Aka group / tag group actions: one full-width segmented bar, same language as
     the saved searches control */
  .filter-panel .group-actions,
  .filter-panel .group-actions > .field-body > .field,
  .filter-panel .group-actions .field.has-addons {
    display: grid !important;
    grid-auto-flow: column;
    grid-auto-columns: 1fr;
    gap: 0;
    margin-bottom: 0;
  }
  .filter-panel .group-actions .b-tooltip,
  .filter-panel .group-actions .b-tooltip > .tooltip-trigger {
    display: block;
    width: 100%;
    margin: 0 !important;
  }
  html[data-theme] .filter-panel .group-actions .button {
    width: 100%;
    height: 36px;
    margin: 0 0 0 -1px !important;
    border-radius: 0 !important;
    border: 1px solid rgba(127, 127, 127, 0.3) !important;
    background: transparent !important;
    color: inherit !important;
    box-shadow: none !important;
  }
  html[data-theme] .filter-panel .group-actions .button:hover:not([disabled]) {
    background: rgba(127, 127, 127, 0.12) !important;
  }
  .filter-panel .group-actions .b-tooltip:first-child .button {
    margin-left: 0 !important;
    border-radius: 8px 0 0 8px !important;
  }
  .filter-panel .group-actions .b-tooltip:last-child .button {
    border-radius: 0 8px 8px 0 !important;
  }
  .filter-panel .group-actions .button[disabled] {
    opacity: 1;
  }
  .filter-panel .group-actions .button[disabled] .icon {
    opacity: 0.35;
  }

  /* Filter chips: state shown by colour (exclude red, must-have green), full
     text on hover via the native title, nothing pops outside the field */
  .filter-panel .taginput-container .tag {
    max-width: 100%;
  }
  .filter-panel .taginput-container .tag .chip-label {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .filter-panel .taginput-container .tag .chip-label .icon {
    margin: 0 !important;
  }
  html[data-theme] .filter-panel .taginput-container .tag.is-danger {
    background: rgba(241, 70, 104, 0.16) !important;
    color: #f2708b !important;
    box-shadow: inset 0 0 0 1px rgba(241, 70, 104, 0.45);
  }
  html[data-theme] .filter-panel .taginput-container .tag.is-success {
    background: rgba(72, 199, 142, 0.16) !important;
    color: #3fbf86 !important;
    box-shadow: inset 0 0 0 1px rgba(72, 199, 142, 0.45);
  }

  /* Narrow sidebar (small windows): labels go above their field instead of
     squeezing the value, chips stack, headers stay on one line */
  .filter-panel {
    container-type: inline-size;
  }
  html[data-theme] .filter-panel .is-divider[data-content]::after {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }
  @container (max-width: 240px) {
    .filter-panel .field.is-floating-label .label {
      position: static !important;
      transform: none !important;
      width: auto;
      margin: 0 0 4px !important;
    }
    .filter-panel .field.is-floating-label {
      flex-wrap: wrap;
    }
    .filter-panel .field.is-floating-label > .label {
      flex: 1 0 100%;
    }
    .filter-panel .field.is-floating-label .select select,
    .filter-panel .field.is-floating-label .control > .input {
      padding-left: 10px !important;
    }
    .filter-panel .field.is-floating-label .taginput .taginput-container {
      padding-left: 8px !important;
    }
    .filter-panel .columns.is-gapless {
      grid-template-columns: 1fr;
    }
    /* full-width chips have room for their text again */
    .filter-panel .columns.is-multiline.is-gapless > .column.is-half .b-checkbox.button span:not(.icon) {
      display: inline !important;
    }
  }

  /* Cards inside a modal (actor scenes, colleagues, akas) sit one step above the
     modal surface, otherwise their footer blends into it. */
  html .modal .modal-card .scene-card,
  html .modal .modal-card .actor-card {
    background: #f5f6f8;
    box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.06);
  }
  html[data-theme="dark"] .modal .modal-card .scene-card,
  html[data-theme="dark"] .modal .modal-card .actor-card {
    background: #262633;
    box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.06);
  }
  .modal-background {
    background-color: rgba(0, 0, 0, .40) !important;
  }

  /* ── Dark Mode (JS-driven: html[data-theme="dark"] set by App.vue) ── */

  /* Base surfaces */
  html[data-theme="dark"],
  html[data-theme="dark"] body,
  html[data-theme="dark"] .section,
  html[data-theme="dark"] #app {
    background-color: #121218 !important;
    color: #d4d4d8 !important;
  }

  /* Navbar */
  html[data-theme="dark"] .navbar,
  html[data-theme="dark"] .navbar-menu,
  html[data-theme="dark"] .navbar-dropdown,
  html[data-theme="dark"] .navbar-start,
  html[data-theme="dark"] .navbar-end {
    background-color: #18181f !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .navbar-item,
  html[data-theme="dark"] .navbar-link,
  html[data-theme="dark"] .navbar-burger,
  html[data-theme="dark"] .navbar-brand .navbar-item {
    color: #d4d4d8 !important;
  }
  /* hover is a faint tint; only the current page gets the filled tab + accent line,
     so two tabs never look selected at once */
  html[data-theme="dark"] .navbar-item:hover,
  html[data-theme="dark"] .navbar-item:focus,
  html[data-theme="dark"] .navbar-item:active,
  html[data-theme="dark"] .navbar-item:focus-within,
  html[data-theme="dark"] .navbar-link:hover,
  html[data-theme="dark"] .navbar-link:focus {
    background-color: rgba(255, 255, 255, 0.05) !important;
    color: #fff !important;
  }
  html[data-theme="dark"] .navbar-item.is-active,
  html[data-theme="dark"] .navbar-item.router-link-active {
    background-color: #222230 !important;
    color: #fff !important;
  }
  .navbar-start .navbar-item.router-link-active {
    box-shadow: inset 0 -2px 0 var(--bulma-primary, #7957d5);
  }
  .navbar-item:focus:not(:focus-visible) {
    outline: none;
  }
  .navbar-item:focus-visible {
    outline: 2px solid var(--bulma-primary, #7957d5);
    outline-offset: -2px;
  }
  html[data-theme="dark"] .navbar-divider {
    background-color: #2a2a35 !important;
  }

  /* Cards & Boxes */
  html[data-theme="dark"] .card,
  html[data-theme="dark"] .box,
  html[data-theme="dark"] .panel {
    background-color: #1c1c26 !important;
    color: #d4d4d8 !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .card-header {
    background-color: #1f1f2a !important;
    border-color: #2a2a35 !important;
    box-shadow: none !important;
  }
  html[data-theme="dark"] .card-header-title,
  html[data-theme="dark"] .card-content,
  html[data-theme="dark"] .card-footer {
    color: #d4d4d8 !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .card-footer-item {
    border-color: #2a2a35 !important;
    color: #a0a0b0 !important;
  }

  /* Modal */
  html[data-theme="dark"] .modal-card,
  html[data-theme="dark"] .modal-card-body,
  html[data-theme="dark"] .modal-card-head,
  html[data-theme="dark"] .modal-card-foot {
    background-color: #1c1c26 !important;
    color: #d4d4d8 !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .modal-card-title { color: #e8e8ec !important; }
  html[data-theme="dark"] .modal-background { background-color: rgba(0,0,0,.70) !important; }
  html[data-theme="dark"] .modal-close::before,
  html[data-theme="dark"] .modal-close::after { background-color: #d4d4d8 !important; }

  /* Tables */
  html[data-theme="dark"] .table,
  html[data-theme="dark"] .table thead,
  html[data-theme="dark"] .table tbody,
  html[data-theme="dark"] .table tfoot {
    background-color: #1c1c26 !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] .table th {
    color: #a0a0b0 !important;
    border-color: #2a2a35 !important;
    background-color: #1f1f2a !important;
  }
  html[data-theme="dark"] .table td {
    border-color: #2a2a35 !important;
    color: #c8c8d0 !important;
    background-color: #1c1c26 !important;
  }
  html[data-theme="dark"] .table tr,
  html[data-theme="dark"] .table tbody tr { background-color: #1c1c26 !important; }
  html[data-theme="dark"] .table tr:hover,
  html[data-theme="dark"] .table tr.is-selected { background-color: #222230 !important; }
  html[data-theme="dark"] .table tr:hover td { background-color: #222230 !important; }
  html[data-theme="dark"] .table.is-striped tbody tr:nth-child(even),
  html[data-theme="dark"] .table.is-striped tbody tr:nth-child(even) td {
    background-color: #1e1e28 !important;
  }

  /* Forms & Inputs */
  html[data-theme="dark"] .input,
  html[data-theme="dark"] .textarea,
  html[data-theme="dark"] .select select,
  html[data-theme="dark"] .taginput .taginput-container,
  html[data-theme="dark"] .datepicker .dropdown-content,
  html[data-theme="dark"] .autocomplete .dropdown-content {
    background-color: #222230 !important;
    border-color: #3a3a48 !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] .input:focus,
  html[data-theme="dark"] .textarea:focus,
  html[data-theme="dark"] .select select:focus {
    border-color: #5a5a70 !important;
    box-shadow: 0 0 0 0.125em rgba(100,100,140,0.25) !important;
  }
  html[data-theme="dark"] .input::placeholder,
  html[data-theme="dark"] .textarea::placeholder { color: #666 !important; }
  html[data-theme="dark"] .label { color: #b0b0c0 !important; }
  html[data-theme="dark"] .field-label .label { color: #b0b0c0 !important; }
  html[data-theme="dark"] .checkbox,
  html[data-theme="dark"] .radio { color: #d4d4d8 !important; }
  html[data-theme="dark"] .select::after { border-color: #888 !important; }

  /* Buttons */
  html[data-theme="dark"] .button {
    background-color: #222230 !important;
    border-color: #3a3a48 !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] .button:hover { border-color: #5a5a70 !important; color: #fff !important; }
  html[data-theme="dark"] .button.is-primary {
    background-color: #5b4fc7 !important;
    border-color: #5b4fc7 !important;
    color: #fff !important;
  }
  html[data-theme="dark"] .button.is-primary.is-outlined {
    background-color: transparent !important;
    border-color: #6b5fd7 !important;
    color: #8b7ff0 !important;
  }
  html[data-theme="dark"] .button.is-primary.is-outlined:hover {
    background-color: #5b4fc7 !important;
    color: #fff !important;
  }
  html[data-theme="dark"] .button.is-info {
    background-color: #2a6090 !important;
    border-color: #2a6090 !important;
  }
  html[data-theme="dark"] .button.is-info.is-outlined {
    background-color: transparent !important;
    border-color: #3a80b0 !important;
    color: #5aa0d0 !important;
  }
  html[data-theme="dark"] .button.is-dark {
    background-color: #2a2a3a !important;
    border-color: #3a3a48 !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] .button.is-dark.is-outlined {
    border-color: #4a4a58 !important;
    color: #b0b0c0 !important;
    background-color: transparent !important;
  }
  html[data-theme="dark"] .button.is-danger {
    background-color: #8b2030 !important;
    border-color: #8b2030 !important;
  }
  html[data-theme="dark"] .button.is-danger.is-outlined {
    background-color: transparent !important;
    border-color: #a03040 !important;
    color: #e06070 !important;
  }
  html[data-theme="dark"] .button.is-warning {
    background-color: #7a6500 !important;
    border-color: #7a6500 !important;
    color: #f0e070 !important;
  }
  html[data-theme="dark"] .button.is-success {
    background-color: #1a7040 !important;
    border-color: #1a7040 !important;
  }
  html[data-theme="dark"] .button[disabled] { opacity: 0.4 !important; }

  /* Tabs */
  html[data-theme="dark"] .tabs,
  html[data-theme="dark"] .tabs ul {
    border-bottom-color: #2a2a35 !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .tabs a {
    color: #888 !important;
    border-bottom-color: transparent !important;
    background-color: transparent !important;
  }
  html[data-theme="dark"] .tabs a:hover {
    color: #c0c0c8 !important;
    border-bottom-color: #5a5a70 !important;
    background-color: #222230 !important;
  }
  html[data-theme="dark"] .tabs li.is-active a {
    color: #8b7ff0 !important;
    border-bottom-color: #8b7ff0 !important;
    background-color: transparent !important;
  }
  html[data-theme="dark"] .tabs.is-boxed a {
    border-color: transparent !important;
    border-radius: 4px 4px 0 0 !important;
  }
  html[data-theme="dark"] .tabs.is-boxed a:hover {
    background-color: #222230 !important;
    border-color: #3a3a48 !important;
  }
  html[data-theme="dark"] .tabs.is-boxed li.is-active a {
    background-color: #1c1c26 !important;
    border-color: #3a3a48 #3a3a48 #1c1c26 !important;
    color: #e8e8ec !important;
  }
  html[data-theme="dark"] .tab-content,
  html[data-theme="dark"] .tabs + .tab-content,
  html[data-theme="dark"] section.tab-content { background-color: transparent !important; }

  /* Tags */
  html[data-theme="dark"] .tag { background-color: #2a2a3a !important; color: #d4d4d8 !important; }
  html[data-theme="dark"] .tag.is-info { background-color: #1a4060 !important; color: #80c0f0 !important; }
  html[data-theme="dark"] .tag.is-warning { background-color: #4a3a00 !important; color: #f0d860 !important; }
  html[data-theme="dark"] .tag.is-primary { background-color: #2a2070 !important; color: #a090f0 !important; }
  html[data-theme="dark"] .tag.is-danger { background-color: #5a1020 !important; color: #f08090 !important; }
  html[data-theme="dark"] .tag.is-success { background-color: #0a4020 !important; color: #60d090 !important; }
  html[data-theme="dark"] .tag.is-light { background-color: #2a2a3a !important; color: #c0c0c8 !important; }
  html[data-theme="dark"] .tag .delete { background-color: rgba(255,255,255,0.2) !important; }

  /* Dropdowns & Menus */
  html[data-theme="dark"] .dropdown-content {
    background-color: #1c1c26 !important;
    border-color: #2a2a35 !important;
    box-shadow: 0 6px 20px rgba(0,0,0,0.5) !important;
  }
  html[data-theme="dark"] .dropdown-item,
  html[data-theme="dark"] a.dropdown-item { color: #d4d4d8 !important; }
  html[data-theme="dark"] .dropdown-item:hover,
  html[data-theme="dark"] a.dropdown-item:hover { background-color: #222230 !important; color: #fff !important; }
  html[data-theme="dark"] .dropdown-divider { background-color: #2a2a35 !important; }

  /* Pagination */
  html[data-theme="dark"] .pagination-link,
  html[data-theme="dark"] .pagination-previous,
  html[data-theme="dark"] .pagination-next,
  html[data-theme="dark"] .pagination-ellipsis {
    background-color: #222230 !important;
    border-color: #3a3a48 !important;
    color: #b0b0c0 !important;
  }
  html[data-theme="dark"] .pagination-link:hover,
  html[data-theme="dark"] .pagination-previous:hover,
  html[data-theme="dark"] .pagination-next:hover { border-color: #5a5a70 !important; color: #fff !important; }
  html[data-theme="dark"] .pagination-link.is-current {
    background-color: #5b4fc7 !important;
    border-color: #5b4fc7 !important;
    color: #fff !important;
  }

  /* Tooltips */
  html[data-theme="dark"] .b-tooltip .tooltip-content { background-color: #2a2a3a !important; color: #e8e8ec !important; }
  .b-tooltip.is-small .tooltip-content {
    padding: 4px 8px !important;
    font-size: 11px !important;
    font-weight: 400 !important;
    border-radius: 4px !important;
    background: rgba(0,0,0,0.85) !important;
    color: #fff !important;
  }
  html[data-theme="dark"] .b-tooltip.is-small .tooltip-content {
    background: rgba(0,0,0,0.85) !important;
    color: #fff !important;
  }

  /* Progress */
  html[data-theme="dark"] .progress { background-color: #2a2a3a !important; }

  /* Notifications & Messages */
  html[data-theme="dark"] .notification { background-color: #1f1f2a !important; color: #d4d4d8 !important; }
  html[data-theme="dark"] .message { background-color: #1f1f2a !important; }
  html[data-theme="dark"] .message-body {
    background-color: #1c1c26 !important;
    border-color: #2a2a35 !important;
    color: #d4d4d8 !important;
  }

  /* Loading */
  html[data-theme="dark"] .loading-overlay .loading-background { background-color: rgba(18,18,24,0.7) !important; }

  /* Switch */
  html[data-theme="dark"] .switch .control-label { color: #d4d4d8 !important; }
  html[data-theme="dark"] .switch input[type=checkbox] + .check { background: #3a3a4a !important; }
  html[data-theme="dark"] .switch input[type=checkbox]:checked + .check { background: #5b4fc7 !important; }
  html[data-theme="dark"] .switch input[type=checkbox] + .check::before { background: #c0c0cc !important; }
  html[data-theme="dark"] .switch input[type=checkbox]:checked + .check::before { background: #fff !important; }

  /* Dialog */
  html[data-theme="dark"] .dialog .modal-card-body { background-color: #1c1c26 !important; }
  html[data-theme="dark"] .dialog .modal-card-foot { background-color: #1f1f2a !important; border-color: #2a2a35 !important; }

  /* Breadcrumb */
  html[data-theme="dark"] .breadcrumb a { color: #8b7ff0 !important; }
  html[data-theme="dark"] .breadcrumb li.is-active a { color: #d4d4d8 !important; }

  /* Text & Typography */
  html[data-theme="dark"] .title,
  html[data-theme="dark"] .subtitle { color: #e8e8ec !important; }
  html[data-theme="dark"] .content h1,
  html[data-theme="dark"] .content h2,
  html[data-theme="dark"] .content h3,
  html[data-theme="dark"] .content h4,
  html[data-theme="dark"] .content h5,
  html[data-theme="dark"] .content h6 { color: #e8e8ec !important; }
  html[data-theme="dark"] .content p,
  html[data-theme="dark"] .content li,
  html[data-theme="dark"] .content blockquote { color: #d4d4d8 !important; }
  html[data-theme="dark"] strong,
  html[data-theme="dark"] b { color: #e8e8ec !important; }
  html[data-theme="dark"] a { color: #8b7ff0 !important; }
  html[data-theme="dark"] a:hover { color: #a899ff !important; }
  html[data-theme="dark"] .has-text-grey,
  html[data-theme="dark"] .has-text-grey-light { color: #888 !important; }

  /* Dividers & Lines */
  html[data-theme="dark"] hr,
  html[data-theme="dark"] .is-divider,
  html[data-theme="dark"] .is-divider-vertical {
    background-color: #2a2a35 !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .is-divider[data-content]::after {
    background-color: #121218 !important;
    color: #888 !important;
  }

  /* Field labels */
  html[data-theme="dark"] .field .label,
  html[data-theme="dark"] .field-label .label { color: #b0b0c0 !important; }
  html[data-theme="dark"] .field.is-floating-label .label {
    background-color: transparent !important;
    color: #888 !important;
    padding: 0 4px !important;
  }
  html[data-theme="dark"] .field.is-floating-label .label::before { background-color: transparent !important; }
  html[data-theme="dark"] .field.has-addons .control .select select,
  html[data-theme="dark"] .field.has-addons .control .input {
    background-color: #222230 !important;
    border-color: #3a3a48 !important;
    color: #d4d4d8 !important;
  }

  /* Slider */
  html[data-theme="dark"] .b-slider .b-slider-track { background-color: #2a2a35 !important; }
  html[data-theme="dark"] .b-slider .b-slider-fill { background-color: #5b4fc7 !important; }
  html[data-theme="dark"] .b-slider .b-slider-thumb-wrapper .b-slider-thumb {
    background-color: #d4d4d8 !important;
    border-color: #5b4fc7 !important;
  }

  /* Columns borders */
  html[data-theme="dark"] .columns.is-variable > .column { border-color: #2a2a35 !important; }

  /* Scrollbars */
  html[data-theme="dark"] * { scrollbar-color: #3a3a48 #1c1c26; }
  html[data-theme="dark"] *::-webkit-scrollbar { background: #1c1c26; }
  html[data-theme="dark"] *::-webkit-scrollbar-thumb { background: #3a3a48; border-radius: 4px; }
  html[data-theme="dark"] *::-webkit-scrollbar-thumb:hover { background: #5a5a70; }

  /* Star rating */
  html[data-theme="dark"] .vue-star-rating-star { filter: brightness(0.9); }

  /* Buefy specific */
  html[data-theme="dark"] .b-checkbox.checkbox input[type=checkbox] + .check {
    background-color: #222230 !important;
    border-color: #3a3a48 !important;
  }
  html[data-theme="dark"] .b-radio.radio input[type=radio] + .check { border-color: #3a3a48 !important; }
  html[data-theme="dark"] .taginput .taginput-container > .tag {
    background-color: #3a3a4a !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] .upload .upload-draggable {
    background-color: #1f1f2a !important;
    border-color: #3a3a48 !important;
    color: #d4d4d8 !important;
  }
  html[data-theme="dark"] .sidebar-content { background-color: #18181f !important; color: #d4d4d8 !important; }
  html[data-theme="dark"] .menu-list a { color: #b0b0c0 !important; }
  html[data-theme="dark"] .menu-list a:hover,
  html[data-theme="dark"] .menu-list a.is-active { background-color: #222230 !important; color: #fff !important; }
  html[data-theme="dark"] .menu-label { color: #888 !important; }
  html[data-theme="dark"] .panel-heading {
    background-color: #1f1f2a !important;
    color: #d4d4d8 !important;
    border-color: #2a2a35 !important;
  }
  html[data-theme="dark"] .panel-block,
  html[data-theme="dark"] .panel-tabs { border-color: #2a2a35 !important; color: #d4d4d8 !important; }
  html[data-theme="dark"] .panel-block:hover { background-color: #222230 !important; }
  html[data-theme="dark"] .image.is-1by1,
  html[data-theme="dark"] .image.is-4by3 { background-color: #222230 !important; }
  html[data-theme="dark"] .notices .toast,
  html[data-theme="dark"] .notices .snackbar { background-color: #2a2a3a !important; color: #d4d4d8 !important; }

  /* ── Mobile Responsive ── */
  @media screen and (max-width: 768px) {
    /* Reduce container fluid padding on mobile */
    .container.is-fluid {
      padding-left: 16px !important;
      padding-right: 16px !important;
    }

    /* Stack columns vertically, filters full width */
    .container.is-fluid > .columns > .column.is-one-fifth {
      width: 100% !important;
      flex: none !important;
    }
    .container.is-fluid > .columns > .column.is-four-fifths {
      width: 100% !important;
      flex: none !important;
    }

    /* Card grid: add bottom margin between cards */
    .columns.is-multiline > .column {
      padding-bottom: 0.5rem !important;
    }

    /* Properties 2x2 grid: force flex row on mobile */
    .columns.is-multiline.is-gapless {
      display: flex !important;
      flex-direction: row !important;
      flex-wrap: wrap !important;
    }
    .columns.is-multiline.is-gapless > .column.is-half {
      width: 50% !important;
      flex: 0 0 50% !important;
      max-width: 50% !important;
    }
    .columns.is-multiline.is-gapless > .column.is-half .b-checkbox.button {
      width: 100% !important;
      justify-content: center !important;
    }
    /* Hide button text, show only icon on mobile properties */
    .columns.is-multiline.is-gapless > .column.is-half .b-checkbox.button span:not(.icon) {
      display: none !important;
    }

    /* Navbar end: prevent status table from taking too much space */
    .navbar-end .navbar-item table {
      font-size: 0.75em !important;
      max-width: 200px;
      overflow: hidden;
    }
    .navbar-end .navbar-item table td {
      max-width: 120px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    /* Reduce top padding */
    .navbar-pad {
      margin-top: 0.5rem !important;
    }
  }

  /* ── Properties checkbox-buttons: icon + label always centred ── */
  /* This applies everywhere (scenes & actors filters) without per-component duplication */
  .b-checkbox.button {
    display: inline-flex !important;
    align-items: center !important;
    justify-content: center !important;
    gap: 0.35em;
    width: 100%;
  }
  .b-checkbox.button .icon {
    margin: 0 !important;
    height: 1em !important;
    width: 1em !important;
    line-height: 1 !important;
  }
  .b-checkbox.button .icon .mdi::before {
    line-height: 1 !important;
    vertical-align: middle !important;
  }

  /* ── Consistent bottom scroll padding on mobile ── */
  @media screen and (max-width: 768px) {
    .navbar-pad {
      padding-bottom: 1.5rem !important;
    }
  }

  /* Medium screens: properties buttons icon-only when sidebar is narrow */
  @media screen and (max-width: 1215px) {
    .columns.is-multiline.is-gapless > .column.is-half .b-checkbox.button span:not(.icon) {
      display: none !important;
    }
  }

  .scene-card,
  .actor-card {
    container-type: inline-size;
  }
  html[data-theme] .hover-actions.is-glass {
    flex-wrap: nowrap;
    max-width: calc(100% - 12px);
    left: auto;
    transition: opacity 0.18s ease !important;
    gap: 0;
    padding: 3px;
    border-radius: 12px;
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.12), rgba(255, 255, 255, 0.02) 55%);
    -webkit-backdrop-filter: saturate(150%) brightness(1.06);
    backdrop-filter: url(#glass-refract) saturate(150%) brightness(1.06);
    box-shadow:
      inset 0 1px 1px rgba(255, 255, 255, 0.55),
      inset 0 -1px 1px rgba(255, 255, 255, 0.18),
      inset 0 0 0 1px rgba(255, 255, 255, 0.22),
      0 6px 18px rgba(0, 0, 0, 0.28);
  }

  html[data-theme] .hover-actions.is-glass .button,
  html[data-theme] .hover-actions.is-glass .alt-link {
    --tint: 255, 255, 255;
    border: none !important;
    border-radius: 9px !important;
    background: transparent !important;
    color: #fff !important;
    opacity: 1 !important;
    box-shadow: none !important;
    transition: background 0.15s ease;
  }
  html[data-theme] .hover-actions.is-glass .button .icon,
  html[data-theme] .hover-actions.is-glass .alt-link img {
    filter: drop-shadow(0 1px 1.5px rgba(0, 0, 0, 0.55));
    opacity: 0.8;
    transition: filter 0.15s ease, opacity 0.15s ease;
  }
  .hover-actions .button svg {
    width: 13px;
    height: 13px;
  }
  html[data-theme] .hover-actions.is-glass .button:hover .icon,
  html[data-theme] .hover-actions.is-glass .button:not(.is-outlined) .icon {
    opacity: 1;
  }
  .hover-actions [data-tooltip]:not(.tip-right)::after {
    left: 0;
    right: auto;
  }
  .hover-actions [data-tooltip].tip-right::after {
    left: auto;
    right: 0;
  }
  html[data-theme] .hover-actions.is-glass .button.is-primary { --tint: 167, 139, 250; }
  html[data-theme] .hover-actions.is-glass .button.is-danger { --tint: 251, 113, 133; }
  html[data-theme] .hover-actions.is-glass .button.is-info { --tint: 96, 165, 250; }
  html[data-theme] .hover-actions.is-glass .button.is-link { --tint: 129, 140, 248; }
  html[data-theme] .hover-actions.is-glass .button.is-success { --tint: 52, 211, 153; }
  html[data-theme] .hover-actions.is-glass .button.is-warning { --tint: 251, 191, 36; }
  html[data-theme] .hover-actions.is-glass .button:not(.is-outlined) .icon {
    filter: drop-shadow(0 0 7px rgba(var(--tint), 0.9));
  }
  html[data-theme] .hover-actions.is-glass .button:hover,
  html[data-theme] .hover-actions.is-glass .alt-link:hover {
    background: rgba(255, 255, 255, 0.1) !important;
  }
  .thumbnail-wrapper:has(.hover-actions.is-glass)::after {
    content: '';
    position: absolute;
    inset: auto 0 0 0;
    height: 45%;
    border-radius: inherit;
    background: linear-gradient(to bottom, transparent, rgba(0, 0, 0, 0.45));
    pointer-events: none;
    z-index: 24;
    opacity: 0;
    transition: opacity 0.18s ease;
  }
  .thumbnail-wrapper:has(.hover-actions.is-glass.is-visible)::after {
    opacity: 1;
  }
  @container (max-width: 260px) {
    html[data-theme] .hover-actions.is-glass .button,
    html[data-theme] .hover-actions.is-glass .alt-link {
      width: 22px !important;
      height: 22px !important;
      min-width: 22px !important;
      min-height: 22px !important;
      max-width: 22px !important;
      max-height: 22px !important;
    }
    html[data-theme] .hover-actions.is-glass .button .mdi {
      font-size: 13px !important;
    }
  }
  @container (max-width: 190px) {
    html[data-theme] .hover-actions.is-glass {
      padding: 2px;
      border-radius: 9px;
    }
    html[data-theme] .hover-actions.is-glass .button,
    html[data-theme] .hover-actions.is-glass .alt-link {
      width: 18px !important;
      height: 18px !important;
      min-width: 18px !important;
      min-height: 18px !important;
      max-width: 18px !important;
      max-height: 18px !important;
    }
    html[data-theme] .hover-actions.is-glass .button .mdi {
      font-size: 11px !important;
    }
  }

  /* tables everywhere (settings, files, matching): one modern look */
  .b-table .table-wrapper {
    border: 1px solid rgba(128, 128, 160, 0.16);
    border-radius: 10px;
  }
  .b-table .table {
    font-size: 0.875rem;
    background: transparent !important;
  }
  .b-table .table th {
    font-size: 0.72rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    opacity: 0.75;
    border-bottom-width: 1px !important;
  }
  .b-table .table td,
  .b-table .table th {
    vertical-align: middle;
    padding: 0.6rem 0.9rem;
  }
  .b-table .table td .switch {
    margin: 0;
  }
  .b-table .table tbody tr:hover {
    background: rgba(128, 128, 160, 0.07) !important;
  }
  .b-table .table tfoot td {
    opacity: 0.7;
    font-weight: 600;
  }
  .b-table .table td .button:not([class*="is-primary"]):has(> .icon:only-child) {
    width: 30px;
    height: 30px;
    padding: 0;
    border-radius: 8px;
    background: transparent !important;
    border: 1px solid transparent !important;
    color: inherit;
    opacity: 0.75;
  }
  .b-table .table td .button:has(> .icon:only-child) .icon,
  .b-table .table td .button:has(> .icon:only-child) .mdi::before {
    font-size: 18px;
  }
  .b-table .table td .button:not([class*="is-primary"]):has(> .icon:only-child):hover {
    opacity: 1;
    background: rgba(128, 128, 160, 0.16) !important;
  }
  .b-table .table td .button.is-success:has(> .icon:only-child):hover {
    color: #34d399 !important;
    background: rgba(52, 211, 153, 0.14) !important;
  }
  .b-table .table td > :is(.block, .field):has(> .button) {
    display: flex;
    gap: 0.5rem;
    margin: 0;
  }
  .b-table .table td > :is(.block, .field) > .button {
    margin: 0 !important;
  }
  .b-table .table td .button {
    height: 30px;
    border-radius: 8px;
    font-size: 0.8rem;
  }
  .b-table .table td .button.is-primary.is-outlined {
    background: rgba(124, 108, 240, 0.12) !important;
    border-color: rgba(124, 108, 240, 0.45) !important;
    color: rgb(150, 136, 255) !important;
  }
  .b-table .table td .button.is-danger:has(> .icon:only-child):hover {
    color: #f14668 !important;
    background: rgba(241, 70, 104, 0.14) !important;
  }
</style>
