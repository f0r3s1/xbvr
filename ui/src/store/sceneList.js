import ky from '@/api'

// several components refresh the filters on mount; they share one request
let filtersInFlight = null

const defaultFilterState = {
  dlState: 'available',
  cardSize: localStorage.getItem('sceneCardSize') || '2',  // 1 is now XS and 2 is now S

  lists: [],
  isAvailable: true,
  isAccessible: true,
  isHidden: false,
  isWatched: null,
  releaseMonth: '',
  cast: [],
  sites: [],
  tags: [],
  cuepoint: [],
  attributes: [],
  volume: 0,
  sort: 'release_desc'
}

const pristineFilters = JSON.stringify(defaultFilterState)

const state = {
  items: [],
  // new scenes that belong above what the user is looking at; shown on demand so the page never jumps
  pendingNew: [],
  playlists: [],
  isLoading: false,
  offset: 0,
  total: 0,
  limit: 80,
  counts: {
    any: 0,
    available: 0,
    downloaded: 0,
    not_downloaded: 0,
    hidden: 0
  },
  show_scene_id: '',
  filterOpts: {
    cast: [],
    sites: [],
    tags: [],
    volumes: []
  },
  filters: defaultFilterState
}

const getters = {
  filterQueryParams: (state) => {
    const st = Object.assign({}, state.filters)
    delete st.cardSize

    return btoa(unescape(encodeURIComponent(JSON.stringify(st))))
  },
  getQueryParamsFromObject: () => (payload) => {
    const st = Object.assign({}, JSON.parse(payload))
    delete st.cardSize

    return btoa(unescape(encodeURIComponent(JSON.stringify(st))))
  },
  prevScene: (state) => (currentScene) => {
    const i = state.items.findIndex(item => item.scene_id === currentScene.scene_id)
    if (i === 0) {
      return null
    }
    return state.items[i - 1]
  },
  nextScene: (state) => (currentScene) => {
    const i = state.items.findIndex(item => item.scene_id === currentScene.scene_id)
    if (i === state.items.length - 1) {
      return null
    }
    return state.items[i + 1]
  }
}

const mutations = {
  setItems (state, payload) {
    state.items = payload
  },
  toggleSceneList (state, payload) {
    const idx = state.items.findIndex(obj => obj.scene_id === payload.scene_id)
    if (idx !== -1) {
      const item = state.items[idx]
      if (payload.list === 'watchlist') {
        item.watchlist = !item.watchlist
      }
      if (payload.list === 'favourite') {
        item.favourite = !item.favourite
      }
      if (payload.list == 'watched') {
        item.is_watched = !item.is_watched
      }
      if (payload.list === 'trailerlist') {
        item.trailerlist = !item.trailerlist
      }
      if (payload.list === 'needs_update') {
        item.needs_update = !item.needs_update
      }
      if (payload.list === 'is_hidden') {
        item.is_hidden = !item.is_hidden
      }
      if (payload.list === 'wishlist') {
        item.wishlist = !item.wishlist
      }
    }

    ky.post('/api/scene/toggle', {
      json: {
        scene_id: payload.scene_id,
        list: payload.list
      }
    })
  },
  updateScene (state, payload) {
    const idx = state.items.findIndex(obj => obj.scene_id === payload.scene_id)
    if (idx !== -1) {
      state.items[idx] = payload
    }
  },
  stateFromJSON (state, payload) {
    try {
      const obj = JSON.parse(payload)
      for (const [k, v] of Object.entries(obj)) {
        state.filters[k] = v
      }
    } catch { /* ignore malformed JSON from persisted state */ }
  },
  resetFilters (state) {
    state.filters = { ...JSON.parse(pristineFilters), cardSize: state.filters.cardSize }
  },
  stateFromQuery (state, payload) {
    try {
      state.show_scene_id=payload.scene_id
      const obj = JSON.parse(decodeURIComponent(escape(atob(payload.q))))
      for (const [k, v] of Object.entries(obj)) {
        state.filters[k] = v
      }
    } catch { /* ignore malformed query string */ }
  }
}

