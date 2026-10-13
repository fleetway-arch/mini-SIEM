import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import * as api from '../api/client'
import { useAuthStore } from './auth'
import { useEventsStore } from './events'
import { useAlertsStore } from './alerts'
import { useDashboardStore } from './dashboard'
import { useAddressesStore } from './addresses'

vi.mock('../api/client', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  signup: vi.fn(),
  acceptInvitation: vi.fn(),
  getCurrentUser: vi.fn(),
  getAddresses: vi.fn(),
  getToken: vi.fn(),
  storeToken: vi.fn(),
  refreshSession: vi.fn(),
  setUnauthorizedHandler: vi.fn(),
  setSessionRefreshedHandler: vi.fn(),
}))

const mocked = vi.mocked(api)
const user = { id: 1, email: 'owner@acme.test', companyId: 10, companyName: 'Acme', role: 'owner' as const }

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
  mocked.getAddresses.mockResolvedValue([{ id: 1, value: '10.0.0.1', createdAt: '' }])
})

function seedOtherStores() {
  useEventsStore().events = [{ id: 1, timestamp: '', source: 'sshd', severity: 'info', ip: '1.1.1.1', message: 'x', rawLog: 'x' }]
  useAlertsStore().alerts = [{ id: 1, createdAt: '', rule: 'r', severity: 'critical', ip: '1.1.1.1', description: 'd', eventCount: 1, status: 'open' }]
  useDashboardStore().topIps = [{ ip: '1.1.1.1', count: 1, failedCount: 0 }]
  useAddressesStore().addresses = [{ id: 1, value: '10.0.0.1', createdAt: '' }]
}

function expectOtherStoresCleared() {
  expect(useEventsStore().events).toEqual([])
  expect(useAlertsStore().alerts).toEqual([])
  expect(useDashboardStore().topIps).toEqual([])
  expect(useAddressesStore().addresses).toEqual([])
}

describe('auth store', () => {
  it('login stores the session and loads addresses', async () => {
    mocked.login.mockResolvedValue({ token: 'access-1', user })
    const store = useAuthStore()

    await store.login('owner@acme.test', 'Str0ng!Passw0rd')

    expect(store.isAuthenticated).toBe(true)
    expect(store.user).toEqual(user)
    expect(mocked.storeToken).toHaveBeenCalledWith('access-1')
    expect(store.addresses).toHaveLength(1)
    expect(store.loading).toBe(false)
  })

  it('login failure exposes the error and resets loading', async () => {
    mocked.login.mockRejectedValue(new Error('invalid credentials'))
    const store = useAuthStore()

    await expect(store.login('owner@acme.test', 'nope')).rejects.toThrow('invalid credentials')
    expect(store.error).toBe('invalid credentials')
    expect(store.loading).toBe(false)
    expect(store.isAuthenticated).toBe(false)
  })

  it('logout revokes the session server-side and clears every store', async () => {
    mocked.login.mockResolvedValue({ token: 'access-1', user })
    mocked.logout.mockResolvedValue(undefined)
    const store = useAuthStore()
    await store.login('owner@acme.test', 'x')
    seedOtherStores()

    await store.logout()

    expect(mocked.logout).toHaveBeenCalledOnce()
    expect(store.isAuthenticated).toBe(false)
    expect(store.addresses).toEqual([])
    expect(mocked.storeToken).toHaveBeenLastCalledWith(null)
    expectOtherStoresCleared()
  })

  it('logout still clears local state when the server is unreachable', async () => {
    mocked.logout.mockRejectedValue(new Error('network down'))
    const store = useAuthStore()
    store.setSession('access-1', user)
    seedOtherStores()

    await store.logout()

    expect(store.isAuthenticated).toBe(false)
    expectOtherStoresCleared()
  })

  it('the 401 handler registered with the client clears all stores', () => {
    const store = useAuthStore()
    store.setSession('access-1', user)
    seedOtherStores()
    const handler = mocked.setUnauthorizedHandler.mock.calls[0][0]

    handler()

    expect(store.isAuthenticated).toBe(false)
    expectOtherStoresCleared()
  })

  it('init restores a session from the stored access token', async () => {
    mocked.getToken.mockReturnValue('access-1')
    mocked.getCurrentUser.mockResolvedValue(user)
    const store = useAuthStore()

    await store.init()

    expect(mocked.refreshSession).not.toHaveBeenCalled()
    expect(store.isAuthenticated).toBe(true)
  })

  it('init uses the refresh cookie when the tab has no access token', async () => {
    mocked.getToken.mockReturnValueOnce(null).mockReturnValue('access-2')
    mocked.refreshSession.mockResolvedValue(true)
    mocked.getCurrentUser.mockResolvedValue(user)
    const store = useAuthStore()

    await store.init()

    expect(mocked.refreshSession).toHaveBeenCalledOnce()
    expect(store.token).toBe('access-2')
    expect(store.isAuthenticated).toBe(true)
  })

  it('init stays logged out when there is no session at all', async () => {
    mocked.getToken.mockReturnValue(null)
    mocked.refreshSession.mockResolvedValue(false)
    const store = useAuthStore()

    await store.init()

    expect(mocked.getCurrentUser).not.toHaveBeenCalled()
    expect(store.isAuthenticated).toBe(false)
  })

  it('a silent refresh updates the token and user', () => {
    const store = useAuthStore()
    const handler = mocked.setSessionRefreshedHandler.mock.calls[0][0]

    handler({ token: 'access-3', user: { ...user, role: 'admin' } })

    expect(store.token).toBe('access-3')
    expect(store.user?.role).toBe('admin')
  })
})
