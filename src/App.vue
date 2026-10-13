<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useAuthStore } from './stores/auth'
import { useToastStore } from './stores/toast'
import ToastContainer from './components/ToastContainer.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const toastStore = useToastStore()

const { user, addresses, selectedAddress, isAuthenticated } = storeToRefs(authStore)
const sidebarOpen = ref(false)

const canManageTeam = computed(() => user.value?.role === 'owner' || user.value?.role === 'admin')

const allNavItems = [
  { name: 'dashboard', label: 'Dashboard', path: '/', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 0 0 1 1h3m10-11l2 2m-2-2v10a1 1 0 0 1-1 1h-3m-6 0a1 1 0 0 0 1-1v-4a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1v4a1 1 0 0 0 1 1m-6 0h6' },
  { name: 'events', label: 'Événements', path: '/events', icon: 'M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2M9 5a2 2 0 0 0 2 2h2a2 2 0 0 0 2-2M9 5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2m-6 9l2 2 4-4' },
  { name: 'alerts', label: 'Alertes', path: '/alerts', icon: 'M15 17h5l-1.405-1.405A2.032 2.032 0 0 1 18 14.158V11a6.002 6.002 0 0 0-4-5.659V5a2 2 0 1 0-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 1 1-6 0v-1m6 0H9' },
  { name: 'addresses', label: 'Adresses', path: '/addresses', icon: 'M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z M12 6a3 3 0 1 0 0 6 3 3 0 0 0 0-6z' },
  { name: 'team', label: 'Équipe', path: '/team', icon: 'M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M1 10h6M1 6h6M1 14h6M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75' },
]

const navItems = computed(() =>
  allNavItems.filter(item => item.name !== 'team' || canManageTeam.value)
)

async function handleLogout() {
  // logout() révoque la session côté serveur et nettoie tous les stores
  await authStore.logout()
  toastStore.show('Déconnecté', 'info')
  router.push('/login')
  sidebarOpen.value = false
}

function handleAddressChange(value: string) {
  authStore.setSelectedAddress(value)
}

// Ferme la sidebar mobile lors d'un changement de route
watch(() => route.fullPath, () => {
  sidebarOpen.value = false
})
</script>

<template>
  <!-- Pages publiques : rendu sans layout -->
  <template v-if="route.meta.public">
    <RouterView />
    <ToastContainer />
  </template>

  <!-- Layout principal avec sidebar + header -->
  <template v-else>
    <div class="app-layout">
      <!-- Bouton menu mobile -->
      <button
        v-if="isAuthenticated"
        class="mobile-menu-btn"
        @click="sidebarOpen = !sidebarOpen"
        aria-label="Ouvrir le menu"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12h18M3 6h18M3 18h18" /></svg>
      </button>

      <!-- Overlay mobile pour fermer la sidebar -->
      <div
        v-if="sidebarOpen"
        class="sidebar-mobile-overlay"
        @click="sidebarOpen = false"
      ></div>

      <!-- Sidebar -->
      <aside class="sidebar" :class="{ open: sidebarOpen }" v-if="isAuthenticated">
        <div class="sidebar-brand">
          <div class="sidebar-brand-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><path d="M12 2L4 6v6c0 5.5 3.5 10 8 12 4.5-2 8-6.5 8-12V6l-8-4z" /><path d="M9 12l2 2 4-4" /></svg>
          </div>
          <div class="sidebar-brand-text">
            <h1>Mini SIEM</h1>
            <span>Security Console</span>
          </div>
        </div>

        <nav class="sidebar-nav" aria-label="Navigation principale">
          <RouterLink
            v-for="item in navItems"
            :key="item.name"
            :to="item.path"
            class="nav-link"
            :class="{ active: route.name === item.name }"
            @click="sidebarOpen = false"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path :d="item.icon" />
            </svg>
            {{ item.label }}
          </RouterLink>
        </nav>

        <div class="sidebar-footer">
          <div class="sidebar-user" v-if="user">
            <div class="sidebar-user-info">
              <span class="sidebar-user-email">{{ user.email }}</span>
              <span class="sidebar-user-company">{{ user.companyName }}</span>
            </div>
            <button class="sidebar-logout" @click="handleLogout" aria-label="Se déconnecter">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" /></svg>
            </button>
          </div>
        </div>
      </aside>

      <!-- Main content area -->
      <div class="main-area" v-if="isAuthenticated">
        <!-- Header -->
        <header class="topbar">
          <div class="topbar-left">
            <span class="topbar-company">{{ user?.companyName }}</span>
          </div>

          <div class="topbar-right">
            <!-- Sélecteur d'adresse global -->
            <div class="address-selector">
              <label for="address-select" class="address-selector-label">Adresse</label>
              <select
                id="address-select"
                :value="selectedAddress"
                @change="handleAddressChange(($event.target as HTMLSelectElement).value)"
              >
                <option value="">Toutes</option>
                <option v-for="addr in addresses" :key="addr.id" :value="addr.value">
                  {{ addr.value }}
                </option>
              </select>
            </div>

            <!-- Infos utilisateur -->
            <div class="user-badge">
              <div class="user-avatar">{{ user?.email.charAt(0).toUpperCase() }}</div>
              <div class="user-details">
                <span class="user-email">{{ user?.email }}</span>
                <span class="user-role">{{ user?.role }}</span>
              </div>
            </div>

            <button class="logout-btn" @click="handleLogout" aria-label="Se déconnecter">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" /></svg>
            </button>
          </div>
        </header>

        <!-- Page content -->
        <main class="main-content">
          <RouterView v-slot="{ Component }">
            <Transition name="route" mode="out-in">
              <component :is="Component" />
            </Transition>
          </RouterView>
        </main>
      </div>
    </div>

    <ToastContainer />
  </template>
</template>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
}

