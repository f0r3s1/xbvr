<template>
  <div>
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keypress.prevent.questionMark="$store.commit('overlay/showQuickFind')"
    />
    <Navbar/>
    <div class="navbar-pad">
      <router-view/>
    </div>

    <Details v-if="showOverlay"/>
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
import GlobalEvents from 'vue-global-events'

import Navbar from './Navbar.vue'
import Socket from './Socket.vue'
import QuickFind from './QuickFind'
import Details from './views/scenes/Details'
import EditScene from './views/scenes/EditScene'
import ActorDetails from './views/actors/ActorDetails'
import EditActor from './views/actors/EditActor'
import SearchStashdbScenes from './views/scenes/SearchStashdbScenes'
import SearchStashdbActors from './views/actors/SearchStashdbActors'
import MigrationOverlay from './components/MigrationOverlay'

export default {
  components: { Navbar, Socket, QuickFind, GlobalEvents, Details, EditScene, ActorDetails, EditActor, SearchStashdbScenes, SearchStashdbActors, MigrationOverlay },
  mounted () {
    // Apply saved theme immediately from localStorage to avoid flash on load
    const saved = localStorage.getItem('xbvr-theme') || 'auto'
    this.applyTheme(saved)

    // Listen for OS preference changes (for auto mode)
    this._darkMQ = window.matchMedia('(prefers-color-scheme: dark)')
    this._darkMQHandler = () => {
      if ((this.$store.state.optionsWeb.web.theme || 'auto') === 'auto') {
        this.applyTheme('auto')
      }
    }
    this._darkMQ.addEventListener('change', this._darkMQHandler)

    // Load from server — watch will apply if different from localStorage
    this.$store.dispatch('optionsWeb/load')
  },
  beforeDestroy () {
    if (this._darkMQ) this._darkMQ.removeEventListener('change', this._darkMQHandler)
  },
  watch: {
    '$store.state.optionsWeb.web.theme' (theme) {
      this.applyTheme(theme || 'auto')
    }
  },
  methods: {
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
  }
}
</script>

<style>
  html.has-navbar-fixed-top,
  body.has-navbar-fixed-top {
    padding-top: 0 !important;
  }
  html {
    padding-top: 3.25rem !important;
  }
  .navbar-pad {
    margin-top: 1rem;
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
  html[data-theme="dark"] .navbar-item:hover,
  html[data-theme="dark"] .navbar-link:hover {
    background-color: #222230 !important;
    color: #fff !important;
  }
  html[data-theme="dark"] .navbar-item.is-active,
  html[data-theme="dark"] .navbar-item.router-link-exact-active {
    background-color: #222230 !important;
    color: #fff !important;
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

  /* Medium screens: properties buttons icon-only when sidebar is narrow */
  @media screen and (max-width: 1215px) {
    .columns.is-multiline.is-gapless > .column.is-half .b-checkbox.button span:not(.icon) {
      display: none !important;
    }
  }
</style>
