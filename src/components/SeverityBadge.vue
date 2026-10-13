<script setup lang="ts">
defineProps<{
  severity: 'info' | 'warning' | 'critical'
}>()
</script>

<template>
  <span class="severity-badge" :class="`severity-${severity}`">
    <span class="dot"></span>
    {{ severity }}
  </span>
</template>
