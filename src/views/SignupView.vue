<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import PasswordStrengthMeter from '../components/PasswordStrengthMeter.vue'
import {
  isEmail, normalizeEmail, isDomain, isRequired, hasLength,
  isPasswordStrong, passwordsMatch,
  COUNTRIES, INDUSTRIES, COMPANY_SIZES,
} from '../utils/validators'
import type { SignupPayload } from '../types'

const router = useRouter()
const authStore = useAuthStore()
const toastStore = useToastStore()

const step = ref(1)

// Step 1: Company
const companyName = ref('')
const industry = ref('')
const companySize = ref('')
const country = ref('')
const website = ref('')
const securityContact = ref('')

// Step 2: Admin
const fullName = ref('')
const adminEmail = ref('')
const password = ref('')
const showPassword = ref(false)
const confirmPassword = ref('')
const showConfirm = ref(false)
const acceptTerms = ref(false)

// Errors
const errors = ref<Record<string, string>>({})
const accountMayExist = ref(false)
const apiErrors = ref<Record<string, string>>({})
const submitting = ref(false)

// Refs for focus management
const formRef = ref<HTMLElement | null>(null)

async function focusFirstError() {
  await nextTick()
  const firstError = formRef.value?.querySelector('[aria-invalid="true"]') as HTMLElement | null
  if (firstError) firstError.focus()
}

function validateStep1(): boolean {
  errors.value = {}
  if (!isRequired(companyName.value)) errors.value.companyName = 'Nom de l\'entreprise requis'
  else if (!hasLength(companyName.value, 2, 80)) errors.value.companyName = 'Entre 2 et 80 caractères'
  if (!industry.value) errors.value.industry = 'Secteur requis'
  if (!companySize.value) errors.value.companySize = 'Taille requise'
  if (!country.value) errors.value.country = 'Pays requis'
  if (!isRequired(securityContact.value)) errors.value.securityContact = 'Email de contact sécurité requis'
  else if (!isEmail(securityContact.value)) errors.value.securityContact = 'Email invalide'
  if (website.value && !isDomain(website.value)) errors.value.website = 'Domaine invalide (ex: entreprise.com)'
  return Object.keys(errors.value).length === 0
}

function validateStep2(): boolean {
  errors.value = {}
  if (!isRequired(fullName.value)) errors.value.fullName = 'Nom complet requis'
  if (!isRequired(adminEmail.value)) errors.value.adminEmail = 'Email requis'
  else if (!isEmail(adminEmail.value)) errors.value.adminEmail = 'Email invalide'
  if (!isPasswordStrong(password.value)) errors.value.password = 'Le mot de passe ne respecte pas toutes les règles'
  if (!passwordsMatch(password.value, confirmPassword.value)) errors.value.confirmPassword = 'Les mots de passe ne correspondent pas'
  if (!acceptTerms.value) errors.value.acceptTerms = 'Vous devez accepter les conditions'
  return Object.keys(errors.value).length === 0
}

function nextStep() {
  if (validateStep1()) {
    step.value = 2
    focusFirstError()
  } else {
    focusFirstError()
  }
}

function prevStep() {
  step.value = 1
  errors.value = {}
}

async function handleSubmit() {
  if (!validateStep2()) {
    focusFirstError()
    return
  }
  submitting.value = true
  errors.value = {}
  apiErrors.value = {}
  accountMayExist.value = false

  const payload: SignupPayload = {
    company: {
      name: companyName.value.trim(),
      industry: industry.value,
      size: companySize.value,
      country: country.value,
      website: website.value.trim() || undefined,
      securityContact: normalizeEmail(securityContact.value),
    },
    admin: {
      fullName: fullName.value.trim(),
      email: normalizeEmail(adminEmail.value),
      password: password.value,
    },
  }

  try {
    await authStore.signup(payload)
    toastStore.show(`Bienvenue ${fullName.value.trim()} ! Votre entreprise a été créée.`, 'success', 5000)
    router.push('/')
  } catch (e) {
    const msg = (e as Error).message
    if (msg === 'could not create account') {
      // The API deliberately does not say whether the email or the company
      // name is taken, so that registered emails cannot be discovered.
      apiErrors.value.general = 'Impossible de créer le compte : cet email ou ce nom d\'entreprise est peut-être déjà enregistré.'
      accountMayExist.value = true
    } else if (msg.includes('email already used') || msg.includes('email')) {
      apiErrors.value.adminEmail = 'Cet email est déjà utilisé'
    } else {
      apiErrors.value.general = msg
    }
    focusFirstError()
  } finally {
    submitting.value = false
  }
}


</script>

