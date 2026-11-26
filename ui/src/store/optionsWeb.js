import ky, { getOptionsState } from '@/api'
const state = {
  loading: false,
  web: {
    tagSort: 'By Tag Count',
    sceneHidden: true,
    sceneWatchlist: true,
    sceneFavourite: true,
    sceneWishlist: true,
    sceneWatched: false,
    sceneEdit: false,
    sceneDuration: false,
    sceneScriptSpeed: false,
    sceneCuepoint: true,
    showHspFile: true,
    showSubtitlesFile: true,
    sceneTrailerlist: true,
    isAvailOpacity: 40,
    showScriptHeatmap: false,
    showAllHeatmaps: false,
    showOpenInNewWindow: true,
    sceneCardAspectRatio: "16:9",
    sceneCardScaleToFit: false,
    cardButtonStyle: 'classic',
    actorCardAspectRatio: "1:1",
    actorCardScaleToFit: true,
    showSiteLogo: true,
    dateFormat: 'yyyy-MM-dd',
    accentColorTags: true,
    updateCheck: true,
    theme: 'auto',
    showUpcoming: true,
    upcomingDays: 14
  }
}

const mutations = {}

const actions = {
  async load ({ state }) {
    state.loading = true
    return getOptionsState()
      .then(data => {
        state.web.tagSort = data.config.web.tagSort
        state.web.sceneHidden = data.config.web.sceneHidden
        state.web.sceneWatchlist = data.config.web.sceneWatchlist
        state.web.sceneFavourite = data.config.web.sceneFavourite
        state.web.sceneWishlist = data.config.web.sceneWishlist
        state.web.sceneWatched = data.config.web.sceneWatched
        state.web.sceneEdit = data.config.web.sceneEdit
        state.web.sceneDuration = data.config.web.sceneDuration
        state.web.sceneScriptSpeed = data.config.web.sceneScriptSpeed
        state.web.sceneCuepoint = data.config.web.sceneCuepoint
        state.web.showHspFile = data.config.web.showHspFile
        state.web.showSubtitlesFile = data.config.web.showSubtitlesFile
        state.web.sceneTrailerlist = data.config.web.sceneTrailerlist
        state.web.showScriptHeatmap = data.config.web.showScriptHeatmap
        state.web.showAllHeatmaps = data.config.web.showAllHeatmaps
        state.web.updateCheck = data.config.web.updateCheck
        state.web.isAvailOpacity = data.config.web.isAvailOpacity
        state.web.showOpenInNewWindow = data.config.web.showOpenInNewWindow
        state.web.sceneCardAspectRatio = data.config.web.sceneCardAspectRatio
        state.web.sceneCardScaleToFit = data.config.web.sceneCardScaleToFit
        state.web.cardButtonStyle = data.config.web.cardButtonStyle || 'classic'
        state.web.actorCardAspectRatio = data.config.web.actorCardAspectRatio
        state.web.actorCardScaleToFit = data.config.web.actorCardScaleToFit
        state.web.showSiteLogo = data.config.web.showSiteLogo !== false
        state.web.dateFormat = data.config.web.dateFormat || 'yyyy-MM-dd'
        state.web.accentColorTags = data.config.web.accentColorTags !== false
        state.web.showUpcoming = data.config.web.showUpcoming !== false
        state.web.upcomingDays = data.config.web.upcomingDays ?? 14
        const theme = data.config.web.theme || 'auto'
        state.web.theme = theme
        localStorage.setItem('xbvr-theme', theme)
        state.loading = false
      })
  },
  async save ({ state }) {
    state.loading = true
    ky.put('/api/options/interface/web', { json: { ...state.web } })
      .json()
      .then(data => {
        state.web.tagSort = data.tagSort
        state.web.sceneHidden = data.sceneHidden
        state.web.sceneWatchlist = data.sceneWatchlist
        state.web.sceneFavourite = data.sceneFavourite
        state.web.sceneWishlist = data.sceneWishlist
        state.web.sceneWatched = data.sceneWatched
        state.web.sceneEdit = data.sceneEdit
        state.web.sceneDuration = data.sceneDuration
        state.web.sceneScriptSpeed = data.sceneScriptSpeed
        state.web.sceneCuepoint = data.sceneCuepoint
        state.web.showHspFile = data.showHspFile
        state.web.showSubtitlesFile = data.showSubtitlesFile
        state.web.sceneTrailerlist = data.sceneTrailerlist
        state.web.showScriptHeatmap = data.showScriptHeatmap
        state.web.showAllHeatmaps = data.showAllHeatmaps
        state.web.updateCheck = data.updateCheck
        state.web.isAvailOpacity = data.isAvailOpacity
        state.web.showOpenInNewWindow = data.showOpenInNewWindow
        state.web.sceneCardAspectRatio = data.sceneCardAspectRatio
        state.web.sceneCardScaleToFit = data.sceneCardScaleToFit
        state.web.cardButtonStyle = data.cardButtonStyle || 'classic'
        state.web.actorCardAspectRatio = data.actorCardAspectRatio
        state.web.actorCardScaleToFit = data.actorCardScaleToFit
        state.web.showSiteLogo = data.showSiteLogo !== false
        state.web.dateFormat = data.dateFormat || 'yyyy-MM-dd'
        state.web.accentColorTags = data.accentColorTags !== false
        state.web.showUpcoming = data.showUpcoming !== false
        state.web.upcomingDays = data.upcomingDays ?? 14
        const theme = data.theme || 'auto'
        state.web.theme = theme
        localStorage.setItem('xbvr-theme', theme)
        state.loading = false
      })
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
