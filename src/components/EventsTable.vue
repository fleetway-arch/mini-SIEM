<script setup lang="ts">
import type { SiemEvent } from '../types'
import SeverityBadge from './SeverityBadge.vue'

defineProps<{
  events: SiemEvent[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'row-click', event: SiemEvent): void
}>()

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('fr-FR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}
</script>

<template>
  <div class="events-table-wrapper">
    <!-- Squelettes de chargement -->
    <div v-if="loading && events.length === 0" class="table-skeleton">
      <div class="skeleton skeleton-line" v-for="i in 6" :key="i" style="height: 44px; border-radius: var(--radius-sm);"></div>
    </div>

    <table class="events-table" v-else-if="events.length > 0">
      <thead>
        <tr>
          <th>Horodatage</th>
          <th>Source</th>
          <th>Sévérité</th>
          <th>Adresse IP</th>
          <th>Message</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="event in events"
          :key="event.id"
          class="event-row"
          @click="emit('row-click', event)"
        >
          <td class="mono small">{{ formatTime(event.timestamp) }}</td>
          <td><span class="source-tag">{{ event.source }}</span></td>
          <td><SeverityBadge :severity="event.severity" /></td>
          <td class="mono">{{ event.ip }}</td>
          <td class="message-cell" :title="event.message">{{ event.message }}</td>
        </tr>
      </tbody>
    </table>

    <!-- État vide -->
    <div v-if="!loading && events.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12h6M9 16h6M9 8h6M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z" /></svg>
      </div>
      <p>Aucun événement trouvé</p>
      <span>Essayez de modifier les filtres</span>
    </div>
  </div>
</template>

<style scoped>
.events-table-wrapper {
  overflow-x: auto;
}
.events-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.events-table thead {
  position: sticky;
  top: 0;
  z-index: 1;
}
.events-table th {
  text-align: left;
  padding: 0.75rem 1rem;
  font-size: 0.6875rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
  background: var(--card);
}
.events-table td {
  padding: 0.625rem 1rem;
  border-bottom: 1px solid rgba(30, 42, 69, 0.4);
  vertical-align: middle;
}
.event-row {
  cursor: pointer;
  transition: background 0.15s ease;
}
.event-row:hover {
  background: var(--card-hover);
}
.small {
  font-size: 0.75rem;
  white-space: nowrap;
}
.source-tag {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(59, 130, 246, 0.12);
  color: var(--accent);
}
.message-cell {
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-dim);
}
.table-skeleton {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
