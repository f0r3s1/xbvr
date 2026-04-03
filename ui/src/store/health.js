const state = {
  progress: {
    running: false,
    step: '',
    step_num: 0,
    total_steps: 15,
    percent: 0
  }
}

const mutations = {
  setProgress (state, payload) {
    state.progress = payload
  }
}

export default {
  namespaced: true,
  state,
  mutations
}
