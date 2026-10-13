import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, Address } from '../types'
import {
  login as apiLogin, logout as apiLogout, getCurrentUser, getAddresses,
  signup as apiSignup, acceptInvitation as apiAcceptInvitation,
  getToken, storeToken, refreshSession,
  setUnauthorizedHandler, setSessionRefreshedHandler,
} from '../api/client'
import { useDashboardStore } from './dashboard'
import { useEventsStore } from './events'
import { useAlertsStore } from './alerts'
import { useAddressesStore } from './addresses'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const user = ref<User | null>(null)
  const addresses = ref<Address[]>([])
  const selectedAddress = ref<string>('')
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value && !!user.value)

  // Restores the session on page load. A new tab has no access token in
  // sessionStorage, so it silently asks for one with the refresh cookie.
  async function init() {
    try {
      if (!getToken() && !(await refreshSession())) return
      token.value = getToken()
      user.value = await getCurrentUser()
      await fetchAddresses()
    } catch {
      clearSession()
    }
  }

  async function fetchAddresses() {
    try {
      addresses.value = await getAddresses()
    } catch {
      addresses.value = []
    }
  }

  function setSession(t: string, u: User) {
    token.value = t
    user.value = u
    storeToken(t)
  }

  async function login(email: string, password: string) {
    loading.value = true
    error.value = null
    try {
      const res = await apiLogin(email, password)
      setSession(res.token, res.user)
      await fetchAddresses()
    } catch (e) {
      error.value = (e as Error).message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function signup(payload: Parameters<typeof apiSignup>[0]) {
    loading.value = true
    error.value = null
    try {
      const res = await apiSignup(payload)
      setSession(res.token, res.user)
      await fetchAddresses()
    } catch (e) {
      error.value = (e as Error).message
      throw e
    } finally {
      loading.value = false
    }
  }

  async function acceptInvitation(token: string, fullName: string, password: string) {
    loading.value = true
    error.value = null
    try {
      const res = await apiAcceptInvitation(token, fullName, password)
      setSession(res.token, res.user)
      await fetchAddresses()
    } catch (e) {
      error.value = (e as Error).message
      throw e
    } finally {
      loading.value = false
    }
  }

  function setSelectedAddress(addr: string) {
    selectedAddress.value = addr
  }

  // Clears every store so no data leaks to the next user of the browser.
  function clearSession() {
    token.value = null
    user.value = null
    addresses.value = []
    selectedAddress.value = ''
    storeToken(null)
    useDashboardStore().reset()
    useEventsStore().reset()
    useAlertsStore().reset()
    useAddressesStore().reset()
  }

  // Revokes the refresh token server-side; local state is cleared even if
  // the server cannot be reached.
  async function logout() {
    try {
      await apiLogout()
    } catch {
      // ignored: the session is cleared locally below
    } finally {
      clearSession()
    }
  }

  setUnauthorizedHandler(() => {
    clearSession()
  })

  setSessionRefreshedHandler((session) => {
    token.value = session.token
    user.value = session.user
  })

  return {
    token, user, addresses, selectedAddress,
    loading, error, isAuthenticated,
    init, login, signup, acceptInvitation, logout,
    fetchAddresses, setSelectedAddress, clearSession, setSession,
  }
})
