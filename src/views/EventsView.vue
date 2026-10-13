<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useEventsStore } from '../stores/events'
import { useAuthStore } from '../stores/auth'
import EventsTable from '../components/EventsTable.vue'
import EventDrawer from '../components/EventDrawer.vue'
import FilterBar from '../components/FilterBar.vue'
import type { SiemEvent, Source, Severity } from '../types'

const store = useEventsStore()
const authStore = useAuthStore()
const {
  events, total, loading, error,
  page, limit, source, severity, ip, q, from, to,
  totalPages,
} = storeToRefs(store)
const { selectedAddress } = storeToRefs(authStore)

const selectedEvent = ref<SiemEvent | null>(null)
const drawerOpen = ref(false)

function openDrawer(event: SiemEvent) {
  selectedEvent.value = event
  drawerOpen.value = true
}

function closeDrawer() {
  drawerOpen.value = false
}

function handleSource(v: Source | '') { store.setFilter('source', v, selectedAddress.value || undefined) }
function handleSeverity(v: Severity | '') { store.setFilter('severity', v, selectedAddress.value || undefined) }
function handleIp(v: string) { store.setFilter('ip', v, selectedAddress.value || undefined) }
function handleQ(v: string) { store.setQ(v, selectedAddress.value || undefined) }
function handleFrom(v: string) { store.setFilter('from', v, selectedAddress.value || undefined) }
function handleTo(v: string) { store.setFilter('to', v, selectedAddress.value || undefined) }

function handleLimitChange(e: Event) {
  store.setLimit(Number((e.target as HTMLSelectElement).value), selectedAddress.value || undefined)
}

function handlePrevPage() { store.setPage(page.value - 1, selectedAddress.value || undefined) }
function handleNextPage() { store.setPage(page.value + 1, selectedAddress.value || undefined) }

// Recharge quand l'adresse change
watch(selectedAddress, () => {
  store.fetchEvents(selectedAddress.value || undefined)
})

onMounted(() => {
  store.fetchEvents(selectedAddress.value || undefined)
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Événements{{ selectedAddress ? ' pour ' + selectedAddress : '' }}</h2>
        <p>Journal complet des événements de sécurité{{ selectedAddress ? ' — filtré par adresse' : '' }}</p>
      </div>
    </div>

    <!-- État d'erreur -->
    <div v-if="error" class="card error-state">
      <div class="error-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--critical)"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
      </div>
      <p>Erreur lors du chargement des événements</p>
      <button class="btn btn-primary" @click="store.fetchEvents(selectedAddress || undefined)">Réessayer</button>
    </div>

    <template v-else>
      <FilterBar
        :source="source"
        :severity="severity"
        :ip="ip"
        :q="q"
        :from="from"
        :to="to"
        @update:source="handleSource"
        @update:severity="handleSeverity"
        @update:ip="handleIp"
        @update:q="handleQ"
        @update:from="handleFrom"
        @update:to="handleTo"
        @reset="store.resetFilters(selectedAddress || undefined)"
      />

      <div class="card">
        <div class="table-info">
          <span>{{ total }} événement{{ total > 1 ? 's' : '' }}</span>
        </div>
        <EventsTable :events="events" :loading="loading" @row-click="openDrawer" />
      </div>

      <!-- Pagination -->
      <div class="pagination" v-if="total > 0">
        <div class="page-size">
          <select :value="limit" @change="handleLimitChange">
            <option :value="10">10 / page</option>
            <option :value="20">20 / page</option>
            <option :value="50">50 / page</option>
            <option :value="100">100 / page</option>
          </select>
        </div>
        <div class="page-nav">
          <button class="btn btn-secondary btn-sm" :disabled="page === 1" @click="handlePrevPage">Précédent</button>
          <span class="page-info">Page {{ page }} / {{ totalPages }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="page >= totalPages" @click="handleNextPage">Suivant</button>
        </div>
      </div>
    </template>

    <EventDrawer :event="selectedEvent" :open="drawerOpen" @close="closeDrawer" />
  </div>
</template>

<style scoped>
.table-info {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-bottom: 0.75rem;
}
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 1.25rem;
  flex-wrap: wrap;
  gap: 1rem;
}
.page-nav {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.page-info {
  font-size: 0.8125rem;
  color: var(--text-dim);
}
.page-size select {
  font-size: 0.75rem;
  padding: 4px 8px;
}
</style>
