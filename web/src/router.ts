import { createRouter, createWebHistory } from 'vue-router'
import GroceryView from './views/GroceryView.vue'
import TasksView from './views/TasksView.vue'
import SettingsView from './views/SettingsView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'grocery', component: GroceryView },
    { path: '/attivita', name: 'tasks', component: TasksView },
    { path: '/impostazioni', name: 'settings', component: SettingsView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})
