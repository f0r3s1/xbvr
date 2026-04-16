import ky from '@/api'

const state = {
  items: []
}

const mutations = {
  setItems (state, items) {
    state.items = items
  }
}

const actions = {
  async load ({ commit }) {
    try {
      const items = await ky.get('/api/options/sites', { timeout: 60000 }).json()
      commit('setItems', items)
    } catch (e) {
      console.error('Failed to load sites:', e)
    }
  },
  async toggleSite ({ commit }, params) {
    const items = await ky.put(`/api/options/sites/${params.id}`, { json: {}, timeout: 60000 }).json()
    commit('setItems', items)
  },
  async toggleSubscribed ({ commit }, params) {
    const items = await ky.put(`/api/options/sites/subscribed/${params.id}`, { json: {}, timeout: 60000 }).json()
    commit('setItems', items)
  },
  async toggleLimitScraping ({ commit }, params) {
    const items = await ky.put(`/api/options/sites/limit_scraping/${params.id}`, { json: {}, timeout: 60000 }).json()
    commit('setItems', items)
  },
  async toggleScrapeStash ({ commit }, params) {
    const items = await ky.put(`/api/options/sites/scrape_stash/${params.id}`, { json: {}, timeout: 60000 }).json()
    commit('setItems', items)
  },
  async toggleUseFlareSolverr ({ commit }, params) {
    const items = await ky.put(`/api/options/sites/use_flaresolverr/${params.id}`, { json: {}, timeout: 60000 }).json()
    commit('setItems', items)
  },
  async toggleUseProxy ({ commit }, params) {
    const items = await ky.put(`/api/options/sites/use_proxy/${params.id}`, { json: {}, timeout: 60000 }).json()
    commit('setItems', items)
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
