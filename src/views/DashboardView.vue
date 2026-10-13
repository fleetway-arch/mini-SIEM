<script setup lang="ts">
import { onMounted, onUnmounted, watch, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useDashboardStore } from '../stores/dashboard'
import { useAuthStore } from '../stores/auth'
import StatCard from '../components/StatCard.vue'
import TimelineChart from '../components/TimelineChart.vue'
import TopIpsChart from '../components/TopIpsChart.vue'
import SeverityBadge from '../components/SeverityBadge.vue'
import type { TimeRange } from '../types'

const store = useDashboardStore()
const authStore = useAuthStore()
const { summary, timeline, topIps, latestCriticalAlerts, loading, rangeLoading, error, range, autoRefresh } = storeToRefs(store)
const { selectedAddress } = storeToRefs(authStore)

const ranges: TimeRange[] = ['1h', '24h', '7d']

const addressLabel = ref('')
function updateAddressLabel() {
  addressLabel.value = selectedAddress.value ? ` — ${selectedAddress.value}` : ''
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('fr-FR', {
    day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit',
  })
}

function loadAll() {
  store.fetchAll(selectedAddress.value || undefined)
}

// Recharge quand l'adresse change
watch(selectedAddress, () => {
  updateAddressLabel()
  loadAll()
})

onMounted(() => {
  updateAddressLabel()
  loadAll()
})

onUnmounted(() => {
  store.cleanup()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Dashboard{{ addressLabel }}</h2>
        <p>Vue d'ensemble de l'activité de sécurité{{ addressLabel ? ` pour ${selectedAddress}` : '' }}</p>
      </div>
      <div class="auto-refresh">
        <label class="toggle-switch">
          <input type="checkbox" :checked="autoRefresh" @change="store.toggleAutoRefresh()" aria-label="Activer l'auto-refresh" />
          <span class="toggle-slider"></span>
        </label>
        Auto-refresh (10s)
      </div>
    </div>

    <!-- État d'erreur global -->
    <div v-if="error && !loading" class="card error-state">
      <div class="error-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--critical)"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
      </div>
      <p>Erreur lors du chargement des données</p>
      <button class="btn btn-primary" @click="loadAll">Réessayer</button>
    </div>

    <!-- Stat cards -->
    <div class="stat-grid">
      <template v-if="loading && !summary">
        <div class="card skeleton-card" v-for="i in 4" :key="i">
          <div class="skeleton skeleton-block"></div>
        </div>
      </template>
      <template v-else-if="summary">
        <StatCard label="Total événements" :value="summary.totalEvents" icon="events" color="#3b82f6" />
        <StatCard label="Logins échoués" :value="summary.failedLogins" icon="failed" color="#f59e0b" />
        <StatCard label="IPs uniques" :value="summary.uniqueIps" icon="ips" color="#22c55e" />
        <StatCard label="Alertes ouvertes" :value="summary.openAlerts" icon="alerts" color="#ef4444" />
      </template>
    </div>

    <!-- Charts -->
    <div class="dashboard-grid">
      <div class="card">
        <div class="card-header-row">
          <span class="card-title">Événements dans le temps</span>
          <div class="range-selector" role="group" aria-label="Sélecteur de plage temporelle">
            <button
              v-for="r in ranges"
              :key="r"
              class="range-btn"
              :class="{ active: range === r }"
              :disabled="rangeLoading"
              @click="store.setRange(r, selectedAddress || undefined)"
            >
              {{ r }}
            </button>
          </div>
        </div>
        <div v-if="rangeLoading" class="spinner-container">
          <div class="spinner"></div>
        </div>
        <div v-else-if="timeline.length === 0 && !loading" class="empty-state">
          <div class="empty-icon"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12h18M3 6h18M3 18h18" /></svg></div>
          <p>Aucune donnée temporelle</p>
        </div>
        <TimelineChart v-else :data="timeline" />
      </div>

      <div class="card">
        <span class="card-title">Top 10 IPs sources</span>
        <div v-if="loading && topIps.length === 0" class="spinner-container">
          <div class="spinner"></div>
        </div>
        <div v-else-if="topIps.length === 0" class="empty-state">
          <div class="empty-icon"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="6" width="20" height="12" rx="2" /></svg></div>
          <p>Aucune donnée IP</p>
        </div>
        <TopIpsChart v-else :data="topIps" />
      </div>
    </div>

    <!-- Latest critical alerts -->
    <div class="card">
      <span class="card-title">Dernières alertes critiques</span>
      <div v-if="loading && latestCriticalAlerts.length === 0" class="spinner-container">
        <div class="spinner"></div>
      </div>
      <div v-else-if="latestCriticalAlerts.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z" /></svg>
        </div>
        <p>Aucune alerte critique ouverte</p>
      </div>
      <div v-else class="critical-list">
        <div v-for="alert in latestCriticalAlerts" :key="alert.id" class="critical-row">
          <SeverityBadge severity="critical" />
          <span class="critical-rule">{{ alert.rule }}</span>
          <span class="critical-ip mono">{{ alert.ip }}</span>
          <span class="critical-date">{{ formatDate(alert.createdAt) }}</span>
          <span class="critical-events">{{ alert.eventCount }} événements</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.skeleton-card { height: 96px; display: flex; align-items: center; }
.critical-list { display: flex; flex-direction: column; gap: 0.625rem; }
.critical-row {
  display: flex; align-items: center; gap: 1rem;
  padding: 0.625rem 0.875rem;
  background: var(--bg); border-radius: var(--radius-sm);
  border-left: 3px solid var(--critical);
  font-size: 0.8125rem;
}
.critical-rule { font-weight: 600; flex: 1; }
.critical-ip { color: var(--text-dim); font-size: 0.75rem; }
.critical-date { color: var(--text-muted); font-size: 0.75rem; white-space: nowrap; }
.critical-events { color: var(--text-dim); font-size: 0.75rem; white-space: nowrap; }

@media (max-width: 768px) {
  .critical-row { flex-wrap: wrap; gap: 0.5rem; }
}
</style>
