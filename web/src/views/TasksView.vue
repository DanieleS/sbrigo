<script setup lang="ts">
import { computed, ref } from 'vue'
import { CollapsibleContent, CollapsibleRoot, CollapsibleTrigger } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListSwitcher from '../components/ListSwitcher.vue'
import PageHeader from '../components/PageHeader.vue'
import QuickAdd from '../components/QuickAdd.vue'
import TaskEditor from '../components/TaskEditor.vue'
import TaskRow from '../components/TaskRow.vue'
import { daysFromToday, formatDue } from '../format'
import { summarize } from '../listSummary'
import { useCatalogStore } from '../stores/catalog'
import { useTasksStore } from '../stores/tasks'
import type { Task, TaskFields } from '../types'

const { t } = useI18n()
const catalog = useCatalogStore()
const tasks = useTasksStore()
const editing = ref<Task | null>(null)
const showDone = ref(false)
const confirmClear = ref(false)

const currentList = computed(() => catalog.currentList('general_task'))
const items = computed(() => (currentList.value ? tasks.byList(currentList.value.id) : []))
const byDue = (a: Task, b: Task) =>
  (a.due_date ?? '9').localeCompare(b.due_date ?? '9') || a.created_at.localeCompare(b.created_at)

interface Bucket {
  key: string
  label: string
  danger?: boolean
  items: Task[]
}

const buckets = computed<Bucket[]>(() => {
  const open = items.value.filter((x) => !x.is_completed).sort(byDue)
  const pick = (test: (diff: number | null) => boolean) => open.filter((x) => test(daysFromToday(x.due_date)))
  const out: Bucket[] = [
    { key: 'overdue', label: t('tasks.overdue'), danger: true, items: pick((d) => d !== null && d < 0) },
    { key: 'today', label: t('tasks.today'), items: pick((d) => d === 0) },
    { key: 'week', label: t('tasks.thisWeek'), items: pick((d) => d !== null && d > 0 && d <= 7) },
    { key: 'later', label: t('tasks.later'), items: pick((d) => d !== null && d > 7) },
    { key: 'nodate', label: t('tasks.noDate'), items: pick((d) => d === null) },
  ]
  return out.filter((b) => b.items.length > 0)
})
const done = computed(() =>
  items.value.filter((x) => x.is_completed).sort((a, b) => b.updated_at.localeCompare(a.updated_at)),
)

const summary = computed(() => {
  if (!currentList.value) return ''
  const s = summarize(currentList.value.id)
  if (s.overdue > 0) return t('tasks.summaryOverdue', { open: s.open, overdue: s.overdue })
  return s.nextDue
    ? t('tasks.summary', { open: s.open, next: formatDue(s.nextDue) })
    : t('tasks.summaryNoNext', { open: s.open })
})

async function add(title: string) {
  if (!currentList.value) return
  editing.value = await tasks.add({ title, item_type: 'general_task', list_id: currentList.value.id })
}
async function save(patch: Partial<TaskFields>) {
  if (editing.value) await tasks.patch(editing.value.id, patch)
  editing.value = null
}
async function remove() {
  if (editing.value) await tasks.removeTask(editing.value.id)
  editing.value = null
}
async function clearDone() {
  confirmClear.value = false
  if (currentList.value) await tasks.clearCompleted(currentList.value.id)
}
</script>

<template>
  <main class="page" style="--dock-h: 70px">
    <PageHeader :subtitle="summary">
      <template #title>
        <ListSwitcher kind="general_task" />
      </template>
    </PageHeader>

    <div v-if="items.length === 0" class="empty">
      <strong>{{ t('tasks.empty') }}</strong
      >{{ t('tasks.emptyHint') }}
    </div>

    <section v-for="b in buckets" :key="b.key" class="group">
      <div class="group-head" :class="{ danger: b.danger }">
        <h2 class="grow">{{ b.label }}</h2>
        <span class="count">{{ b.items.length }}</span>
      </div>
      <div class="card">
        <TaskRow
          v-for="x in b.items"
          :key="x.id"
          :task="x"
          show-details
          @toggle="tasks.toggle(x.id)"
          @open="editing = x"
        />
      </div>
    </section>

    <CollapsibleRoot v-if="done.length" v-model:open="showDone" class="group">
      <div class="group-head">
        <CollapsibleTrigger class="btn ghost small" style="padding: 0 4px; gap: 8px">
          <h2 style="margin: 0">{{ t('tasks.completed') }}</h2>
          <span class="count">{{ done.length }}</span>
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.4"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
            :style="{ transform: showDone ? 'rotate(180deg)' : '' }"
          >
            <path d="M6 9l6 6 6-6" />
          </svg>
        </CollapsibleTrigger>
        <span class="grow" />
        <button v-if="showDone" class="btn small danger" @click="confirmClear = true">
          {{ t('tasks.clearCompleted') }}
        </button>
      </div>
      <CollapsibleContent class="card">
        <TaskRow
          v-for="x in done"
          :key="x.id"
          :task="x"
          show-details
          @toggle="tasks.toggle(x.id)"
          @open="editing = x"
        />
      </CollapsibleContent>
    </CollapsibleRoot>

    <div class="dock">
      <div class="dock-inner">
        <QuickAdd :placeholder="t('tasks.placeholder')" @add="add" />
      </div>
    </div>

    <TaskEditor v-if="editing" :task="editing" @save="save" @remove="remove" @close="editing = null" />
    <ConfirmDialog
      :open="confirmClear"
      :title="t('tasks.clearCompletedTitle', { n: done.length })"
      @confirm="clearDone"
      @cancel="confirmClear = false"
    />
  </main>
</template>
