import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import type { Role } from '../types'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
    { path: '/signup', name: 'signup', component: () => import('../views/SignupView.vue'), meta: { public: true } },
    { path: '/forgot-password', name: 'forgot-password', component: () => import('../views/ForgotPasswordView.vue'), meta: { public: true } },
    { path: '/reset-password/:token', name: 'reset-password', component: () => import('../views/ResetPasswordView.vue'), meta: { public: true } },
    { path: '/invite/:token', name: 'invite', component: () => import('../views/InviteView.vue'), meta: { public: true } },
    { path: '/', name: 'dashboard', component: () => import('../views/DashboardView.vue') },
    { path: '/events', name: 'events', component: () => import('../views/EventsView.vue') },
    { path: '/alerts', name: 'alerts', component: () => import('../views/AlertsView.vue') },
    { path: '/addresses', name: 'addresses', component: () => import('../views/AddressesView.vue') },
    {
      path: '/team',
      name: 'team',
      component: () => import('../views/TeamView.vue'),
      meta: { roles: ['owner', 'admin'] as Role[] },
    },
  ],
})

let initPromise: Promise<void> | null = null

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const toastStore = useToastStore()

  if (!initPromise) {
    initPromise = auth.init()
  }
  await initPromise

  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if ((to.name === 'login' || to.name === 'signup') && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }

  // Role guard
  if (to.meta.roles && auth.user) {
    const allowed = to.meta.roles as Role[]
    if (!allowed.includes(auth.user.role)) {
      toastStore.show('Vous n\'avez pas accès à cette page', 'error')
      return { name: 'dashboard' }
    }
  }
})

export default router
