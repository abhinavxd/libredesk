import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@widget/layouts/MainLayout.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
  },

]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
