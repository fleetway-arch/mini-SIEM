<script setup lang="ts">
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js'
import type { TopIp } from '../types'

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend)

const props = defineProps<{
  data: TopIp[]
}>()

// Barres horizontales : on inverse les données pour un affichage top-to-bottom
const chartData = computed(() => ({
  labels: props.data.map(d => d.ip),
  datasets: [
    {
      label: 'Total',
      data: props.data.map(d => d.count),
      backgroundColor: 'rgba(59, 130, 246, 0.6)',
      borderColor: '#3b82f6',
      borderWidth: 1,
      borderRadius: 4,
    },
    {
      label: 'Failed',
      data: props.data.map(d => d.failedCount),
      backgroundColor: 'rgba(239, 68, 68, 0.6)',
      borderColor: '#ef4444',
      borderWidth: 1,
      borderRadius: 4,
    },
  ],
}))

const chartOptions = {
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      labels: { color: '#94a3b8', font: { size: 12 }, usePointStyle: true, pointStyle: 'circle' as const },
    },
    tooltip: {
      backgroundColor: '#111a2e',
      borderColor: '#1e2a45',
      borderWidth: 1,
      titleColor: '#e2e8f0',
      bodyColor: '#94a3b8',
    },
  },
  scales: {
    x: {
      grid: { color: 'rgba(30, 42, 69, 0.5)' },
      ticks: { color: '#64748b', font: { size: 11 } },
      beginAtZero: true,
    },
    y: {
      grid: { display: false },
      ticks: { color: '#94a3b8', font: { size: 11, family: "'JetBrains Mono', monospace" } },
    },
  },
}
</script>

<script lang="ts">
import { computed } from 'vue'
</script>

<template>
  <div class="chart-wrapper">
    <Bar :data="chartData" :options="chartOptions" />
  </div>
</template>

<style scoped>
.chart-wrapper {
  height: 320px;
  position: relative;
}
</style>
