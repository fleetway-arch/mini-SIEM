<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import { getInvitation } from '../api/client'
import PasswordStrengthMeter from '../components/PasswordStrengthMeter.vue'
import { isRequired, isPasswordStrong, passwordsMatch } from '../utils/validators'
import type { InvitationDetails } from '../types'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const toastStore = useToastStore()

const token = computed(() => route.params.token as string)

const invitation = ref<InvitationDetails | null>(null)
const loading = ref(true)
const errorState = ref<string | null>(null) // 'invalid' | 'expired' | 'used' | null

const fullName = ref('')
const password = ref('')
const showPassword = ref(false)
const confirmPassword = ref('')
const showConfirm = ref(false)
const formErrors = ref<Record<string, string>>({})
const submitting = ref(false)
const formRef = ref<HTMLElement | null>(null)

const errorTitle = computed(() => {
  switch (errorState.value) {
    case 'invalid': return 'Invitation invalide'
    case 'expired': return 'Invitation expirée'
    case 'used': return 'Invitation déjà utilisée'
    default: return 'Erreur'
  }
})

const errorMessage = computed(() => {
  switch (errorState.value) {
    case 'invalid': return 'Ce lien d\'invitation n\'est pas valide ou n\'existe plus.'
    case 'expired': return 'Cette invitation a expiré. Contactez un administrateur pour en demander une nouvelle.'
    case 'used': return 'Cette invitation a déjà été acceptée. Vous pouvez vous connecter avec votre compte.'
    default: return 'Une erreur est survenue.'
  }
})

async function focusFirstError() {
  await nextTick()
  const el = formRef.value?.querySelector('[aria-invalid="true"]') as HTMLElement | null
  if (el) el.focus()
}

async function fetchInvitation() {
  loading.value = true
  errorState.value = null
  try {
    invitation.value = await getInvitation(token.value)
  } catch (e) {
    const msg = (e as Error).message
    if (msg.includes('expired')) errorState.value = 'expired'
    else if (msg.includes('used')) errorState.value = 'used'
    else errorState.value = 'invalid'
  } finally {
    loading.value = false
  }
}

async function handleAccept() {
  formErrors.value = {}
  if (!isRequired(fullName.value)) formErrors.value.fullName = 'Nom complet requis'
  if (!isPasswordStrong(password.value)) formErrors.value.password = 'Le mot de passe ne respecte pas les règles'
  if (!passwordsMatch(password.value, confirmPassword.value)) formErrors.value.confirmPassword = 'Les mots de passe ne correspondent pas'

  if (Object.keys(formErrors.value).length > 0) {
    focusFirstError()
    return
  }

  submitting.value = true
  try {
    await authStore.acceptInvitation(token.value, fullName.value.trim(), password.value)
    toastStore.show(`Bienvenue ${fullName.value.trim()} !`, 'success', 4000)
    router.push('/')
  } catch (e) {
    formErrors.value.general = (e as Error).message
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchInvitation()
})
</script>

