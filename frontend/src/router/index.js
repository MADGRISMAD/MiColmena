import { createRouter, createWebHistory } from 'vue-router';
import HomeView from '@/views/HomeView.vue';

const routes = [
  {
    path: '/',
    name: 'home',
    component: HomeView
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    // Carga diferida: esta vista se descarga solo cuando se visita.
    component: () => import('@/views/DashboardView.vue')
  }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

export default router;
