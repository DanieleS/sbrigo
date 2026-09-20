<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { initials } from '../format'
import { useAuthStore } from '../stores/auth'
import { useTasksStore } from '../stores/tasks'
import { useOnline } from '../useOnline'

defineProps<{ title?: string; subtitle?: string }>()
const { t } = useI18n()
const auth = useAuthStore()
const tasks = useTasksStore()
const online = useOnline()

const status = computed(() => {
  if (!online.value) return { cls: 'offline', text: t('status.offline') }
  if (tasks.pending > 0) return { cls: 'syncing', text: t('status.syncing', { n: tasks.pending }) }
  return { cls: '', text: t('status.online') }
})
const name = computed(() => auth.user?.display_name || auth.user?.email || '?')
</script>

<template>
  <header class="page-head">
    <div class="grow">
      <slot name="title">
        <h1>{{ title }}</h1>
      </slot>
      <p v-if="subtitle" class="sub">{{ subtitle }}</p>
    </div>
    <slot name="actions" />
    <div style="display: flex; flex-direction: column; align-items: flex-end; gap: 6px">
      <router-link to="/impostazioni" class="avatar" :aria-label="t('nav.settings')">{{ initials(name) }}</router-link>
      <span class="status" :class="status.cls" role="status">{{ status.text }}</span>
    </div>
  </header>
</template>
