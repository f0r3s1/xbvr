import ky from '@/api'

function defaultValue (v, d) {
  if (v === undefined) {
    return d
  }
  return v
}

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

const state = {
  items: [],
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
  getQueryParamsFromObject: (state) => (payload) => {
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
    } catch (err) {
    }
  },
  stateFromQuery (state, payload) {
    try {
      state.show_scene_id=payload.scene_id
      const obj = JSON.parse(decodeURIComponent(escape(atob(payload.q))))
      for (const [k, v] of Object.entries(obj)) {
        state.filters[k] = v
      }
    } catch (err) {
    }
  }
}

const actions = {
  async filters ({ state }) {
    state.playlists = await ky.get('/api/playlist').json()
    state.filterOpts = await ky.get('/api/scene/filters').json()

    if (state.filterOpts.release_month) {
      state.filterOpts.release_month = state.filterOpts.release_month.reverse()
    }
  },
  async load ({ state, getters, commit }, params) {
    const iOffset = params.offset || 0

    state.isLoading = true

    try {
      const q = Object.assign({}, state.filters)
      q.offset = iOffset
      q.limit = state.limit

      const data = await ky.post('/api/scene/list', { json: q }).json()

      if (iOffset === 0) {
        commit('setItems', data.scenes || [])
      } else {
        commit('setItems', state.items.concat(data.scenes || []))
      }
      state.offset = iOffset + state.limit
      state.total = data.results

      state.counts.any = data.count_any
      state.counts.available = data.count_available
      state.counts.downloaded = data.count_downloaded
      state.counts.not_downloaded = data.count_not_downloaded
      state.counts.hidden = data.count_hidden
    } finally {
      state.isLoading = false
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
