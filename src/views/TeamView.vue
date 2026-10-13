<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import { useCan } from '../composables/useCan'
import { getTeam, createInvitations, updateMember, deleteMember, resendInvitation } from '../api/client'
import { isEmail, normalizeEmail, parseEmailList } from '../utils/validators'
import type { TeamMember, Role, MemberStatus } from '../types'

const authStore = useAuthStore()
const toastStore = useToastStore()
const { user } = storeToRefs(authStore)
const { canRemoveMember, canChangeRoleOf } = useCan()

const members = ref<TeamMember[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

// Single invite form
const inviteEmail = ref('')
const inviteRole = ref<Role>('viewer')
const inviteError = ref<string | null>(null)
const inviteLoading = ref(false)

// Bulk add
const showBulk = ref(false)
const bulkText = ref('')
const bulkRole = ref<Role>('viewer')
const bulkPreview = computed(() => parseEmailList(bulkText.value))
const bulkLoading = ref(false)

// Confirm dialog
const confirmDialog = ref<{ title: string; message: string; action: () => void } | null>(null)

// Action loading set
const actionLoading = ref<Set<number>>(new Set())
function isLoading(id: number) { return actionLoading.value.has(id) }

async function fetchTeam() {
  loading.value = true
  error.value = null
  try {
    members.value = await getTeam()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    loading.value = false
  }
}

async function handleSingleInvite() {
  inviteError.value = null
  const email = normalizeEmail(inviteEmail.value)
  if (!isEmail(email)) {
    inviteError.value = 'Email invalide'
    return
  }
  if (members.value.some(m => m.email === email)) {
    inviteError.value = 'Cet email est déjà dans l\'équipe'
    return
  }
  inviteLoading.value = true
  try {
    const result = await createInvitations([email], inviteRole.value)
    if (result.created.length > 0) {
      members.value.push(result.created[0])
      toastStore.show('Invitation envoyée', 'success')
      inviteEmail.value = ''
    }
  } catch (e) {
    inviteError.value = (e as Error).message
  } finally {
    inviteLoading.value = false
  }
}

async function handleBulkInvite() {
  if (bulkPreview.value.valid.length === 0) {
    toastStore.show('Aucun email valide à inviter', 'error')
    return
  }
  bulkLoading.value = true
  try {
    const result = await createInvitations(bulkPreview.value.valid, bulkRole.value)
    members.value.push(...result.created)
    const msg = result.created.length > 0
      ? `${result.created.length} invitation(s) envoyée(s)`
      : 'Aucune nouvelle invitation'
    toastStore.show(msg, result.created.length > 0 ? 'success' : 'info')
    if (result.skipped.length > 0) {
      toastStore.show(`${result.skipped.length} email(s) ignoré(s) (déjà dans l'équipe)`, 'info', 5000)
    }
    bulkText.value = ''
    showBulk.value = false
  } catch (e) {
    toastStore.show((e as Error).message, 'error')
  } finally {
    bulkLoading.value = false
  }
}

function handleRevoke(member: TeamMember) {
  confirmDialog.value = {
    title: 'Révoquer l\'accès',
    message: `Voulez-vous vraiment révoquer l'accès de ${member.email} ? Cette personne ne pourra plus se connecter.`,
    action: async () => {
      confirmDialog.value = null
      actionLoading.value.add(member.id)
      try {
        const updated = await updateMember(member.id, { status: 'revoked' as MemberStatus })
        const idx = members.value.findIndex(m => m.id === member.id)
        if (idx >= 0) members.value[idx] = updated
        toastStore.show('Accès révoqué', 'success')
      } catch (e) {
        toastStore.show((e as Error).message, 'error')
      } finally {
        actionLoading.value.delete(member.id)
      }
    },
  }
}

function handleRemove(member: TeamMember) {
  confirmDialog.value = {
    title: 'Supprimer le membre',
    message: `Voulez-vous vraiment retirer ${member.email} de l'équipe ?`,
    action: async () => {
      confirmDialog.value = null
      actionLoading.value.add(member.id)
      try {
        await deleteMember(member.id)
        members.value = members.value.filter(m => m.id !== member.id)
        toastStore.show('Membre supprimé', 'success')
      } catch (e) {
        toastStore.show((e as Error).message, 'error')
      } finally {
        actionLoading.value.delete(member.id)
      }
    },
  }
}

async function handleResend(member: TeamMember) {
  actionLoading.value.add(member.id)
  try {
    const updated = await resendInvitation(member.id)
    const idx = members.value.findIndex(m => m.id === member.id)
    if (idx >= 0) members.value[idx] = updated
    toastStore.show('Invitation renvoyée', 'success')
  } catch (e) {
    toastStore.show((e as Error).message, 'error')
  } finally {
    actionLoading.value.delete(member.id)
  }
}

async function handleRoleChange(member: TeamMember, newRole: Role) {
  if (newRole === member.role) return
  actionLoading.value.add(member.id)
  try {
    const updated = await updateMember(member.id, { role: newRole })
    const idx = members.value.findIndex(m => m.id === member.id)
    if (idx >= 0) members.value[idx] = updated
    toastStore.show('Rôle mis à jour', 'success')
  } catch (e) {
    toastStore.show((e as Error).message, 'error')
  } finally {
    actionLoading.value.delete(member.id)
  }
}

function formatDate(iso?: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('fr-FR', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

const memberCount = computed(() => members.value.length)

onMounted(() => {
  fetchTeam()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Gestion de l'équipe</h2>
        <p>{{ user?.companyName }} — {{ memberCount }} membre{{ memberCount > 1 ? 's' : '' }}</p>
      </div>
    </div>

    <!-- Error state -->
    <div v-if="error" class="card error-state">
      <div class="error-icon">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--critical)"><circle cx="12" cy="12" r="10" /><path d="M12 8v4M12 16h.01" /></svg>
      </div>
      <p>Erreur lors du chargement de l'équipe</p>
      <button class="btn btn-primary" @click="fetchTeam">Réessayer</button>
    </div>

    <!-- Loading skeleton -->
    <div v-else-if="loading && members.length === 0" class="card">
      <div class="skeleton skeleton-line" v-for="i in 4" :key="i" style="height: 48px; margin-bottom: 8px;"></div>
    </div>

    <template v-else>
      <!-- Invite section -->
      <div class="card invite-section">
        <h3 class="card-title">Inviter un membre</h3>
        <form @submit.prevent="handleSingleInvite" class="invite-form" novalidate>
          <div class="invite-form-row">
            <div class="form-field invite-email-field">
              <label for="invite-email">Email</label>
              <input
                id="invite-email"
                v-model="inviteEmail"
                type="email"
                placeholder="collegue@entreprise.com"
                :aria-invalid="!!inviteError"
                :aria-describedby="inviteError ? 'invite-err' : undefined"
              />
            </div>
            <div class="form-field invite-role-field">
              <label for="invite-role">Rôle</label>
              <select id="invite-role" v-model="inviteRole">
                <option value="viewer">Viewer</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <button type="submit" class="btn btn-primary invite-btn" :disabled="inviteLoading">
              <span v-if="inviteLoading" class="btn-spinner-sm"></span>
              <span v-else>Envoyer l'invitation</span>
            </button>
          </div>
          <span v-if="inviteError" id="invite-err" class="field-error" role="alert">{{ inviteError }}</span>
        </form>

        <!-- Bulk toggle -->
        <button class="bulk-toggle-btn" @click="showBulk = !showBulk">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M1 10h6M1 6h6M1 14h6M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /><circle cx="8.5" cy="7" r="4" /><path d="M20 8v6M23 11h-6" /></svg>
          {{ showBulk ? 'Masquer' : 'Ajout en masse' }}
        </button>

        <!-- Bulk form -->
        <div v-if="showBulk" class="bulk-form">
          <div class="form-field">
            <label for="bulk-textarea">Emails séparés par des virgules, espaces ou sauts de ligne</label>
            <textarea id="bulk-textarea" v-model="bulkText" rows="4" placeholder="alice@entreprise.com, bob@entreprise.com&#10;carol@entreprise.com" class="mono"></textarea>
          </div>
          <div class="bulk-controls">
            <div class="form-field bulk-role-field">
              <label for="bulk-role">Rôle</label>
              <select id="bulk-role" v-model="bulkRole">
                <option value="viewer">Viewer</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <button class="btn btn-primary" :disabled="bulkLoading || bulkPreview.valid.length === 0" @click="handleBulkInvite">
              <span v-if="bulkLoading" class="btn-spinner-sm"></span>
              <span v-else>Inviter {{ bulkPreview.valid.length }} email(s)</span>
            </button>
          </div>
          <!-- Preview -->
          <div v-if="bulkText.trim().length > 0" class="bulk-preview">
            <div v-if="bulkPreview.valid.length > 0" class="bulk-valid">
              <span class="bulk-preview-label">Valides ({{ bulkPreview.valid.length }}):</span>
              <span v-for="email in bulkPreview.valid" :key="email" class="bulk-chip bulk-chip-valid">{{ email }}</span>
            </div>
            <div v-if="bulkPreview.invalid.length > 0" class="bulk-invalid">
              <span class="bulk-preview-label">Invalides ({{ bulkPreview.invalid.length }}):</span>
              <span v-for="email in bulkPreview.invalid" :key="email" class="bulk-chip bulk-chip-invalid">{{ email }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Members table -->
      <div class="card" v-if="members.length > 0">
        <h3 class="card-title">Membres ({{ memberCount }})</h3>
        <div class="team-table-wrapper">
          <table class="team-table">
            <thead>
              <tr>
                <th>Email</th>
                <th>Rôle</th>
                <th>Statut</th>
                <th>Invité le</th>
                <th>Dernière connexion</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in members" :key="member.id" :class="{ 'row-disabled': isLoading(member.id) }">
                <td>
                  <span class="member-email mono">{{ member.email }}</span>
                </td>
                <td>
                  <select
                    v-if="canChangeRoleOf(member.role) && member.role !== 'owner' && member.status !== 'invited'"
                    :value="member.role"
                    class="role-select"
                    :disabled="isLoading(member.id)"
                    @change="handleRoleChange(member, ($event.target as HTMLSelectElement).value as Role)"
                  >
                    <option value="viewer">Viewer</option>
                    <option value="admin">Admin</option>
                  </select>
                  <span v-else class="role-badge" :class="`role-${member.role}`">{{ member.role }}</span>
                </td>
                <td>
                  <span class="member-status" :class="`status-${member.status}`">{{ member.status }}</span>
                </td>
                <td class="mono small">{{ formatDate(member.invitedAt) }}</td>
                <td class="mono small">{{ member.lastLoginAt ? formatDate(member.lastLoginAt) : '—' }}</td>
                <td>
                  <div class="action-btns">
                    <button
                      v-if="member.status === 'invited' && canRemoveMember(member.role)"
                      class="btn btn-sm"
                      :disabled="isLoading(member.id)"
                      @click.stop="handleResend(member)"
                      title="Renvoyer l'invitation"
                      aria-label="Renvoyer l'invitation"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 2L11 13" /><path d="M22 2l-7 20-4-9-9-4 20-7z" /></svg>
                    </button>
                    <button
                      v-if="member.status === 'active' && canRemoveMember(member.role)"
                      class="btn btn-warn btn-sm"
                      :disabled="isLoading(member.id)"
                      @click.stop="handleRevoke(member)"
                      title="Révoquer l'accès"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18.36 2.64a4 4 0 0 1 0 5.66l-5.66 5.66a4 4 0 0 1-5.66 0 4 4 0 0 1 0-5.66l5.66-5.66a4 4 0 0 1 5.66 0z" /><path d="M2 2l20 20" /></svg>
                    </button>
                    <button
                      v-if="canRemoveMember(member.role)"
                      class="btn btn-sm action-remove"
                      :disabled="isLoading(member.id)"
                      @click.stop="handleRemove(member)"
                      :title="member.status === 'invited' ? 'Annuler l\'invitation' : 'Supprimer'"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" /></svg>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else class="card empty-state">
        <div class="empty-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M1 10h6M1 6h6M1 14h6M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /><circle cx="8.5" cy="7" r="4" /><path d="M20 8v6M23 11h-6" /></svg>
        </div>
        <p>Aucun membre dans l'équipe</p>
        <span>Invitez des collègues avec le formulaire ci-dessus</span>
      </div>
    </template>

    <!-- Confirm dialog -->
    <Teleport to="body">
      <div v-if="confirmDialog" class="confirm-overlay" @click.self="confirmDialog = null">
        <div class="confirm-dialog" role="dialog" aria-modal="true">
          <h3>{{ confirmDialog.title }}</h3>
          <p>{{ confirmDialog.message }}</p>
          <div class="confirm-actions">
            <button class="btn btn-secondary" @click="confirmDialog = null">Annuler</button>
            <button class="btn btn-primary" @click="confirmDialog!.action()">Confirmer</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.invite-section { margin-bottom: 1.5rem; }
.invite-form { display: flex; flex-direction: column; gap: 0.5rem; }
.invite-form-row {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
  flex-wrap: wrap;
}
.form-field { display: flex; flex-direction: column; gap: 0.375rem; }
.form-field label { font-size: 0.6875rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; color: var(--text-muted); }
.invite-email-field { flex: 1; min-width: 200px; }
.invite-role-field { min-width: 120px; }
.invite-btn { height: 38px; white-space: nowrap; }
.field-error { font-size: 0.75rem; color: var(--critical); }

.bulk-toggle-btn {
  margin-top: 0.75rem;
  background: none;
  color: var(--text-muted);
  font-size: 0.75rem;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
}
.bulk-toggle-btn:hover { color: var(--accent); }

.bulk-form {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.bulk-form textarea {
  width: 100%;
  font-size: 0.8125rem;
  resize: vertical;
}
.bulk-controls {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
}
.bulk-role-field { min-width: 120px; }
.bulk-preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  font-size: 0.75rem;
}
.bulk-preview-label {
  font-weight: 600;
  color: var(--text-muted);
  margin-right: 0.25rem;
}
.bulk-valid, .bulk-invalid { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; }
.bulk-chip {
  padding: 2px 8px;
  border-radius: var(--radius-xs);
  font-family: var(--font-mono);
  font-size: 0.6875rem;
}
.bulk-chip-valid { background: var(--success-bg); color: var(--success); }
.bulk-chip-invalid { background: var(--critical-bg); color: var(--critical); }

.team-table-wrapper { overflow-x: auto; }
.team-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.8125rem;
}
.team-table th {
  text-align: left;
  padding: 0.625rem 0.75rem;
  font-size: 0.6875rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
.team-table td {
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid rgba(30, 42, 69, 0.4);
  vertical-align: middle;
}
.row-disabled { opacity: 0.6; }
.member-email { color: var(--text); }
.invite-link {
  display: block;
  font-size: 0.625rem;
  color: var(--accent);
  margin-top: 2px;
}
.role-select {
  font-size: 0.75rem;
  padding: 3px 8px;
  width: auto;
}
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

.member-status {
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: var(--radius-xs);
}
.status-invited { background: var(--warning-bg); color: var(--warning); }
.status-active { background: var(--success-bg); color: var(--success); }
.status-revoked { background: var(--critical-bg); color: var(--critical); }

.small { font-size: 0.75rem; white-space: nowrap; }
.action-btns { display: flex; gap: 4px; }
.action-remove { background: var(--card-hover); color: var(--critical); }
.action-remove:hover:not(:disabled) { background: var(--critical-bg); }

.btn-spinner-sm {
  display: inline-block;
  width: 14px; height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Confirm dialog */
.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  z-index: 300;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}
.confirm-dialog {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 1.5rem;
  max-width: 400px;
  width: 100%;
  box-shadow: var(--shadow-lg);
}
.confirm-dialog h3 { font-size: 1rem; font-weight: 700; margin-bottom: 0.75rem; }
.confirm-dialog p { font-size: 0.875rem; color: var(--text-dim); line-height: 1.5; margin-bottom: 1.25rem; }
.confirm-actions { display: flex; gap: 0.75rem; justify-content: flex-end; }

@media (max-width: 768px) {
  .invite-form-row { flex-direction: column; align-items: stretch; }
  .invite-btn { width: 100%; }
  .bulk-controls { flex-direction: column; }
}
</style>
