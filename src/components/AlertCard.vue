<script setup lang="ts">
import type { Alert, AlertStatus } from '../types'
import SeverityBadge from './SeverityBadge.vue'
import { useAlertsStore } from '../stores/alerts'

defineProps<{
  alert: Alert
}>()

const emit = defineEmits<{
  (e: 'update-status', id: number, status: AlertStatus): void
}>()

const alertsStore = useAlertsStore()

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('fr-FR', {
    day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}
</script>

<template>
  <div class="card alert-card">
    <div class="alert-header">
      <div class="alert-title-row">
        <h4 class="alert-rule">{{ alert.rule }}</h4>
        <SeverityBadge :severity="alert.severity" />
      </div>
      <span class="status-badge" :class="`status-${alert.status}`">{{ alert.status }}</span>
    </div>

    <p class="alert-desc">{{ alert.description }}</p>

    <div class="alert-meta">
      <div class="meta-item">
        <span class="meta-label">IP</span>
        <span class="meta-value mono">{{ alert.ip }}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Événements</span>
        <span class="meta-value">{{ alert.eventCount }}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Créé le</span>
        <span class="meta-value">{{ formatDate(alert.createdAt) }}</span>
      </div>
    </div>

    <div class="alert-actions" v-if="alert.status !== 'resolved'">
      <button
        v-if="alert.status === 'open'"
        class="btn btn-warn btn-sm"
        :disabled="alertsStore.isUpdating(alert.id)"
        @click="emit('update-status', alert.id, 'acknowledged')"
      >
        <span v-if="alertsStore.isUpdating(alert.id)" class="btn-spinner-sm"></span>
        <span v-else>Acknowledger</span>
      </button>
      <button
        class="btn btn-success btn-sm"
        :disabled="alertsStore.isUpdating(alert.id)"
        @click="emit('update-status', alert.id, 'resolved')"
      >
        <span v-if="alertsStore.isUpdating(alert.id)" class="btn-spinner-sm"></span>
        <span v-else>Résoudre</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.alert-card {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  transition: border-color 0.18s ease;
}
.alert-card:hover {
  border-color: var(--accent);
}
.alert-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.5rem;
}
.alert-title-row {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  flex-wrap: wrap;
}
.alert-rule {
  font-size: 0.9375rem;
  font-weight: 700;
}
.alert-desc {
  font-size: 0.8125rem;
  color: var(--text-dim);
  line-height: 1.5;
}
.alert-meta {
  display: flex;
  gap: 1.5rem;
  flex-wrap: wrap;
  padding-top: 0.625rem;
  border-top: 1px solid rgba(30, 42, 69, 0.4);
}
.meta-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.meta-label {
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
}
.meta-value {
  font-size: 0.8125rem;
  color: var(--text);
}
.alert-actions {
  display: flex;
  gap: 0.5rem;
}
.btn-spinner-sm {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
</style>
