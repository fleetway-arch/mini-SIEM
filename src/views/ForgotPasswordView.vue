<script setup lang="ts">
import { ref } from 'vue'
import { forgotPassword } from '../api/client'
import { isEmail, normalizeEmail } from '../utils/validators'

// Same text whatever the server knows about the email, so this page never
// reveals whether an account exists.
const SUCCESS_MESSAGE = 'Si un compte existe pour cette adresse, un email contenant un lien de réinitialisation vient d\'être envoyé. Le lien est valable 30 minutes.'

const email = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)
const sent = ref(false)

async function handleSubmit() {
  error.value = null
  if (!isEmail(email.value)) {
    error.value = 'Adresse email invalide'
    return
  }
  submitting.value = true
  try {
    await forgotPassword(normalizeEmail(email.value))
    sent.value = true
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <div class="login-logo">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
        </div>
        <h1>Mot de passe oublié</h1>
        <p>Recevez un lien pour choisir un nouveau mot de passe</p>
      </div>

      <div v-if="sent" class="login-success" role="status" aria-live="polite">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><path d="M22 4L12 14.01l-3-3" /></svg>
        <span>{{ SUCCESS_MESSAGE }}</span>
      </div>

      <form v-else @submit.prevent="handleSubmit" class="login-form" novalidate>
        <div class="form-field">
          <label for="forgot-email">Email</label>
          <input
            id="forgot-email"
            v-model="email"
            type="email"
            placeholder="vous@entreprise.com"
            autocomplete="email"
            :aria-invalid="!!error"
            :aria-describedby="error ? 'forgot-error' : undefined"
            :disabled="submitting"
          />
        </div>

        <div v-if="error" id="forgot-error" class="login-error" role="alert">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
          {{ error }}
        </div>

        <button type="submit" class="btn btn-primary login-btn" :disabled="submitting">
          <span v-if="submitting" class="btn-spinner"></span>
          <span v-else>Envoyer le lien</span>
        </button>
      </form>

      <div class="auth-footer">
        <p><RouterLink to="/login">Retour à la connexion</RouterLink></p>
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
.login-brand { text-align: center; margin-bottom: 2rem; }
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
.login-brand h1 { font-size: 1.5rem; font-weight: 700; margin-bottom: 0.25rem; }
.login-brand p { font-size: 0.8125rem; color: var(--text-muted); }
.login-form { display: flex; flex-direction: column; gap: 1.25rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-field label { font-size: 0.8125rem; font-weight: 600; color: var(--text-dim); }
.form-field input { padding: 10px 14px; font-size: 0.9375rem; }
.login-error,
.login-success {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  line-height: 1.5;
}
.login-error { background: var(--critical-bg); color: var(--critical); align-items: center; }
.login-success { background: var(--success-bg); color: var(--success); }
.login-success svg { flex-shrink: 0; margin-top: 2px; }
.login-btn { width: 100%; justify-content: center; padding: 0.625rem; font-size: 0.9375rem; }
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
