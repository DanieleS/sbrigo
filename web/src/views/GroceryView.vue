<script setup lang="ts">
import { computed, ref } from 'vue'
import QuickAdd from '../components/QuickAdd.vue'
import TaskEditor from '../components/TaskEditor.vue'
import TaskRow from '../components/TaskRow.vue'
import { useCatalogStore } from '../stores/catalog'
import { useTasksStore } from '../stores/tasks'
import type { Department, Task, TaskFields } from '../types'

const catalog = useCatalogStore()
const tasks = useTasksStore()
const editing = ref<Task | null>(null)
const lastDepartment = ref(localStorage.getItem('sbrigo.lastDepartment') ?? undefined)

const items = computed(() => tasks.byType('grocery'))
const completedCount = computed(() => items.value.filter((t) => t.is_completed).length)

interface Group {
  department: Department | null
  items: Task[]
}

/** Departments in aisle order for the selected supermarket; empty departments are hidden. */
const groups = computed<Group[]>(() => {
  const byDep = new Map<string | null, Task[]>()
  for (const t of items.value) {
    const key = t.department_id && catalog.departmentById(t.department_id) ? t.department_id : null
    byDep.set(key, [...(byDep.get(key) ?? []), t])
  }
  const sortItems = (list: Task[]) =>
    list.sort((a, b) => Number(a.is_completed) - Number(b.is_completed) || a.created_at.localeCompare(b.created_at))
  const out: Group[] = []
  for (const d of catalog.orderedDepartments) {
    const list = byDep.get(d.id)
    if (list) out.push({ department: d, items: sortItems(list) })
  }
  const none = byDep.get(null)
  if (none) out.push({ department: null, items: sortItems(none) })
  return out
})

async function add(title: string, departmentId: string | null) {
  if (departmentId) localStorage.setItem('sbrigo.lastDepartment', departmentId)
  lastDepartment.value = departmentId ?? undefined
  await tasks.add({ title, item_type: 'grocery', department_id: departmentId })
}

async function save(patch: Partial<TaskFields>) {
  if (editing.value) await tasks.patch(editing.value.id, patch)
  editing.value = null
}

async function remove() {
  if (editing.value) await tasks.removeTask(editing.value.id)
  editing.value = null
}

async function clearCompleted() {
  if (confirm(`Rimuovere ${completedCount.value} articoli spuntati?`)) await tasks.clearCompleted('grocery')
}
</script>

<template>
  <main class="page">
    <header class="topbar">
      <h1>Spesa</h1>
      <select
        class="select"
        style="width: auto; max-width: 55%"
        aria-label="Supermercato"
        :value="catalog.selectedSupermarketId"
        @change="catalog.selectSupermarket(($event.target as HTMLSelectElement).value)"
      >
        <option value="">Ordine predefinito</option>
        <option v-for="m in catalog.supermarkets" :key="m.id" :value="m.id">{{ m.name }}</option>
      </select>
    </header>

    <p v-if="items.length === 0" class="empty">
      La lista è vuota.<br />Aggiungi il primo articolo qui sotto.
    </p>

    <section v-for="g in groups" :key="g.department?.id ?? 'none'" class="group">
      <div class="group-head">
        <h2>{{ g.department?.name ?? 'Senza reparto' }}</h2>
        <p v-if="g.department?.description">{{ g.department.description }}</p>
      </div>
      <div class="card">
        <TaskRow v-for="t in g.items" :key="t.id" :task="t" @toggle="tasks.toggle(t.id)" @open="editing = t" />
      </div>
    </section>

    <div v-if="completedCount > 0" class="row" style="justify-content: center; margin-top: 8px">
      <button class="btn small" @click="clearCompleted">Rimuovi spuntati ({{ completedCount }})</button>
    </div>

    <QuickAdd placeholder="Aggiungi alla spesa…" with-department :default-department="lastDepartment" @add="add" />

    <TaskEditor v-if="editing" :task="editing" @save="save" @remove="remove" @close="editing = null" />
  </main>
</template>