<template>
  <div class="auth-page">
    <div class="auth-card invite-card">
      <div class="login-brand">
        <div class="login-logo">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><path d="M12 2L4 6v6c0 5.5 3.5 10 8 12 4.5-2 8-6.5 8-12V6l-8-4z" /><path d="M9 12l2 2 4-4" /></svg>
        </div>
        <h1>Accepter l'invitation</h1>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="spinner-container">
        <div class="spinner"></div>
      </div>

      <!-- Error states -->
      <div v-else-if="errorState" class="invite-error-state">
        <div class="error-icon-large">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--critical)">
            <circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" />
          </svg>
        </div>
        <h2>{{ errorTitle }}</h2>
        <p>{{ errorMessage }}</p>
        <RouterLink to="/login" class="btn btn-primary">Aller à la page de connexion</RouterLink>
      </div>

      <!-- Invitation form -->
      <template v-else-if="invitation">
        <div class="invite-info">
          <div class="invite-info-row">
            <span class="invite-info-label">Entreprise</span>
            <span class="invite-info-value">{{ invitation.companyName }}</span>
          </div>
          <div class="invite-info-row">
            <span class="invite-info-label">Email</span>
            <span class="invite-info-value mono">{{ invitation.email }}</span>
          </div>
          <div class="invite-info-row">
            <span class="invite-info-label">Rôle</span>
            <span class="role-badge" :class="`role-${invitation.role}`">{{ invitation.role }}</span>
          </div>
        </div>

        <form ref="formRef" @submit.prevent="handleAccept" class="signup-form" novalidate>
          <div class="form-field">
            <label for="inv-name">Nom complet <span class="req">*</span></label>
            <input id="inv-name" v-model="fullName" type="text" :aria-invalid="!!formErrors.fullName" placeholder="Jean Dupont" autocomplete="name" />
            <span v-if="formErrors.fullName" class="field-error" role="alert">{{ formErrors.fullName }}</span>
          </div>

          <div class="form-field">
            <label for="inv-password">Mot de passe <span class="req">*</span></label>
            <div class="password-input-wrapper">
              <input
                id="inv-password"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                :aria-invalid="!!formErrors.password"
                placeholder="••••••••••••"
                autocomplete="new-password"
              />
              <button type="button" class="pw-toggle" @click="showPassword = !showPassword" :aria-label="showPassword ? 'Masquer' : 'Afficher'">
                <svg v-if="showPassword" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" /><path d="M1 1l22 22" /></svg>
                <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
              </button>
            </div>
            <PasswordStrengthMeter :password="password" />
            <span v-if="formErrors.password" class="field-error" role="alert">{{ formErrors.password }}</span>
          </div>

          <div class="form-field">
            <label for="inv-confirm">Confirmer le mot de passe <span class="req">*</span></label>
            <div class="password-input-wrapper">
              <input
                id="inv-confirm"
                v-model="confirmPassword"
                :type="showConfirm ? 'text' : 'password'"
                :aria-invalid="!!formErrors.confirmPassword"
                placeholder="••••••••••••"
                autocomplete="new-password"
              />
              <button type="button" class="pw-toggle" @click="showConfirm = !showConfirm" :aria-label="showConfirm ? 'Masquer' : 'Afficher'">
                <svg v-if="showConfirm" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" /><path d="M1 1l22 22" /></svg>
                <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
              </button>
            </div>
            <span v-if="formErrors.confirmPassword" class="field-error" role="alert">{{ formErrors.confirmPassword }}</span>
          </div>

          <div v-if="formErrors.general" class="login-error" role="alert">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
            {{ formErrors.general }}
          </div>

          <button type="submit" class="btn btn-primary signup-btn" :disabled="submitting">
            <span v-if="submitting" class="btn-spinner"></span>
            <span v-else>Accepter et créer mon compte</span>
          </button>
        </form>

        <div class="auth-footer">
          <p>Déjà un compte ? <RouterLink to="/login">Se connecter</RouterLink></p>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  padding: 1.5rem;
}
.invite-card {
  width: 100%;
  max-width: 460px;
  max-height: 90vh;
  overflow-y: auto;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 2rem;
  box-shadow: var(--shadow-lg);
}
.login-brand { text-align: center; margin-bottom: 1.5rem; }
.login-logo {
  width: 56px; height: 56px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--accent), var(--accent-dim));
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 1rem;
}
.login-brand h1 { font-size: 1.25rem; font-weight: 700; }

.invite-info {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 1rem;
  margin-bottom: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}
.invite-info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}
.invite-info-label {
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
}
.invite-info-value { font-size: 0.875rem; color: var(--text); }

.role-badge {
  font-size: 0.6875rem;
  font-weight: 700;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
}
.role-owner { background: rgba(168, 85, 247, 0.12); color: #a855f7; }
.role-admin { background: var(--info-bg); color: var(--info); }
.role-viewer { background: var(--border); color: var(--text-dim); }

.signup-form { display: flex; flex-direction: column; gap: 1rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-field label { font-size: 0.8125rem; font-weight: 600; color: var(--text-dim); }
.req { color: var(--critical); }
.form-field input { padding: 10px 14px; font-size: 0.9375rem; }
.field-error { font-size: 0.75rem; color: var(--critical); }

.password-input-wrapper { position: relative; }
.password-input-wrapper input { width: 100%; padding-right: 40px; }
.pw-toggle {
  position: absolute;
  right: 8px; top: 50%;
  transform: translateY(-50%);
  background: none;
  color: var(--text-muted);
  padding: 4px;
  border-radius: var(--radius-xs);
  display: flex; align-items: center; justify-content: center;
}
.pw-toggle:hover { color: var(--text); }

.signup-btn { width: 100%; justify-content: center; padding: 0.625rem; font-size: 0.9375rem; }
.btn-spinner {
  width: 16px; height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
.login-error {
  display: flex; align-items: center; gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  background: var(--critical-bg);
  color: var(--critical);
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
}

.invite-error-state {
  text-align: center;
  padding: 1.5rem 0;
}
.error-icon-large {
  width: 56px; height: 56px;
  border-radius: 50%;
  background: var(--critical-bg);
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 1rem;
}
.invite-error-state h2 { font-size: 1.125rem; font-weight: 700; margin-bottom: 0.5rem; }
.invite-error-state p { font-size: 0.875rem; color: var(--text-dim); margin-bottom: 1.5rem; line-height: 1.5; }

.auth-footer {
  text-align: center;
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border);
}
.auth-footer p { font-size: 0.8125rem; color: var(--text-muted); }
.auth-footer a { color: var(--accent); font-weight: 600; }
</style>
