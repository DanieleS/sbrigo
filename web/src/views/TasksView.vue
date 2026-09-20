<script setup lang="ts">
import { computed, ref } from 'vue'
import QuickAdd from '../components/QuickAdd.vue'
import TaskEditor from '../components/TaskEditor.vue'
import TaskRow from '../components/TaskRow.vue'
import { useTasksStore } from '../stores/tasks'
import type { Task, TaskFields } from '../types'

const tasks = useTasksStore()
const editing = ref<Task | null>(null)
const showDone = ref(false)

const items = computed(() => tasks.byType('general_task'))
const open = computed(() =>
  items.value
    .filter((t) => !t.is_completed)
    .sort((a, b) => (a.due_date ?? '9').localeCompare(b.due_date ?? '9') || a.created_at.localeCompare(b.created_at)),
)
const done = computed(() =>
  items.value.filter((t) => t.is_completed).sort((a, b) => b.updated_at.localeCompare(a.updated_at)),
)

async function add(title: string) {
  editing.value = await tasks.add({ title, item_type: 'general_task' })
}

async function save(patch: Partial<TaskFields>) {
  if (editing.value) await tasks.patch(editing.value.id, patch)
  editing.value = null
}

async function remove() {
  if (editing.value) await tasks.removeTask(editing.value.id)
  editing.value = null
}
</script>

<template>
  <main class="page">
    <header class="topbar">
      <h1>Attività</h1>
    </header>

    <p v-if="items.length === 0" class="empty">
      Nessuna attività.<br />Bollette, pratiche, cose da fare in casa: aggiungile qui sotto.
    </p>

    <section v-if="open.length" class="group">
      <div class="card">
        <TaskRow
          v-for="t in open"
          :key="t.id"
          :task="t"
          show-details
          @toggle="tasks.toggle(t.id)"
          @open="editing = t"
        />
      </div>
    </section>

    <section v-if="done.length" class="group">
      <div class="group-head row">
        <h2 style="flex: 1">Completate ({{ done.length }})</h2>
        <button class="btn small" @click="showDone = !showDone">{{ showDone ? 'Nascondi' : 'Mostra' }}</button>
        <button v-if="showDone" class="btn small danger" @click="tasks.clearCompleted('general_task')">Svuota</button>
      </div>
      <div v-if="showDone" class="card">
        <TaskRow
          v-for="t in done"
          :key="t.id"
          :task="t"
          show-details
          @toggle="tasks.toggle(t.id)"
          @open="editing = t"
        />
      </div>
    </section>

    <QuickAdd placeholder="Nuova attività…" @add="add" />

    <TaskEditor v-if="editing" :task="editing" @save="save" @remove="remove" @close="editing = null" />
  </main>
</template>
