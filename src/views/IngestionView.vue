<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { IngestKey, CreatedIngestKey } from '../types'
import { BASE_URL, getIngestKeys, createIngestKey, revokeIngestKey } from '../api/client'
import { useToastStore } from '../stores/toast'

const toastStore = useToastStore()

const keys = ref<IngestKey[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const submitting = ref(false)
const revokingId = ref<number | null>(null)
const revokeConfirmId = ref<number | null>(null)

const newName = ref('')
const validationError = ref<string | null>(null)
// La clé en clair n'est renvoyée qu'une fois par l'API
const createdKey = ref<CreatedIngestKey | null>(null)

const curlExample = computed(() => {
  const key = createdKey.value?.key ?? '<votre-clé>'
  return `curl -X POST ${BASE_URL}/api/ingest/events \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '{"events":[{"source":"sshd","severity":"warning","ip":"203.0.113.9","message":"Failed password for root","timestamp":"${new Date().toISOString().slice(0, 19)}Z"}]}'`
})

async function fetchKeys() {
  loading.value = true
  error.value = null
  try {
    keys.value = await getIngestKeys()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  const name = newName.value.trim()
  validationError.value = name ? null : 'Veuillez nommer la clé (ex : serveur web)'
  if (validationError.value) return
  submitting.value = true
  try {
    const created = await createIngestKey(name)
    createdKey.value = created
    keys.value.push(created)
    newName.value = ''
  } catch (e) {
    toastStore.show((e as Error).message, 'error')
  } finally {
    submitting.value = false
  }
}

async function copyKey() {
  if (!createdKey.value) return
  try {
    await navigator.clipboard.writeText(createdKey.value.key)
    toastStore.show('Clé copiée', 'success')
  } catch {
    toastStore.show('Copie impossible : sélectionnez la clé manuellement', 'error')
  }
}

async function handleRevoke(id: number) {
  revokingId.value = id
  try {
    await revokeIngestKey(id)
    keys.value = keys.value.filter(k => k.id !== id)
    if (createdKey.value?.id === id) createdKey.value = null
    toastStore.show('Clé révoquée', 'success')
  } catch (e) {
    toastStore.show((e as Error).message, 'error')
  } finally {
    revokingId.value = null
    revokeConfirmId.value = null
  }
}

function formatDate(iso?: string): string {
  if (!iso) return 'Jamais'
  return new Date(iso).toLocaleString('fr-FR', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchKeys)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Collecte des événements</h2>
        <p>Clés d'API utilisées par vos serveurs et collecteurs pour envoyer leurs journaux</p>
      </div>
    </div>

    <div class="card add-form-card">
      <span class="card-title">Nouvelle clé d'ingestion</span>
      <form @submit.prevent="handleCreate" class="add-form">
        <div class="add-form-row">
          <div class="form-field">
            <label for="key-name">Nom</label>
            <input id="key-name" v-model="newName" type="text" maxlength="64" placeholder="ex: Serveur web principal" :disabled="submitting" />
          </div>
          <button type="submit" class="btn btn-primary add-btn" :disabled="submitting">
            <span v-if="submitting" class="btn-spinner-sm"></span>
            <span v-else>Créer</span>
          </button>
        </div>
        <div v-if="validationError" class="validation-error" role="alert">{{ validationError }}</div>
      </form>

      <div v-if="createdKey" class="created-key" role="status">
        <p><strong>Copiez cette clé maintenant :</strong> elle ne sera plus jamais affichée.</p>
        <div class="created-key-row">
          <code class="mono key-value">{{ createdKey.key }}</code>
          <button type="button" class="btn btn-secondary btn-sm" @click="copyKey">Copier</button>
        </div>
      </div>
    </div>

    <div class="card" style="margin-top: 1rem;">
      <span class="card-title">Clés actives ({{ keys.length }})</span>

      <div v-if="loading" class="skeleton-list">
        <div class="skeleton skeleton-line" v-for="i in 2" :key="i" style="height: 48px; margin-bottom: 8px;"></div>
      </div>

      <div v-else-if="error" class="error-state">
        <p>Erreur lors du chargement des clés</p>
        <button class="btn btn-primary" @click="fetchKeys">Réessayer</button>
      </div>

      <div v-else-if="keys.length === 0" class="empty-state">
        <p>Aucune clé d'ingestion</p>
        <span>Créez une clé pour commencer à envoyer des événements</span>
      </div>

      <div v-else class="key-list">
        <div v-for="key in keys" :key="key.id" class="key-row">
          <div class="key-info">
            <span class="key-name">{{ key.name }}</span>
            <span class="mono key-prefix">{{ key.prefix }}…</span>
            <span class="key-date">Créée le {{ formatDate(key.createdAt) }}</span>
            <span class="key-date">Dernière utilisation : {{ formatDate(key.lastUsedAt) }}</span>
          </div>
          <div class="key-actions">
            <template v-if="revokeConfirmId === key.id">
              <span class="confirm-text">Les envois avec cette clé seront refusés. Confirmer ?</span>
              <button class="btn btn-sm" style="background: var(--critical); color: #fff;" @click="handleRevoke(key.id)" :disabled="revokingId === key.id">
                <span v-if="revokingId === key.id" class="btn-spinner-sm"></span>
                <span v-else>Révoquer</span>
              </button>
              <button class="btn btn-secondary btn-sm" @click="revokeConfirmId = null">Annuler</button>
            </template>
            <button v-else class="btn btn-sm btn-danger" @click="revokeConfirmId = key.id" :aria-label="`Révoquer la clé ${key.name}`">
              Révoquer
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="card" style="margin-top: 1rem;">
      <span class="card-title">Envoyer des événements</span>
      <p class="help">
        <code class="mono">POST /api/ingest/events</code> accepte jusqu'à 500 événements par requête.
        <code class="mono">source</code> et <code class="mono">message</code> (ou <code class="mono">rawLog</code>) sont obligatoires ;
        <code class="mono">severity</code> vaut <code class="mono">info</code>, <code class="mono">warning</code> ou <code class="mono">critical</code>.
        Les échecs d'authentification répétés depuis une même IP et les événements critiques créent des alertes.
      </p>
      <pre class="mono curl">{{ curlExample }}</pre>
    </div>
  </div>
</template>

<style scoped>
.add-form-card { margin-bottom: 1rem; }
.add-form { display: flex; flex-direction: column; gap: 0.75rem; }
.add-form-row { display: flex; gap: 1rem; align-items: flex-end; flex-wrap: wrap; }
.form-field { display: flex; flex-direction: column; gap: 0.375rem; flex: 1; min-width: 180px; }
.form-field label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--text-muted);
}
.add-btn { padding: 9px 1.25rem; white-space: nowrap; }
.validation-error {
  color: var(--critical);
  font-size: 0.8125rem;
  padding: 0.5rem 0.75rem;
  background: var(--critical-bg);
  border-radius: var(--radius-sm);
}
.created-key {
  margin-top: 1rem;
  padding: 0.75rem 1rem;
  background: var(--warning-bg);
  border-radius: var(--radius-sm);
  font-size: 0.8125rem;
  color: var(--text);
}
.created-key-row { display: flex; align-items: center; gap: 0.75rem; margin-top: 0.5rem; flex-wrap: wrap; }
.key-value { word-break: break-all; font-size: 0.8125rem; user-select: all; }
.key-list { display: flex; flex-direction: column; gap: 0.5rem; }
.key-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: var(--bg);
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  transition: border-color 0.15s;
}
.key-row:hover { border-color: var(--border); }
.key-info { display: flex; align-items: center; gap: 1rem; flex-wrap: wrap; }
.key-name { font-size: 0.875rem; font-weight: 600; color: var(--text); }
.key-prefix { font-size: 0.8125rem; color: var(--text-dim); }
.key-date { font-size: 0.75rem; color: var(--text-muted); }
.key-actions { display: flex; align-items: center; gap: 0.5rem; }
.confirm-text { font-size: 0.75rem; color: var(--warning); font-weight: 600; }
.btn-danger {
  background: var(--critical-bg);
  color: var(--critical);
  border: 1px solid rgba(239, 68, 68, 0.25);
}
.btn-danger:hover { background: rgba(239, 68, 68, 0.2); }
.btn-spinner-sm {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
.help { font-size: 0.8125rem; color: var(--text-dim); line-height: 1.6; margin: 0.5rem 0 0.75rem; }
.curl {
  margin: 0;
  padding: 0.75rem 1rem;
  background: var(--bg);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text);
}

@media (max-width: 768px) {
  .add-form-row { flex-direction: column; }
  .add-btn { width: 100%; }
  .key-row { flex-direction: column; align-items: flex-start; gap: 0.75rem; }
}
</style>
