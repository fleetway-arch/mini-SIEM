<script setup lang="ts">
import type { Source, Severity } from '../types'

defineProps<{
  source: Source | ''
  severity: Severity | ''
  ip: string
  q: string
  from: string
  to: string
}>()

const emit = defineEmits<{
  (e: 'update:source', v: Source | ''): void
  (e: 'update:severity', v: Severity | ''): void
  (e: 'update:ip', v: string): void
  (e: 'update:q', v: string): void
  (e: 'update:from', v: string): void
  (e: 'update:to', v: string): void
  (e: 'reset'): void
}>()

const sources: { value: Source | ''; label: string }[] = [
  { value: '', label: 'Toutes sources' },
  { value: 'sshd', label: 'sshd' },
  { value: 'apache', label: 'apache' },
  { value: 'dns', label: 'dns' },
  { value: 'vsftpd', label: 'vsftpd' },
]

const severities: { value: Severity | ''; label: string }[] = [
  { value: '', label: 'Toutes sévérités' },
  { value: 'info', label: 'info' },
  { value: 'warning', label: 'warning' },
  { value: 'critical', label: 'critical' },
]
</script>

<template>
  <div class="filter-bar">
    <div class="filter-row">
      <div class="filter-group">
        <label for="filter-source">Source</label>
        <select id="filter-source" :value="source" @change="emit('update:source', ($event.target as HTMLSelectElement).value as Source | '')">
          <option v-for="s in sources" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>

      <div class="filter-group">
        <label for="filter-severity">Sévérité</label>
        <select id="filter-severity" :value="severity" @change="emit('update:severity', ($event.target as HTMLSelectElement).value as Severity | '')">
          <option v-for="s in severities" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>

      <div class="filter-group">
        <label for="filter-ip">Adresse IP</label>
        <input id="filter-ip" type="text" :value="ip" @input="emit('update:ip', ($event.target as HTMLInputElement).value)" placeholder="ex: 185.220..." class="mono" />
      </div>

      <div class="filter-group">
        <label for="filter-q">Recherche</label>
        <input id="filter-q" type="text" :value="q" @input="emit('update:q', ($event.target as HTMLInputElement).value)" placeholder="Texte libre..." />
      </div>
    </div>

    <div class="filter-row">
      <div class="filter-group">
        <label for="filter-from">Du</label>
        <input id="filter-from" type="date" :value="from" @change="emit('update:from', ($event.target as HTMLInputElement).value)" />
      </div>
      <div class="filter-group">
        <label for="filter-to">Au</label>
        <input id="filter-to" type="date" :value="to" @change="emit('update:to', ($event.target as HTMLInputElement).value)" />
      </div>
      <button class="btn btn-secondary btn-sm" @click="emit('reset')">Réinitialiser</button>
    </div>
  </div>
</template>

<style scoped>
.filter-bar {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1.5rem;
  box-shadow: var(--shadow-sm);
}
.filter-row {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
  align-items: flex-end;
}
.filter-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 140px;
}
.filter-group label {
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
}
.filter-group input,
.filter-group select {
  min-width: 120px;
}
</style>
