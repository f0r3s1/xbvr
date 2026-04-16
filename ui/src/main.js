import { createApp, h } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import i18n from './i18n'

import vueDebounce from 'vue-debounce'

import Buefy from '@ntohq/buefy-next'
import '@ntohq/buefy-next/dist/buefy.css'

import 'video.js/dist/video-js.css'
import 'videojs-vr/dist/videojs-vr.css'
import '@mdi/font/css/materialdesignicons.css'

const app = createApp({
  render: () => h(App)
})

app.use(router)
app.use(store)
app.use(i18n)
app.use(Buefy)
app.use(vueDebounce)

app.mount('#app')
