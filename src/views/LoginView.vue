<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const toastStore = useToastStore()
const { loading, error } = storeToRefs(authStore)

const email = ref('')
const password = ref('')
const localError = ref<string | null>(null)

const redirectTo = computed(() => (route.query.redirect as string) || '/')

async function handleSubmit() {
  localError.value = null
  if (!email.value || !password.value) {
    localError.value = 'Veuillez saisir votre email et votre mot de passe'
    return
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
    localError.value = 'Adresse email invalide'
    return
  }
  try {
    await authStore.login(email.value, password.value)
    toastStore.show('Connexion réussie', 'success')
    router.push(redirectTo.value)
  } catch (e) {
    localError.value = (e as Error).message
  }
}

</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <div class="login-logo">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><path d="M12 2L4 6v6c0 5.5 3.5 10 8 12 4.5-2 8-6.5 8-12V6l-8-4z" /><path d="M9 12l2 2 4-4" /></svg>
        </div>
        <h1>Mini SIEM</h1>
        <p>Security Information & Event Management</p>
      </div>

      <form @submit.prevent="handleSubmit" class="login-form" novalidate>
        <div class="form-field">
          <label for="login-email">Email</label>
          <input
            id="login-email"
            v-model="email"
            type="email"
            placeholder="vous@entreprise.com"
            autocomplete="email"
            :disabled="loading"
          />
        </div>

        <div class="form-field">
          <label for="login-password">Mot de passe</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            placeholder="••••••••"
            autocomplete="current-password"
            :disabled="loading"
          />
        </div>

        <div v-if="localError || error" class="login-error" role="alert">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
          {{ localError || error }}
        </div>

        <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
          <span v-if="loading" class="btn-spinner"></span>
          <span v-else>Se connecter</span>
        </button>
      </form>

      <div class="auth-footer">
        <p>Pas encore de compte ? <RouterLink to="/signup">Créer un compte entreprise</RouterLink></p>
      </div>

    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  padding: 1.5rem;
}
.login-card {
  width: 100%;
  max-width: 420px;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 2rem;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}
.login-brand {
  text-align: center;
  margin-bottom: 2rem;
}
.login-logo {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--accent), var(--accent-dim));
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}
.login-brand h1 {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 0.25rem;
}
.login-brand p {
  font-size: 0.8125rem;
  color: var(--text-muted);
}
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}
.form-field label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-dim);
}
.form-field input {
  padding: 10px 14px;
  font-size: 0.9375rem;
}
.login-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  background: var(--critical-bg);
  color: var(--critical);
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
}
.login-btn {
  width: 100%;
  justify-content: center;
  padding: 0.625rem;
  font-size: 0.9375rem;
}
.btn-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
.auth-footer {
  text-align: center;
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border);
}
.auth-footer p { font-size: 0.8125rem; color: var(--text-muted); }
.auth-footer a { color: var(--accent); font-weight: 600; }
</style>
