<script setup lang="ts">
import { computed } from 'vue'
import { checkPasswordRules, passwordStrengthScore } from '../utils/validators'

const props = defineProps<{
  password: string
}>()

const rules = computed(() => checkPasswordRules(props.password))
const score = computed(() => passwordStrengthScore(props.password))

const strengthLabel = computed(() => {
  if (score.value === 0) return ''
  if (score.value <= 2) return 'Faible'
  if (score.value <= 3) return 'Moyen'
  if (score.value === 4) return 'Bon'
  return 'Excellent'
})

const ruleItems = [
  { key: 'minLength', label: 'Au moins 12 caractères' },
  { key: 'hasUpper', label: 'Une majuscule' },
  { key: 'hasLower', label: 'Une minuscule' },
  { key: 'hasDigit', label: 'Un chiffre' },
  { key: 'hasSymbol', label: 'Un symbole' },
] as const
</script>

<template>
  <div class="pw-meter" v-if="password.length > 0">
    <div class="pw-bar">
      <div
        v-for="i in 5"
        :key="i"
        class="pw-bar-segment"
        :class="{ filled: i <= score, [`level-${score}`]: i <= score }"
      ></div>
    </div>
    <span class="pw-label" :class="`pw-label-${score}`">{{ strengthLabel }}</span>
    <ul class="pw-rules" aria-live="polite">
      <li v-for="rule in ruleItems" :key="rule.key" :class="{ met: rules[rule.key] }">
        <svg v-if="rules[rule.key]" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="M20 6L9 17l-5-5" /></svg>
        <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M8 12h8" /></svg>
        {{ rule.label }}
      </li>
    </ul>
  </div>
</template>

<style scoped>
.pw-meter {
  margin-top: 0.5rem;
}
.pw-bar {
  display: flex;
  gap: 4px;
  margin-bottom: 0.5rem;
}
.pw-bar-segment {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--border);
  transition: background 0.2s;
}
.pw-bar-segment.filled.level-1,
.pw-bar-segment.filled.level-2 {
  background: var(--critical);
}
.pw-bar-segment.filled.level-3 {
  background: var(--warning);
}
.pw-bar-segment.filled.level-4 {
  background: #84cc16;
}
.pw-bar-segment.filled.level-5 {
  background: var(--success);
}
.pw-label {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--text-muted);
}
.pw-label-0 { color: var(--text-muted); }
.pw-label-1, .pw-label-2 { color: var(--critical); }
.pw-label-3 { color: var(--warning); }
.pw-label-4 { color: #84cc16; }
.pw-label-5 { color: var(--success); }

.pw-rules {
  list-style: none;
  padding: 0;
  margin-top: 0.5rem;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 0.75rem;
}
.pw-rules li {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.6875rem;
  color: var(--text-muted);
}
.pw-rules li.met {
  color: var(--success);
}
@media (max-width: 480px) {
  .pw-rules { grid-template-columns: 1fr; }
}
</style>
