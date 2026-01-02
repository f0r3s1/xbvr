import ky from 'ky'

const state = {
  loading: false,
  advanced: {
    showInternalSceneId: false,
    showHSPApiLink: false,
    showSceneSearchField: false,
    stashApiKey: '',
    scrapeActorAfterScene: 'true',
    useImperialEntry: 'false',
    linkScenesAfterSceneScraping: true,
    useAltSrcInFileMatching: true,
    useAltSrcInScriptFilters: true,
    ignoreReleasedBefore: null,
    collectorConfigs: null,
    flareSolverrAddress: '',
    useFlareSolverr: false,
    proxyAddress: '',
    proxyApiKeyName: '',
    proxyApiKeyValue: ''
  }
}

const mutations = {}

const actions = {
  async load ({ state }) {
    state.loading = true
    ky.get('/api/options/collector-config-list')
      .json()
      .then(data => {
        state.advanced.collectorConfigs = data
      })
    ky.get('/api/options/state')
      .json()
      .then(data => {
        state.advanced.showInternalSceneId = data.config.advanced.showInternalSceneId
        state.advanced.showHSPApiLink = data.config.advanced.showHSPApiLink
        state.advanced.showSceneSearchField = data.config.advanced.showSceneSearchField
        state.advanced.stashApiKey = data.config.advanced.stashApiKey
        state.advanced.scraperProxy = data.config.advanced.scraperProxy
        state.advanced.scrapeActorAfterScene = data.config.advanced.scrapeActorAfterScene
        state.advanced.useImperialEntry = data.config.advanced.useImperialEntry
        state.advanced.linkScenesAfterSceneScraping = data.config.advanced.linkScenesAfterSceneScraping
        state.advanced.useAltSrcInFileMatching = data.config.advanced.useAltSrcInFileMatching
        state.advanced.useAltSrcInScriptFilters = data.config.advanced.useAltSrcInScriptFilters
        state.advanced.ignoreReleasedBefore = data.config.advanced.ignoreReleasedBefore
        state.advanced.flareSolverrAddress = data.config.advanced.flareSolverrAddress
        state.advanced.useFlareSolverr = data.config.advanced.useFlareSolverr
        state.advanced.proxyAddress = data.config.advanced.proxyAddress
        state.advanced.proxyApiKeyName = data.config.advanced.proxyApiKeyName
        state.advanced.proxyApiKeyValue = data.config.advanced.proxyApiKeyValue
        state.loading = false
      })
  },
  async save ({ state }) {
    state.loading = true
    ky.put('/api/options/interface/advanced', { json: { ...state.advanced } })
      .json()
      .then(data => {
        state.advanced.showInternalSceneId = data.showInternalSceneId
        state.advanced.showHSPApiLink = data.showHSPApiLink
        state.advanced.showSceneSearchField = data.showSceneSearchField
        state.advanced.stashApiKey = data.stashApiKey
        state.advanced.scraperProxy = data.scraperProxy
        state.advanced.scrapeActorAfterScene = data.scrapeActorAfterScene
        state.advanced.useImperialEntry = data.useImperialEntry
        state.advanced.linkScenesAfterSceneScraping = data.linkScenesAfterSceneScraping
        state.advanced.useAltSrcInFileMatching = data.useAltSrcInFileMatching
        state.advanced.useAltSrcInScriptFilters = data.useAltSrcInScriptFilters
        state.advanced.ignoreReleasedBefore = data.ignoreReleasedBefore
        state.advanced.flareSolverrAddress = data.flareSolverrAddress
        state.advanced.useFlareSolverr = data.useFlareSolverr
        state.advanced.proxyAddress = data.proxyAddress
        state.advanced.proxyApiKeyName = data.proxyApiKeyName
        state.advanced.proxyApiKeyValue = data.proxyApiKeyValue
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
