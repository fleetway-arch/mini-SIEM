<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useAlertsStore } from '../stores/alerts'
import { useAuthStore } from '../stores/auth'
import AlertCard from '../components/AlertCard.vue'
import type { AlertStatus } from '../types'

const store = useAlertsStore()
const authStore = useAuthStore()
const { alerts, loading, error, statusFilter } = storeToRefs(store)
const { selectedAddress } = storeToRefs(authStore)

const statuses: { value: AlertStatus | ''; label: string }[] = [
  { value: '', label: 'Toutes' },
  { value: 'open', label: 'Ouvertes' },
  { value: 'acknowledged', label: 'Acknowledgeées' },
  { value: 'resolved', label: 'Résolues' },
]

// Recharge quand l'adresse change
watch(selectedAddress, () => {
  store.fetchAlerts(selectedAddress.value || undefined)
})

onMounted(() => {
  store.fetchAlerts(selectedAddress.value || undefined)
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Alertes{{ selectedAddress ? ' pour ' + selectedAddress : '' }}</h2>
        <p>Règles de détection déclenchées et leur statut</p>
      </div>
    </div>

    <!-- Filtres de statut -->
    <div class="status-filters" role="group" aria-label="Filtrer par statut">
      <button
        v-for="s in statuses"
        :key="s.value"
        class="status-filter-btn"
        :class="{ active: statusFilter === s.value }"
        @click="store.setStatusFilter(s.value, selectedAddress || undefined)"
      >
        {{ s.label }}
      </button>
    </div>

    <!-- État d'erreur -->
    <div v-if="error" class="card error-state">
      <div class="error-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--critical)"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
      </div>
      <p>Erreur lors du chargement des alertes</p>
      <button class="btn btn-primary" @click="store.fetchAlerts(selectedAddress || undefined)">Réessayer</button>
    </div>

    <!-- État de chargement -->
    <div v-else-if="loading && alerts.length === 0" class="spinner-container">
      <div class="spinner"></div>
    </div>

    <!-- État vide -->
    <div v-else-if="alerts.length === 0" class="card empty-state">
      <div class="empty-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z" /></svg>
      </div>
      <p>Aucune alerte</p>
      <span>Les alertes apparaîtront ici une fois détectées</span>
    </div>

    <!-- Liste d'alertes -->
    <div v-else class="alerts-grid">
      <AlertCard
        v-for="alert in alerts"
        :key="alert.id"
        :alert="alert"
        @update-status="store.updateAlertStatus"
      />
    </div>
  </div>
</template>

<style scoped>
.status-filters {
  display: flex;
  gap: 4px;
  margin-bottom: 1.5rem;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 4px;
  width: fit-content;
  flex-wrap: wrap;
}
.status-filter-btn {
  padding: 6px 16px;
  font-size: 0.8125rem;
  font-weight: 600;
  background: transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
}
.status-filter-btn.active {
  background: var(--accent);
  color: #fff;
}
.status-filter-btn:hover:not(.active) {
  color: var(--text);
}
.alerts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 1rem;
}
@media (max-width: 768px) {
  .alerts-grid { grid-template-columns: 1fr; }
  .status-filters { width: 100%; }
}
</style>
