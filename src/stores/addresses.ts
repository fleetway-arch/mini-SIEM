import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Address } from '../types'
import { getAddresses, createAddress, deleteAddress } from '../api/client'
import { useToastStore } from './toast'

// Validation des formats IPv4, IPv6 et domaine
export function validateAddress(value: string): string | null {
  const v = value.trim()
  if (!v) return 'Veuillez saisir une adresse'

  // IPv4
  const ipv4 = /^(\d{1,3}\.){3}\d{1,3}$/
  if (ipv4.test(v)) {
    const parts = v.split('.')
    if (parts.every(p => Number(p) >= 0 && Number(p) <= 255)) return null
    return 'Adresse IPv4 invalide'
  }

  // IPv6 (simplifié)
  const ipv6 = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/
  if (ipv6.test(v) && v.includes(':')) return null

  // Domaine
  const domain = /^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
  if (domain.test(v)) return null

  return 'Format invalide (IPv4, IPv6 ou domaine requis)'
}

export const useAddressesStore = defineStore('addresses', () => {
  const addresses = ref<Address[]>([])
  const loading = ref(false)
  const submitting = ref(false)
  const deletingId = ref<number | null>(null)
  const error = ref<string | null>(null)

  async function fetchAddresses() {
    loading.value = true
    error.value = null
    try {
      addresses.value = await getAddresses()
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  async function addAddress(value: string, label?: string) {
    const toast = useToastStore()
    submitting.value = true
    try {
      const addr = await createAddress(value, label)
      addresses.value.push(addr)
      toast.show('Adresse ajoutée avec succès', 'success')
    } catch (e) {
      toast.show((e as Error).message, 'error')
      throw e
    } finally {
      submitting.value = false
    }
  }

  async function removeAddress(id: number) {
    const toast = useToastStore()
    deletingId.value = id
    try {
      await deleteAddress(id)
      addresses.value = addresses.value.filter(a => a.id !== id)
      toast.show('Adresse supprimée', 'success')
    } catch (e) {
      toast.show((e as Error).message, 'error')
      throw e
    } finally {
      deletingId.value = null
    }
  }

  function reset() {
    addresses.value = []
    error.value = null
  }

  return {
    addresses, loading, submitting, deletingId, error,
    fetchAddresses, addAddress, removeAddress, reset,
  }
})
