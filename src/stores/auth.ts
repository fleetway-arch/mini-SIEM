import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, Address } from '../types'
import {
  login as apiLogin, getCurrentUser, getAddresses,
  signup as apiSignup, acceptInvitation as apiAcceptInvitation,
} from '../api/client'
import { setUnauthorizedHandler } from '../api/client'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const user = ref<User | null>(null)
  const addresses = ref<Address[]>([])
  const selectedAddress = ref<string>('')
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value && !!user.value)

  async function init() {
    const stored = sessionStorage.getItem('siem_token')
    if (!stored) return
    token.value = stored
    try {
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
    sessionStorage.setItem('siem_token', t)
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

  function clearSession() {
    token.value = null
    user.value = null
    addresses.value = []
    selectedAddress.value = ''
    sessionStorage.removeItem('siem_token')
  }

  async function logout() {
    clearSession()
  }

  setUnauthorizedHandler(() => {
    clearSession()
  })

  return {
    token, user, addresses, selectedAddress,
    loading, error, isAuthenticated,
    init, login, signup, acceptInvitation, logout,
    fetchAddresses, setSelectedAddress, clearSession, setSession,
  }
})