<template>
  <div class="auth-page">
    <div class="auth-card signup-card">
      <div class="login-brand">
        <div class="login-logo">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><path d="M12 2L4 6v6c0 5.5 3.5 10 8 12 4.5-2 8-6.5 8-12V6l-8-4z" /><path d="M9 12l2 2 4-4" /></svg>
        </div>
        <h1>Créer un compte entreprise</h1>
        <p>Mini SIEM — Security Information & Event Management</p>
      </div>

      <!-- Stepper -->
      <div class="stepper" role="tablist" aria-label="Étapes d'inscription">
        <div class="step" :class="{ active: step >= 1, done: step > 1 }">
          <span class="step-number">1</span>
          <span class="step-label">Entreprise</span>
        </div>
        <div class="step-line" :class="{ done: step > 1 }"></div>
        <div class="step" :class="{ active: step >= 2 }">
          <span class="step-number">2</span>
          <span class="step-label">Administrateur</span>
        </div>
      </div>

      <form ref="formRef" @submit.prevent="handleSubmit" class="signup-form" novalidate>
        <!-- Step 1: Company -->
        <div v-if="step === 1" class="step-content">
          <div class="form-field">
            <label for="su-company">Nom de l'entreprise <span class="req">*</span></label>
            <input
              id="su-company"
              v-model="companyName"
              type="text"
              :aria-invalid="!!errors.companyName"
              :aria-describedby="errors.companyName ? 'err-companyName' : undefined"
              placeholder="Acme Corp"
            />
            <span v-if="errors.companyName" id="err-companyName" class="field-error" role="alert">{{ errors.companyName }}</span>
          </div>

          <div class="form-row">
            <div class="form-field">
              <label for="su-industry">Secteur <span class="req">*</span></label>
              <select id="su-industry" v-model="industry" :aria-invalid="!!errors.industry">
                <option value="">Choisir...</option>
                <option v-for="ind in INDUSTRIES" :key="ind" :value="ind">{{ ind }}</option>
              </select>
              <span v-if="errors.industry" class="field-error" role="alert">{{ errors.industry }}</span>
            </div>
            <div class="form-field">
              <label for="su-size">Taille <span class="req">*</span></label>
              <select id="su-size" v-model="companySize" :aria-invalid="!!errors.companySize">
                <option value="">Choisir...</option>
                <option v-for="s in COMPANY_SIZES" :key="s" :value="s">{{ s }}</option>
              </select>
              <span v-if="errors.companySize" class="field-error" role="alert">{{ errors.companySize }}</span>
            </div>
          </div>

          <div class="form-field">
            <label for="su-country">Pays <span class="req">*</span></label>
            <select id="su-country" v-model="country" :aria-invalid="!!errors.country">
              <option value="">Choisir...</option>
              <option v-for="c in COUNTRIES" :key="c" :value="c">{{ c }}</option>
            </select>
            <span v-if="errors.country" class="field-error" role="alert">{{ errors.country }}</span>
          </div>

          <div class="form-field">
            <label for="su-website">Site web / domaine principal (optionnel)</label>
            <input id="su-website" v-model="website" type="text" :aria-invalid="!!errors.website" placeholder="entreprise.com" />
            <span v-if="errors.website" class="field-error" role="alert">{{ errors.website }}</span>
          </div>

          <div class="form-field">
            <label for="su-security">Email du contact sécurité <span class="req">*</span></label>
            <input id="su-security" v-model="securityContact" type="email" :aria-invalid="!!errors.securityContact" placeholder="sec@entreprise.com" />
            <span v-if="errors.securityContact" class="field-error" role="alert">{{ errors.securityContact }}</span>
          </div>

          <button type="button" class="btn btn-primary signup-btn" @click="nextStep">
            Continuer
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7" /></svg>
          </button>
        </div>

        <!-- Step 2: Admin -->
        <div v-if="step === 2" class="step-content">
          <div class="form-field">
            <label for="su-name">Nom complet <span class="req">*</span></label>
            <input id="su-name" v-model="fullName" type="text" :aria-invalid="!!errors.fullName || !!apiErrors.fullName" placeholder="Jean Dupont" autocomplete="name" />
            <span v-if="errors.fullName" class="field-error" role="alert">{{ errors.fullName }}</span>
          </div>

          <div class="form-field">
            <label for="su-email">Email professionnel <span class="req">*</span></label>
            <input id="su-email" v-model="adminEmail" type="email" :aria-invalid="!!errors.adminEmail || !!apiErrors.adminEmail" placeholder="vous@entreprise.com" autocomplete="email" />
            <span v-if="errors.adminEmail" class="field-error" role="alert">{{ errors.adminEmail }}</span>
            <span v-if="apiErrors.adminEmail" class="field-error" role="alert">{{ apiErrors.adminEmail }}</span>
          </div>

          <div class="form-field">
            <label for="su-password">Mot de passe <span class="req">*</span></label>
            <div class="password-input-wrapper">
              <input
                id="su-password"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                :aria-invalid="!!errors.password"
                placeholder="••••••••••••"
                autocomplete="new-password"
              />
              <button type="button" class="pw-toggle" @click="showPassword = !showPassword" :aria-label="showPassword ? 'Masquer' : 'Afficher'">
                <svg v-if="showPassword" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" /><path d="M1 1l22 22" /></svg>
                <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
              </button>
            </div>
            <PasswordStrengthMeter :password="password" />
            <span v-if="errors.password" class="field-error" role="alert">{{ errors.password }}</span>
          </div>

          <div class="form-field">
            <label for="su-confirm">Confirmer le mot de passe <span class="req">*</span></label>
            <div class="password-input-wrapper">
              <input
                id="su-confirm"
                v-model="confirmPassword"
                :type="showConfirm ? 'text' : 'password'"
                :aria-invalid="!!errors.confirmPassword"
                placeholder="••••••••••••"
                autocomplete="new-password"
              />
              <button type="button" class="pw-toggle" @click="showConfirm = !showConfirm" :aria-label="showConfirm ? 'Masquer' : 'Afficher'">
                <svg v-if="showConfirm" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" /><path d="M1 1l22 22" /></svg>
                <svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
              </button>
            </div>
            <span v-if="errors.confirmPassword" class="field-error" role="alert">{{ errors.confirmPassword }}</span>
          </div>

          <div class="form-field form-checkbox">
            <label class="checkbox-label">
              <input type="checkbox" v-model="acceptTerms" :aria-invalid="!!errors.acceptTerms" />
              <span>J'accepte les <a href="#" @click.prevent>conditions d'utilisation</a> et la <a href="#" @click.prevent>politique de confidentialité</a></span>
            </label>
            <span v-if="errors.acceptTerms" class="field-error" role="alert">{{ errors.acceptTerms }}</span>
          </div>

          <div v-if="apiErrors.general" class="login-error" role="alert">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
            <span>
              {{ apiErrors.general }}
              <template v-if="accountMayExist">
                <RouterLink to="/login">Se connecter</RouterLink> ou
                <RouterLink to="/forgot-password">réinitialiser le mot de passe</RouterLink>.
              </template>
            </span>
          </div>

          <div class="signup-actions">
            <button type="button" class="btn btn-secondary" @click="prevStep">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 12H5M12 19l-7-7 7-7" /></svg>
              Retour
            </button>
            <button type="submit" class="btn btn-primary" :disabled="submitting">
              <span v-if="submitting" class="btn-spinner"></span>
              <span v-else>Créer le compte</span>
            </button>
          </div>
        </div>
      </form>

      <div class="auth-footer">
        <p>Déjà un compte ? <RouterLink to="/login">Se connecter</RouterLink></p>
      </div>
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
.signup-card {
  width: 100%;
  max-width: 520px;
  max-height: 90vh;
  overflow-y: auto;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 2rem;
  box-shadow: var(--shadow-lg);
}
.login-brand {
  text-align: center;
  margin-bottom: 1.5rem;
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
.login-brand h1 { font-size: 1.25rem; font-weight: 700; margin-bottom: 0.25rem; }
.login-brand p { font-size: 0.8125rem; color: var(--text-muted); }

.stepper {
  display: flex;
  align-items: center;
  margin-bottom: 1.5rem;
  gap: 0.5rem;
}
.step {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.step-number {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 700;
  background: var(--border);
  color: var(--text-muted);
  flex-shrink: 0;
  transition: all 0.2s;
}
.step.active .step-number {
  background: var(--accent);
  color: #fff;
}
.step.done .step-number {
  background: var(--success);
  color: #fff;
}
.step-label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--text-muted);
}
.step.active .step-label { color: var(--text); }
.step-line {
  flex: 1;
  height: 2px;
  background: var(--border);
  border-radius: 1px;
  transition: background 0.2s;
}
.step-line.done { background: var(--success); }

