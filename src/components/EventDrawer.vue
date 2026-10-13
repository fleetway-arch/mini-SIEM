<script setup lang="ts">
import { onUnmounted, watch } from 'vue'
import type { SiemEvent } from '../types'
import SeverityBadge from './SeverityBadge.vue'

const props = defineProps<{
  event: SiemEvent | null
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('fr-FR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

// Ferme le drawer avec la touche Escape
function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.open) {
    emit('close')
  }
}

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    document.addEventListener('keydown', onKeyDown)
  } else {
    document.removeEventListener('keydown', onKeyDown)
  }
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeyDown)
})
</script>

<template>
  <div>
    <!-- Overlay : pointer-events:none quand fermé pour ne pas bloquer les clics -->
    <div
      class="drawer-overlay"
      :class="{ visible: open }"
      @click="emit('close')"
    ></div>
    <div
      class="drawer"
      :class="{ open }"
      role="dialog"
      aria-label="Détail de l'événement"
    >
      <template v-if="event">
        <div class="drawer-header">
          <h3>Détail de l'événement #{{ event.id }}</h3>
          <button class="drawer-close" @click="emit('close')" aria-label="Fermer">&times;</button>
        </div>
        <div class="drawer-body">
          <div class="drawer-field">
            <span class="drawer-field-label">Horodatage</span>
            <span class="drawer-field-value mono">{{ formatTime(event.timestamp) }}</span>
          </div>
          <div class="drawer-field">
            <span class="drawer-field-label">Source</span>
            <span class="drawer-field-value"><span class="source-tag mono">{{ event.source }}</span></span>
          </div>
          <div class="drawer-field">
            <span class="drawer-field-label">Sévérité</span>
            <span class="drawer-field-value"><SeverityBadge :severity="event.severity" /></span>
          </div>
          <div class="drawer-field">
            <span class="drawer-field-label">Adresse IP</span>
            <span class="drawer-field-value mono">{{ event.ip }}</span>
          </div>
          <div class="drawer-field">
            <span class="drawer-field-label">Message</span>
            <span class="drawer-field-value">{{ event.message }}</span>
          </div>
          <div class="drawer-field">
            <span class="drawer-field-label">Log brut</span>
            <pre class="drawer-raw-log">{{ event.rawLog }}</pre>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.source-tag {
  padding: 2px 8px;
  border-radius: 4px;
  background: rgba(59, 130, 246, 0.12);
  color: var(--accent);
}
</style>
