import Vue from 'vue'
import Router from 'vue-router'

Vue.use(Router)

export default new Router({
  mode: 'hash',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '/',
      name: 'scenes',
      component: () => import('./views/scenes/Scenes')
    },
    {
      path: '/actors',
      name: 'actors',
      component: () => import('./views/actors/Actors')
    },
    {
      path: '/files',
      name: 'files',
      component: () => import('./views/files/Files')
    },
    {
      path: '/options',
      name: 'options',
      component: () => import('./views/options/Options')
    }
  ]
})
