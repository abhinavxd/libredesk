import { createRouter, createWebHistory } from 'vue-router'
import WelcomeView from '../views/WelcomeView.vue'
import ChatView from '../views/ChatView.vue'

const routes = [
  {
    path: '/',
    name: 'welcome',
    component: WelcomeView
  },
  {
    path: '/chat',
    name: 'chat',
    component: ChatView
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
