import { createApp, h } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import i18n from './i18n'

import vueDebounce from 'vue-debounce'

import Buefy from '@ntohq/buefy-next'
import '@ntohq/buefy-next/dist/buefy.css'

import '@mdi/font/css/materialdesignicons.css'

const app = createApp({
  render: () => h(App)
})

// Buefy wrappers keep their id on the outer element; this puts it on the real control so labels link to it.
app.directive('field-id', {
  mounted (el, binding) {
    const control = el.matches('input, select, textarea') ? el : el.querySelector('input, select, textarea')
    if (control) control.id = binding.value
  }
})
app.use(router)
app.use(store)
app.use(i18n)
app.use(Buefy)
app.directive('debounce', vueDebounce({ lock: true }))

app.mount('#app')