.sidebar-mobile-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 99;
}

.main-area {
  flex: 1;
  margin-left: var(--sidebar-width);
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

/* ---- Topbar ---- */
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  height: 56px;
  background: var(--card);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  gap: 1rem;
}
.topbar-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.topbar-company {
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--text);
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.address-selector {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.address-selector-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.address-selector select {
  font-size: 0.8125rem;
  padding: 5px 10px;
  min-width: 160px;
}

.user-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 4px 10px 4px 4px;
  border-radius: 20px;
  background: var(--bg);
  border: 1px solid var(--border);
}
.user-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--accent), var(--accent-dim));
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 700;
  color: #fff;
}
.user-details {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}
.user-email {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text);
}
.user-role {
  font-size: 0.625rem;
  color: var(--text-muted);
  text-transform: uppercase;
}

.logout-btn {
  background: none;
  color: var(--text-dim);
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  transition: all 0.15s;
}
.logout-btn:hover {
  background: var(--critical-bg);
  color: var(--critical);
}

/* ---- Sidebar footer user ---- */
.sidebar-user {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}
.sidebar-user-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: hidden;
}
.sidebar-user-email {
  font-size: 0.75rem;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sidebar-user-company {
  font-size: 0.625rem;
  color: var(--text-muted);
}
.sidebar-logout {
  background: none;
  color: var(--text-muted);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  transition: all 0.15s;
}
.sidebar-logout:hover {
  background: var(--critical-bg);
  color: var(--critical);
}

/* ---- Route transition ---- */
.route-enter-active,
.route-leave-active {
  transition: opacity 0.18s ease;
}
.route-enter-from,
.route-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .route-enter-active,
  .route-leave-active {
    transition: none;
  }
}

/* ---- Responsive ---- */
@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
    transition: transform 0.25s ease;
  }
  .sidebar.open {
    transform: translateX(0);
  }
  .sidebar-mobile-overlay {
    display: block;
  }
  .main-area {
    margin-left: 0;
  }
  .mobile-menu-btn {
    display: flex;
  }
  .topbar {
    padding: 0 0.75rem;
    height: auto;
    flex-wrap: wrap;
    padding-top: 0.5rem;
    padding-bottom: 0.5rem;
  }
  .topbar-right {
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .user-details {
    display: none;
  }
  .address-selector select {
    min-width: 120px;
  }
}
</style>
