const state = {
  progress: {
    running: false,
    step: '',
    step_num: 0,
    total_steps: 16,
    percent: 0
  },
  // critical + warning issues in the latest report (badge in the Options menu)
  attention: 0
}

const mutations = {
  setProgress (state, payload) {
    state.progress = payload
  },
  setReport (state, report) {
    const s = (report && report.summary) || {}
    state.attention = (s.critical || 0) + (s.warning || 0)
  }
}

export default {
  namespaced: true,
  state,
  mutations
}
