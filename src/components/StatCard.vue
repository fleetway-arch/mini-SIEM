<script setup lang="ts">
defineProps<{
  label: string
  value: number | string
  icon: string
  color: string
}>()
</script>

<template>
  <div class="card stat-card">
    <div class="stat-icon" :style="{ background: color + '20', color }">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <template v-if="icon === 'events'"><path d="M4 4h16v16H4z" /><path d="M8 8h8M8 12h8M8 16h5" /></template>
        <template v-else-if="icon === 'failed'"><circle cx="12" cy="12" r="10" /><path d="M15 9l-6 6M9 9l6 6" /></template>
        <template v-else-if="icon === 'ips'"><rect x="2" y="6" width="20" height="12" rx="2" /><path d="M6 12h.01M12 12h.01" /></template>
        <template v-else-if="icon === 'alerts'"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" /><path d="M12 9v4M12 17h.01" /></template>
      </svg>
    </div>
    <div class="stat-info">
      <span class="stat-value">{{ typeof value === 'number' ? value.toLocaleString() : value }}</span>
      <span class="stat-label">{{ label }}</span>
    </div>
  </div>
</template>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
  transition: border-color 0.2s;
}
.stat-card:hover {
  border-color: var(--accent);
}
.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.stat-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.stat-value {
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1;
}
.stat-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
</style>
