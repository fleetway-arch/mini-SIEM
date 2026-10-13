import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import type { Role } from '../types'

vi.mock('../api/client', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  signup: vi.fn(),
  acceptInvitation: vi.fn(),
  getCurrentUser: vi.fn(),
  getAddresses: vi.fn().mockResolvedValue([]),
  getToken: vi.fn(),
  storeToken: vi.fn(),
  refreshSession: vi.fn().mockResolvedValue(false),
  setUnauthorizedHandler: vi.fn(),
  setSessionRefreshedHandler: vi.fn(),
}))

// Route components are irrelevant to the guards and pull in Chart.js.
vi.mock('../views/DashboardView.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../views/TeamView.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../views/EventsView.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../views/LoginView.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../views/ForgotPasswordView.vue', () => ({ default: { template: '<div />' } }))

// The router caches auth.init() in module state: load a fresh router per test.
async function loadRouter(role: Role | null) {
  vi.resetModules()
  setActivePinia(createPinia())
  const api = await import('../api/client')
  vi.mocked(api.getToken).mockReturnValue(role ? 'access-token' : null)
  if (role) {
    vi.mocked(api.getCurrentUser).mockResolvedValue({ id: 1, email: 'u@acme.test', companyId: 1, companyName: 'Acme', role })
  }
  const { default: router } = await import('./index')
  const { useToastStore } = await import('../stores/toast')
  return { router, toasts: useToastStore() }
}

describe('route guards', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/')
  })

  it('sends anonymous users to /login and keeps the target as redirect', async () => {
    const { router } = await loadRouter(null)
    await router.push('/events')
    expect(router.currentRoute.value.name).toBe('login')
    expect(router.currentRoute.value.query.redirect).toBe('/events')
  })

  it('lets anonymous users open public pages', async () => {
    const { router } = await loadRouter(null)
    await router.push('/forgot-password')
    expect(router.currentRoute.value.name).toBe('forgot-password')
  })

  it('redirects authenticated users away from /login', async () => {
    const { router } = await loadRouter('viewer')
    await router.push('/login')
    expect(router.currentRoute.value.name).toBe('dashboard')
  })

  it('blocks viewers from /team with a toast', async () => {
    const { router, toasts } = await loadRouter('viewer')
    await router.push('/team')
    expect(router.currentRoute.value.name).toBe('dashboard')
    expect(toasts.toasts[toasts.toasts.length - 1]?.type).toBe('error')
  })

  it.each(['owner', 'admin'] as Role[])('allows %s on /team', async (role) => {
    const { router } = await loadRouter(role)
    await router.push('/team')
    expect(router.currentRoute.value.name).toBe('team')
  })
})
