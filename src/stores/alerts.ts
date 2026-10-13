import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Alert, AlertStatus } from '../types'
import { getAlerts, patchAlert } from '../api/client'
import { useToastStore } from './toast'

export const useAlertsStore = defineStore('alerts', () => {
  const alerts = ref<Alert[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const statusFilter = ref<AlertStatus | ''>('')
  const updatingIds = ref<Set<number>>(new Set())

  async function fetchAlerts(address?: string) {
    loading.value = true
    error.value = null
    try {
      alerts.value = await getAlerts(statusFilter.value, address)
    } catch (e) {
      error.value = (e as Error).message
      alerts.value = []
    } finally {
      loading.value = false
    }
  }

  function setStatusFilter(s: AlertStatus | '', address?: string) {
    statusFilter.value = s
    fetchAlerts(address)
  }

  // Mise à jour optimiste : on change le statut localement avant la réponse API
  async function updateAlertStatus(id: number, status: AlertStatus) {
    const toast = useToastStore()
    const idx = alerts.value.findIndex(a => a.id === id)
    const prev = idx >= 0 ? alerts.value[idx].status : null

    if (idx >= 0) {
      alerts.value[idx].status = status
    }
    updatingIds.value.add(id)

    try {
      await patchAlert(id, status)
      toast.show(
        status === 'acknowledged' ? 'Alerte acknowledgeée' : 'Alerte résolue',
        'success'
      )
    } catch (e) {
      if (idx >= 0 && prev) {
        alerts.value[idx].status = prev
      }
      error.value = (e as Error).message
      toast.show((e as Error).message, 'error')
    } finally {
      updatingIds.value.delete(id)
    }
  }

  function isUpdating(id: number): boolean {
    return updatingIds.value.has(id)
  }

  function reset() {
    alerts.value = []
    error.value = null
    loading.value = false
    statusFilter.value = ''
    updatingIds.value.clear()
  }

  return {
    alerts, loading, error, statusFilter,
    fetchAlerts, setStatusFilter, updateAlertStatus, isUpdating, reset,
  }
})
