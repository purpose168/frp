import { createRouter, createWebHashHistory } from 'vue-router'
import Overview from '../views/Overview.vue'
import ClientConfigure from '../views/ClientConfigure.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'Overview',
      component: Overview,
      meta: { title: '概览' },
    },
    {
      path: '/configure',
      name: 'ClientConfigure',
      component: ClientConfigure,
      meta: { title: '客户端配置' },
    },
  ],
})

export default router