let loadSeq = 0

function applyCounts (state, data) {
  state.total = data.results
  state.counts.any = data.count_any
  state.counts.available = data.count_available
  state.counts.downloaded = data.count_downloaded
  state.counts.not_downloaded = data.count_not_downloaded
  state.counts.hidden = data.count_hidden
}

function markFresh (scene) {
  scene.fresh = true
  setTimeout(() => { scene.fresh = false }, 2500)
  return scene
}

// inserts scenes at their sorted position; positions are indexes in the full result, so ascending order keeps them right
function insertScenes (state, entries) {
  for (const { scene, pos } of entries.sort((a, b) => a.pos - b.pos)) {
    state.items.splice(Math.min(pos, state.items.length), 0, markFresh(scene))
    state.offset++
  }
}

const actions = {
  // a websocket "scenes.changed" message: patch the loaded list instead of reloading it
  async applySceneChanges ({ state, dispatch }, { ids, many }) {
    if (!state.items.length) {
      if (state.total === 0) dispatch('load', { offset: 0 })
      return
    }
    if (many || !ids) {
      if (window.scrollY < 300) dispatch('load', { offset: 0 })
      return
    }
    const loaded = new Map(state.items.map((s, i) => [s.id, i]))
    const data = await ky.post('/api/scene/changes', { json: { ...state.filters, limit: state.limit, offset: 0, ids } }).json()
    applyCounts(state, data)
    const byId = new Map((data.scenes || []).map(s => [s.id, s]))

    const removed = ids.filter(id => loaded.has(id) && !(id in data.positions))
    if (removed.length) {
      const gone = new Set(removed)
      state.items = state.items.filter(s => !gone.has(s.id))
      state.offset = Math.max(0, state.offset - removed.length)
    }
    for (const [id, scene] of byId) {
      const i = state.items.findIndex(s => s.id === id)
      if (i !== -1) state.items[i] = scene
    }

    const fresh = []
    for (const [id, scene] of byId) {
      const pos = data.positions[id]
      if (!loaded.has(id) && pos < state.items.length + fresh.length) fresh.push({ scene, pos })
    }
    if (!fresh.length) return
    if (window.scrollY < 300) insertScenes(state, fresh)
    else state.pendingNew.push(...fresh)
  },
  showPendingNew ({ state }) {
    insertScenes(state, state.pendingNew.splice(0))
    window.scrollTo({ top: 0, behavior: 'smooth' })
  },
  filters ({ state }) {
    filtersInFlight ??= Promise.all([
      ky.get('/api/playlist').json(),
      ky.get('/api/scene/filters').json()
    ]).then(([playlists, filterOpts]) => {
      state.playlists = playlists
      if (filterOpts.release_month) filterOpts.release_month.reverse()
      state.filterOpts = filterOpts
    }).finally(() => { filtersInFlight = null })
    return filtersInFlight
  },
  async load ({ state, commit }, params) {
    const iOffset = params.offset || 0
    // Only the latest request may update the list: an older, slower response
    // (e.g. the previous filters) must not overwrite a newer one.
    const seq = ++loadSeq

    state.isLoading = true

    try {
      const q = Object.assign({}, state.filters)
      q.offset = iOffset
      q.limit = state.limit

      const data = await ky.post('/api/scene/list', { json: q }).json()
      if (seq !== loadSeq) return

      if (iOffset === 0) {
        state.pendingNew = []
        commit('setItems', data.scenes || [])
      } else {
        // live inserts can shift pages, so never add a scene twice
        const have = new Set(state.items.map(s => s.id))
        commit('setItems', state.items.concat((data.scenes || []).filter(s => !have.has(s.id))))
      }
      state.offset = iOffset + state.limit
      applyCounts(state, data)
    } finally {
      if (seq === loadSeq) state.isLoading = false
    }
  }
}

export default {
  namespaced: true,
  state,
  getters,
  mutations,
  actions
}
