<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useAddressesStore, validateAddress } from '../stores/addresses'
import { useAuthStore } from '../stores/auth'

const store = useAddressesStore()
const authStore = useAuthStore()
const { addresses, loading, submitting, deletingId, error } = storeToRefs(store)

const newValue = ref('')
const newLabel = ref('')
const validationError = ref<string | null>(null)
const deleteConfirmId = ref<number | null>(null)

async function handleAdd() {
  validationError.value = validateAddress(newValue.value)
  if (validationError.value) return
  try {
    await store.addAddress(newValue.value, newLabel.value || undefined)
    newValue.value = ''
    newLabel.value = ''
    // Rafraîchit les adresses du store auth pour le sélecteur global
    await authStore.fetchAddresses()
  } catch {
    // L'erreur est déjà affichée via toast
  }
}

async function handleDelete(id: number) {
  try {
    await store.removeAddress(id)
    await authStore.fetchAddresses()
    // Si l'adresse supprimée était sélectionnée, réinitialise
    if (authStore.selectedAddress === addresses.value.find(a => a.id === id)?.value) {
      authStore.setSelectedAddress('')
    }
  } catch {
    // Géré par toast
  }
}

function confirmDelete(id: number) {
  deleteConfirmId.value = id
}

function cancelDelete() {
  deleteConfirmId.value = null
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('fr-FR', { day: '2-digit', month: 'short', year: 'numeric' })
}

onMounted(() => {
  store.fetchAddresses()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Adresses surveillées</h2>
        <p>Adresses IP et domaines enregistrés pour {{ authStore.user?.companyName }}</p>
      </div>
    </div>

    <!-- Formulaire d'ajout -->
    <div class="card add-form-card">
      <span class="card-title">Ajouter une adresse</span>
      <form @submit.prevent="handleAdd" class="add-form">
        <div class="add-form-row">
          <div class="form-field">
            <label for="addr-value">Adresse (IP ou domaine)</label>
            <input
              id="addr-value"
              v-model="newValue"
              type="text"
              placeholder="ex: 10.0.0.12 ou serveur.exemple.com"
              class="mono"
              :disabled="submitting"
            />
          </div>
          <div class="form-field">
            <label for="addr-label">Libellé (optionnel)</label>
            <input
              id="addr-label"
              v-model="newLabel"
              type="text"
              placeholder="ex: Serveur web principal"
              :disabled="submitting"
            />
          </div>
          <button type="submit" class="btn btn-primary add-btn" :disabled="submitting">
            <span v-if="submitting" class="btn-spinner-sm"></span>
            <span v-else>Ajouter</span>
          </button>
        </div>
        <div v-if="validationError" class="validation-error" role="alert">{{ validationError }}</div>
      </form>
    </div>

    <!-- Liste des adresses -->
    <div class="card" style="margin-top: 1rem;">
      <span class="card-title">Adresses enregistrées ({{ addresses.length }})</span>

      <!-- Loading -->
      <div v-if="loading" class="skeleton-list">
        <div class="skeleton skeleton-line" v-for="i in 3" :key="i" style="height: 48px; margin-bottom: 8px;"></div>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="error-state">
        <div class="error-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--critical)"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
        </div>
        <p>Erreur lors du chargement des adresses</p>
        <button class="btn btn-primary" @click="store.fetchAddresses()">Réessayer</button>
      </div>

      <!-- Empty -->
      <div v-else-if="addresses.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14" /></svg>
        </div>
        <p>Aucune adresse enregistrée</p>
        <span>Ajoutez une adresse ci-dessus pour commencer le suivi</span>
      </div>

      <!-- List -->
      <div v-else class="address-list">
        <div v-for="addr in addresses" :key="addr.id" class="address-row">
          <div class="address-info">
            <span class="address-value mono">{{ addr.value }}</span>
            <span v-if="addr.label" class="address-label">{{ addr.label }}</span>
            <span class="address-date">{{ formatDate(addr.createdAt) }}</span>
          </div>
          <div class="address-actions">
            <template v-if="deleteConfirmId === addr.id">
              <span class="confirm-text">Confirmer ?</span>
              <button class="btn btn-sm" style="background: var(--critical); color: #fff;" @click="handleDelete(addr.id)" :disabled="deletingId === addr.id">
                <span v-if="deletingId === addr.id" class="btn-spinner-sm"></span>
                <span v-else>Oui</span>
              </button>
              <button class="btn btn-secondary btn-sm" @click="cancelDelete">Non</button>
            </template>
            <button
              v-else
              class="btn btn-sm btn-danger"
              @click="confirmDelete(addr.id)"
              aria-label="Supprimer l'adresse"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" /></svg>
              Supprimer
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.add-form-card { margin-bottom: 1rem; }
.add-form { display: flex; flex-direction: column; gap: 0.75rem; }
.add-form-row {
  display: flex;
  gap: 1rem;
  align-items: flex-end;
  flex-wrap: wrap;
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  flex: 1;
  min-width: 180px;
}
.form-field label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--text-muted);
}
.add-btn {
  padding: 9px 1.25rem;
  white-space: nowrap;
}
.validation-error {
  color: var(--critical);
  font-size: 0.8125rem;
  padding: 0.5rem 0.75rem;
  background: var(--critical-bg);
  border-radius: var(--radius-sm);
}
.address-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.address-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: var(--bg);
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  transition: border-color 0.15s;
}
.address-row:hover {
  border-color: var(--border);
}
.address-info {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}
.address-value {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text);
}
.address-label {
  font-size: 0.8125rem;
  color: var(--text-dim);
}
.address-date {
  font-size: 0.75rem;
  color: var(--text-muted);
}
.address-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.confirm-text {
  font-size: 0.75rem;
  color: var(--warning);
  font-weight: 600;
}
.btn-danger {
  background: var(--critical-bg);
  color: var(--critical);
  border: 1px solid rgba(239, 68, 68, 0.25);
}
.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}
.btn-spinner-sm {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
.skeleton-list { padding: 0; }

@media (max-width: 768px) {
  .add-form-row { flex-direction: column; }
  .add-btn { width: 100%; }
  .address-row { flex-direction: column; align-items: flex-start; gap: 0.75rem; }
}
</style>
