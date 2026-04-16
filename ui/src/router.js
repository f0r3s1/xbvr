import { createRouter, createWebHashHistory } from 'vue-router'

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'scenes',
      component: () => import('./views/scenes/Scenes.vue')
    },
    {
      path: '/actors',
      name: 'actors',
      component: () => import('./views/actors/Actors.vue')
    },
    {
      path: '/files',
      name: 'files',
      component: () => import('./views/files/Files.vue')
    },
    {
      path: '/options',
      name: 'options',
      component: () => import('./views/options/Options.vue')
    }
  ]
})
