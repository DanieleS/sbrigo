<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppToaster from './components/AppToaster.vue'
import { useAuthStore } from './stores/auth'
import { useCatalogStore } from './stores/catalog'
import { useTasksStore } from './stores/tasks'
import { applyTheme, useUiStore } from './stores/ui'
import { connect, disconnect } from './sse'
import { kick } from './sync'
import { useOnline } from './useOnline'

const { t } = useI18n()
const auth = useAuthStore()
const catalog = useCatalogStore()
const tasks = useTasksStore()
const ui = useUiStore()
const online = useOnline()

const showApp = computed(() => auth.status === 'authenticated' || auth.status === 'offline')

applyTheme(ui.theme)

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
    <p class="muted">{{ t('common.loading') }}</p>
  </div>

  <div v-else-if="auth.status === 'anonymous'" class="login">
    <h1>{{ t('app.name') }}</h1>
    <p class="muted">{{ t('app.tagline') }}</p>
    <button class="btn primary" style="min-width: 200px" @click="auth.login()">{{ t('common.login') }}</button>
  </div>

  <template v-else-if="showApp">
    <router-view />

    <nav class="nav" :aria-label="t('nav.grocery') + ' / ' + t('nav.tasks') + ' / ' + t('nav.settings')">
      <router-link to="/">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M3 4h2l2.4 11.2a2 2 0 0 0 2 1.6h8.8a2 2 0 0 0 2-1.5L22 8H6" />
          <circle cx="10" cy="20" r="1.5" />
          <circle cx="18" cy="20" r="1.5" />
        </svg>
        {{ t('nav.grocery') }}
      </router-link>
      <router-link to="/attivita">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M9 6h11M9 12h11M9 18h11" />
          <path d="M4 6l1 1 2-2M4 12l1 1 2-2M4 18l1 1 2-2" />
        </svg>
        {{ t('nav.tasks') }}
      </router-link>
      <router-link to="/impostazioni">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="3" />
          <path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1L7 17M17 7l2.1-2.1" />
        </svg>
        {{ t('nav.settings') }}
      </router-link>
    </nav>
  </template>

  <AppToaster />
</template>
