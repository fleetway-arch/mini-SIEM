<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { resetPassword } from '../api/client'
import { useToastStore } from '../stores/toast'
import PasswordStrengthMeter from '../components/PasswordStrengthMeter.vue'
import { isPasswordStrong, passwordsMatch } from '../utils/validators'

const route = useRoute()
const router = useRouter()
const toastStore = useToastStore()

const token = computed(() => route.params.token as string)

const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const formErrors = ref<Record<string, string>>({})
const submitting = ref(false)
const linkInvalid = ref(false)
const formRef = ref<HTMLElement | null>(null)

async function focusFirstError() {
  await nextTick()
  const el = formRef.value?.querySelector('[aria-invalid="true"]') as HTMLElement | null
  el?.focus()
}

async function handleSubmit() {
  formErrors.value = {}
  if (!isPasswordStrong(password.value)) formErrors.value.password = 'Le mot de passe ne respecte pas les règles'
  if (!passwordsMatch(password.value, confirmPassword.value)) formErrors.value.confirmPassword = 'Les mots de passe ne correspondent pas'
  if (Object.keys(formErrors.value).length > 0) {
    focusFirstError()
    return
  }

  submitting.value = true
  try {
    await resetPassword(token.value, password.value)
    toastStore.show('Mot de passe modifié. Connectez-vous avec votre nouveau mot de passe.', 'success', 5000)
    router.push('/login')
  } catch (e) {
    const message = (e as Error).message
    if (message.includes('reset link')) linkInvalid.value = true
    else formErrors.value.general = message
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
        <h1>Nouveau mot de passe</h1>
        <p>Choisissez un mot de passe robuste pour votre compte</p>
      </div>

      <div v-if="linkInvalid" class="invalid-state" role="alert">
        <div class="login-error">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
          Ce lien de réinitialisation est invalide, expiré ou a déjà été utilisé.
        </div>
        <RouterLink to="/forgot-password" class="btn btn-primary login-btn">Demander un nouveau lien</RouterLink>
      </div>

      <form v-else ref="formRef" @submit.prevent="handleSubmit" class="login-form" novalidate>
        <div class="form-field">
          <label for="reset-password">Nouveau mot de passe</label>
          <div class="password-input-wrapper">
            <input
              id="reset-password"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="••••••••••••"
              autocomplete="new-password"
              :aria-invalid="!!formErrors.password"
              :aria-describedby="formErrors.password ? 'reset-password-error' : undefined"
              :disabled="submitting"
            />
            <button type="button" class="pw-toggle" @click="showPassword = !showPassword" :aria-label="showPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'">
              <svg v-if="showPassword" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" /><path d="M1 1l22 22" /></svg>
              <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
            </button>
          </div>
          <PasswordStrengthMeter :password="password" />
          <span v-if="formErrors.password" id="reset-password-error" class="field-error" role="alert">{{ formErrors.password }}</span>
        </div>

        <div class="form-field">
          <label for="reset-confirm">Confirmer le mot de passe</label>
          <input
            id="reset-confirm"
            v-model="confirmPassword"
            :type="showPassword ? 'text' : 'password'"
            placeholder="••••••••••••"
            autocomplete="new-password"
            :aria-invalid="!!formErrors.confirmPassword"
            :aria-describedby="formErrors.confirmPassword ? 'reset-confirm-error' : undefined"
            :disabled="submitting"
          />
          <span v-if="formErrors.confirmPassword" id="reset-confirm-error" class="field-error" role="alert">{{ formErrors.confirmPassword }}</span>
        </div>

        <div v-if="formErrors.general" class="login-error" role="alert">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
          {{ formErrors.general }}
        </div>

        <button type="submit" class="btn btn-primary login-btn" :disabled="submitting">
          <span v-if="submitting" class="btn-spinner"></span>
          <span v-else>Changer le mot de passe</span>
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
.login-form,
.invalid-state { display: flex; flex-direction: column; gap: 1.25rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-field label { font-size: 0.8125rem; font-weight: 600; color: var(--text-dim); }
.form-field input { padding: 10px 14px; font-size: 0.9375rem; }
.field-error { font-size: 0.75rem; color: var(--critical); }
.password-input-wrapper { position: relative; }
.password-input-wrapper input { width: 100%; padding-right: 40px; }
.pw-toggle {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  color: var(--text-muted);
  padding: 4px;
  border-radius: var(--radius-xs);
  display: flex;
}
.pw-toggle:hover { color: var(--text); }
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
