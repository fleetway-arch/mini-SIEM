import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { StatsSummary, TimelinePoint, TopIp, Alert, TimeRange } from '../types'
import { getStatsSummary, getTimeline, getTopIps, getAlerts } from '../api/client'

export const useDashboardStore = defineStore('dashboard', () => {
  const summary = ref<StatsSummary | null>(null)
  const timeline = ref<TimelinePoint[]>([])
  const topIps = ref<TopIp[]>([])
  const latestCriticalAlerts = ref<Alert[]>([])

  const loading = ref(false)
  const rangeLoading = ref(false)  // Loading spécifique au changement de plage
  const error = ref<string | null>(null)

  const range = ref<TimeRange>('24h')
  const autoRefresh = ref(false)
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  async function fetchSummary(address?: string) {
    try {
      summary.value = await getStatsSummary(address)
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  async function fetchTimeline(address?: string) {
    try {
      timeline.value = await getTimeline(range.value, address)
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  async function fetchTopIps(address?: string) {
    try {
      topIps.value = await getTopIps(10, address)
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  async function fetchCriticalAlerts(address?: string) {
    try {
      const all = await getAlerts('open', address)
      latestCriticalAlerts.value = all.filter(a => a.severity === 'critical').slice(0, 5)
    } catch (e) {
      error.value = (e as Error).message
    }
  }

  // Charge toutes les données du dashboard
  async function fetchAll(address?: string) {
    loading.value = true
    error.value = null
    try {
      await Promise.all([
        fetchSummary(address),
        fetchTimeline(address),
        fetchTopIps(address),
        fetchCriticalAlerts(address),
      ])
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  async function setRange(r: TimeRange, address?: string) {
    rangeLoading.value = true
    range.value = r
    try {
      timeline.value = await getTimeline(r, address)
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      rangeLoading.value = false
    }
  }

  function toggleAutoRefresh() {
    autoRefresh.value = !autoRefresh.value
    if (autoRefresh.value) {
      refreshTimer = setInterval(() => {
        fetchAll()
      }, 10000)
    } else {
      if (refreshTimer) {
        clearInterval(refreshTimer)
        refreshTimer = null
      }
    }
  }

  function cleanup() {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
    autoRefresh.value = false
  }

  function reset() {
    summary.value = null
    timeline.value = []
    topIps.value = []
    latestCriticalAlerts.value = []
    error.value = null
    loading.value = false
    rangeLoading.value = false
    range.value = '24h'
    cleanup()
  }

  return {
    summary, timeline, topIps, latestCriticalAlerts,
    loading, rangeLoading, error,
    range, autoRefresh,
    fetchAll, fetchSummary, fetchTimeline, setRange,
    toggleAutoRefresh, cleanup, reset,
  }
})
