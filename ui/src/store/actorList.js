import ky from '@/api'

let filtersInFlight = null

const defaultFilterState = {
  dlState: 'available',
  cardSize: localStorage.getItem('actorCardSize') || '2',  // 1 is now XS and 2 is now S

  lists: [],
  cast: [],
  sites: [],
  tags: [],
  attributes: [],
  jumpTo: '',
  min_age: 0,
  max_age: 100,  
  min_height: 120,
  max_height: 220,  
  min_weight: 25,
  max_weight: 150,  
  min_count: 0,
  max_count: 150,  
  min_avail: 0,
  max_avail: 150,  
  min_rating: 0,
  max_rating: 5,
  min_scene_rating: 0,
  max_scene_rating: 5,
  sort: 'scene_available_desc'
}

const state = {
  actors: [],
  playlists: [],
  isLoading: false,
  offset: 0,
  total: 0,
  limit: 36,
  show_actor_id: '',
  filterOpts: {
    cast: [],
    sites: [],
    tags: []
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
  prevActor: (state) => (currentActor) => {
    const i = state.actors.findIndex(actor => actor.id === currentActor.id)
    if (i === 0) {
      return null
    }
    return state.actors[i - 1]
  },
  nextActor: (state) => (currentActor) => {
    const i = state.actors.findIndex(actor => actor.id === currentActor.id)
    if (i === state.actors.length - 1) {
      return null
    }
    return state.actors[i + 1]
  },
  firstActor: (state) => () => {    
    return state.actors[0]
  },
  lastActor: (state) => () => {    
    return state.actors[state.actors.length-1]
  }
}

const mutations = {
  setActors (state, payload) {
    state.actors = payload
  },
  toggleActorList (state, payload) {
    const idx = state.actors.findIndex(obj => obj.actor_id === payload.actor_id)
    if (idx !== -1) {
      const item = state.actors[idx]
      if (payload.list === 'watchlist') {
        item.watchlist = !item.watchlist
      }
      if (payload.list === 'favourite') {
        item.favourite = !item.favourite
      }
      if (payload.list === 'needs_update') {
        item.needs_update = !item.needs_update
      }
    }

    ky.post('/api/actor/toggle', {
      json: {
        actor_id: payload.actor_id,
        list: payload.list
      }
    })
  },
  updateActor (state, payload) {
    const idx = state.actors.findIndex(obj => obj.id === payload.id)
    if (idx !== -1) {
      state.actors[idx] = payload
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
  stateFromQuery (state, payload) {
    try {
      state.show_actor_id=payload.actor_id
      const obj = JSON.parse(decodeURIComponent(escape(atob(payload.q))))
      for (const [k, v] of Object.entries(obj)) {
        state.filters[k] = v
      }
    } catch { /* ignore malformed query string */ }
  }
}

let loadSeq = 0

const actions = {
  filters ({ state }) {
    filtersInFlight ??= Promise.all([
      ky.get('/api/playlist/actor', { timeout: 30000 }).json(),
      ky.get('/api/actor/filters', { timeout: 30000 }).json()
    ]).then(([playlists, filterOpts]) => {
      state.playlists = playlists
      state.filterOpts = filterOpts
    }).catch(e => console.error('Failed to load actor filters:', e))
      .finally(() => { filtersInFlight = null })
    return filtersInFlight
  },
  // Lazy list: offset 0 replaces the list, the current end appends the next batch,
  // and any other offset (callers refreshing "the current page" after an edit)
  // reloads everything already on screen in place.
  async load ({ state, commit }, params) {
    let iOffset = params.offset || 0
    let limit = state.limit
    const append = iOffset > 0 && iOffset >= state.actors.length
    if (append) {
      iOffset = state.actors.length
    } else if (iOffset > 0) {
      limit = Math.max(state.limit, state.actors.length)
      iOffset = 0
    }
    // Only the latest request may update the list (see sceneList.js).
    const seq = ++loadSeq

    state.isLoading = true

    try {
      const q = Object.assign({}, state.filters)
      q.offset = iOffset
      q.limit = limit

      const data = await ky
        .post('/api/actor/list', { json: q, timeout: 30000 })
        .json()
      if (seq !== loadSeq) return

      if (append) {
        commit('setActors', state.actors.concat(data.actors || []))
      } else {
        commit('setActors', data.actors || [])
      }
      state.offset = state.actors.length
      state.total = data.results
    } catch (e) {
      console.error('Failed to load actors:', e)
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
