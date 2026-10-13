import { computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import type { Role } from '../types'

export function useCan() {
  const authStore = useAuthStore()
  const role = computed<Role | null>(() => authStore.user?.role ?? null)

  const isOwner = computed(() => role.value === 'owner')
  const isAdmin = computed(() => role.value === 'owner' || role.value === 'admin')
  const isViewer = computed(() => role.value === 'viewer')

  const canManageTeam = computed(() => isAdmin.value)
  const canEditRoles = computed(() => isOwner.value)
  const canRemoveMember = (memberRole: Role): boolean => {
    if (role.value === 'owner') return true
    if (role.value === 'admin') return memberRole === 'viewer'
    return false
  }
  const canChangeRoleOf = (memberRole: Role): boolean => {
    if (role.value === 'owner') return true
    if (role.value === 'admin') return memberRole === 'viewer'
    return false
  }

  return {
    role,
    isOwner,
    isAdmin,
    isViewer,
    canManageTeam,
    canEditRoles,
    canRemoveMember,
    canChangeRoleOf,
  }
}
