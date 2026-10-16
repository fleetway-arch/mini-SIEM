import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SiemEvent, Source, Severity } from '../types'
import { getEvents } from '../api/client'

// Les filtres de date sont des jours locaux (YYYY-MM-DD) : l'API reçoit le
// début du jour en UTC. « Au » inclut le jour choisi, donc la borne est le
// début du jour suivant (exclusive côté API).
export function localDayStart(day: string, offsetDays = 0): string {
  const [year, month, date] = day.split('-').map(Number)
  return new Date(year, month - 1, date + offsetDays).toISOString()
}

export const useEventsStore = defineStore('events', () => {
  const events = ref<SiemEvent[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const page = ref(1)
  const limit = ref(20)
  const source = ref<Source | ''>('')
  const severity = ref<Severity | ''>('')
  const ip = ref('')
  const q = ref('')
  const from = ref('')
  const to = ref('')

  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / limit.value)))

  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  async function fetchEvents(address?: string) {
    loading.value = true
    error.value = null
    try {
      const result = await getEvents({
        page: page.value,
        limit: limit.value,
        source: source.value,
        severity: severity.value,
        ip: ip.value,
        q: q.value,
        from: from.value ? localDayStart(from.value) : undefined,
        to: to.value ? localDayStart(to.value, 1) : undefined,
        address: address || undefined,
      })
      events.value = result.items
      total.value = result.total
    } catch (e) {
      error.value = (e as Error).message
      events.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  // Debounce pour la recherche texte
  function setQ(value: string, address?: string) {
    q.value = value
    page.value = 1
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      fetchEvents(address)
    }, 400)
  }

  function setFilter(key: 'source' | 'severity' | 'ip' | 'from' | 'to', value: string, address?: string) {
    if (key === 'source') source.value = value as Source | ''
    else if (key === 'severity') severity.value = value as Severity | ''
    else if (key === 'ip') ip.value = value
    else if (key === 'from') from.value = value
    else if (key === 'to') to.value = value
    page.value = 1
    fetchEvents(address)
  }

  function setPage(p: number, address?: string) {
    page.value = p
    fetchEvents(address)
  }

  function setLimit(l: number, address?: string) {
    limit.value = l
    page.value = 1
    fetchEvents(address)
  }

  function resetFilters(address?: string) {
    source.value = ''
    severity.value = ''
    ip.value = ''
    q.value = ''
    from.value = ''
    to.value = ''
    page.value = 1
    fetchEvents(address)
  }

  function reset() {
    events.value = []
    total.value = 0
    error.value = null
    loading.value = false
    page.value = 1
    source.value = ''
    severity.value = ''
    ip.value = ''
    q.value = ''
    from.value = ''
    to.value = ''
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
  }

  return {
    events, total, loading, error,
    page, limit, source, severity, ip, q, from, to,
    totalPages,
    fetchEvents, setQ, setFilter, setPage, setLimit, resetFilters, reset,
  }
})
