<script setup lang="ts">
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import type { TimelinePoint } from '../types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{
  data: TimelinePoint[]
}>()

function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
}

const chartData = computed(() => ({
  labels: props.data.map(d => formatTime(d.time)),
  datasets: [
    {
      label: 'Total',
      data: props.data.map(d => d.total),
      borderColor: '#3b82f6',
      backgroundColor: 'rgba(59, 130, 246, 0.1)',
      fill: true,
      tension: 0.35,
      pointRadius: 0,
      pointHoverRadius: 5,
    },
    {
      label: 'Failed',
      data: props.data.map(d => d.failed),
      borderColor: '#ef4444',
      backgroundColor: 'rgba(239, 68, 68, 0.08)',
      fill: true,
      tension: 0.35,
      pointRadius: 0,
      pointHoverRadius: 5,
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'index' as const,
    intersect: false,
  },
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
      ticks: { color: '#64748b', font: { size: 11 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 8 },
    },
    y: {
      grid: { color: 'rgba(30, 42, 69, 0.5)' },
      ticks: { color: '#64748b', font: { size: 11 } },
      beginAtZero: true,
    },
  },
}
</script>

<script lang="ts">
import { computed } from 'vue'
</script>

<template>
  <div class="chart-wrapper">
    <Line :data="chartData" :options="chartOptions" />
  </div>
</template>

<style scoped>
.chart-wrapper {
  height: 280px;
  position: relative;
}
</style>