.signup-form { display: flex; flex-direction: column; gap: 1rem; }
.step-content { display: flex; flex-direction: column; gap: 1rem; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-field label { font-size: 0.8125rem; font-weight: 600; color: var(--text-dim); }
.req { color: var(--critical); }
.form-field input,
.form-field select { padding: 10px 14px; font-size: 0.9375rem; }
.form-row { display: flex; gap: 1rem; }
.form-row .form-field { flex: 1; }
.field-error {
  font-size: 0.75rem;
  color: var(--critical);
  padding-left: 4px;
}
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
  align-items: center;
  justify-content: center;
}
.pw-toggle:hover { color: var(--text); }

.form-checkbox { margin-top: 0.25rem; }
.checkbox-label {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--text-dim);
  cursor: pointer;
}
.checkbox-label input {
  margin-top: 3px;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.checkbox-label a { color: var(--accent); }

.signup-btn {
  width: 100%;
  justify-content: center;
  padding: 0.625rem;
  font-size: 0.9375rem;
  margin-top: 0.5rem;
}
.signup-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: 0.5rem;
}
.signup-actions .btn { flex: 1; justify-content: center; padding: 0.625rem; }
.btn-spinner {
  width: 16px; height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
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
.auth-footer {
  text-align: center;
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border);
}
.auth-footer p { font-size: 0.8125rem; color: var(--text-muted); }
.auth-footer a { color: var(--accent); font-weight: 600; }

@media (max-width: 480px) {
  .form-row { flex-direction: column; }
  .signup-card { padding: 1.25rem; }
  .step-label { display: none; }
}
</style>
