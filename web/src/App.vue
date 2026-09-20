<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useAuthStore } from './stores/auth'
import { useCatalogStore } from './stores/catalog'
import { useTasksStore } from './stores/tasks'
import { connect, disconnect } from './sse'
import { kick } from './sync'
import { useOnline } from './useOnline'

const auth = useAuthStore()
const catalog = useCatalogStore()
const tasks = useTasksStore()
const online = useOnline()

const showApp = computed(() => auth.status === 'authenticated' || auth.status === 'offline')

onMounted(async () => {
  await Promise.all([catalog.init(), tasks.init()])
  await auth.load()
})

watch(
  () => auth.status,
  (status) => {
    if (status === 'authenticated') {
      connect()
      kick()
      void catalog.refresh()
      void tasks.refreshFromServer()
    } else {
      disconnect()
    }
  },
)

// Back online after an offline start: re-check the session, which then wires everything up.
watch(online, (isOnline) => {
  if (isOnline && auth.status === 'offline') void auth.load()
})
</script>

<template>
  <div v-if="auth.status === 'loading'" class="login">
    <p class="muted">Caricamento…</p>
  </div>

  <div v-else-if="auth.status === 'anonymous'" class="login">
    <h1>Sbrigo</h1>
    <p class="muted">Liste e attività di casa, condivise in famiglia.</p>
    <button class="btn primary" @click="auth.login()">Accedi</button>
  </div>

  <template v-else-if="showApp">
    <div v-if="!online" class="banner warn">Sei offline: le modifiche verranno inviate appena torni in rete.</div>
    <div v-else-if="tasks.pending > 0" class="banner">Sincronizzazione di {{ tasks.pending }} modifiche…</div>

    <router-view />

    <nav class="nav">
      <router-link to="/" aria-label="Spesa">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M3 4h2l2.4 11.2a2 2 0 0 0 2 1.6h8.8a2 2 0 0 0 2-1.5L22 8H6" />
          <circle cx="10" cy="20" r="1.5" />
          <circle cx="18" cy="20" r="1.5" />
        </svg>
        Spesa
      </router-link>
      <router-link to="/attivita" aria-label="Attività">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M9 6h11M9 12h11M9 18h11" />
          <path d="M4 6l1 1 2-2M4 12l1 1 2-2M4 18l1 1 2-2" />
        </svg>
        Attività
      </router-link>
      <router-link to="/impostazioni" aria-label="Impostazioni">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <circle cx="12" cy="12" r="3" />
          <path
            d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"
          />
        </svg>
        Impostazioni
      </router-link>
    </nav>
  </template>
</template>
