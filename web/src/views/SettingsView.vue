<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useCatalogStore } from '../stores/catalog'
import { useTasksStore } from '../stores/tasks'
import type { Department } from '../types'

const auth = useAuthStore()
const catalog = useCatalogStore()
const tasks = useTasksStore()

const error = ref('')
async function guard(fn: () => Promise<void>) {
  error.value = ''
  try {
    await fn()
  } catch (err) {
    error.value = navigator.onLine ? 'Operazione non riuscita.' : 'Serve la connessione per modificare reparti e supermercati.'
    console.error(err)
  }
}

// Supermarkets
const newSupermarket = ref('')
const orderingId = ref<string | null>(null)
const draftOrder = ref<string[]>([])

function startOrdering(id: string) {
  orderingId.value = id
  const explicit = catalog.orders[id] ?? []
  const rest = catalog.departments
    .filter((d) => !explicit.includes(d.id))
    .sort((a, b) => a.default_sort_order - b.default_sort_order)
    .map((d) => d.id)
  draftOrder.value = [...explicit.filter((depId) => catalog.departmentById(depId)), ...rest]
}
function move(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= draftOrder.value.length) return
  const list = [...draftOrder.value]
  ;[list[index], list[target]] = [list[target]!, list[index]!]
  draftOrder.value = list
}
async function saveOrder() {
  const id = orderingId.value
  if (!id) return
  await guard(() => catalog.setDepartmentOrder(id, draftOrder.value))
  orderingId.value = null
}
async function renameSupermarket(id: string, current: string) {
  const name = prompt('Nuovo nome', current)?.trim()
  if (name && name !== current) await guard(() => catalog.renameSupermarket(id, name))
}
async function deleteSupermarket(id: string, name: string) {
  if (confirm(`Eliminare "${name}"?`)) await guard(() => catalog.deleteSupermarket(id))
}

// Departments
const editingDep = ref<Department | null>(null)
const depForm = reactive({ name: '', description: '', default_sort_order: 0 })
const sortedDepartments = computed(() =>
  [...catalog.departments].sort((a, b) => a.default_sort_order - b.default_sort_order || a.name.localeCompare(b.name)),
)
function editDepartment(d: Department | null) {
  editingDep.value = d ?? { id: '', name: '', description: '', default_sort_order: (sortedDepartments.value.at(-1)?.default_sort_order ?? 0) + 10 }
  Object.assign(depForm, { name: editingDep.value.name, description: editingDep.value.description, default_sort_order: editingDep.value.default_sort_order })
}
async function saveDepartment() {
  const d = editingDep.value
  if (!d || !depForm.name.trim()) return
  const payload = { name: depForm.name.trim(), description: depForm.description.trim(), default_sort_order: Number(depForm.default_sort_order) || 0 }
  await guard(() => (d.id ? catalog.updateDepartment({ id: d.id, ...payload }) : catalog.createDepartment(payload)))
  editingDep.value = null
}
async function deleteDepartment(d: Department) {
  if (confirm(`Eliminare il reparto "${d.name}"? Gli articoli resteranno senza reparto.`)) {
    await guard(() => catalog.deleteDepartment(d.id))
    editingDep.value = null
  }
}

async function reload() {
  await guard(async () => {
    await catalog.refresh()
    await tasks.refreshFromServer()
  })
}
</script>

<template>
  <main class="page">
    <header class="topbar"><h1>Impostazioni</h1></header>

    <p v-if="error" class="banner warn">{{ error }}</p>

    <section class="section">
      <h2>Account</h2>
      <div class="card">
        <div class="list-item">
          <div class="grow">
            <div>{{ auth.user?.display_name || auth.user?.email || 'Utente' }}</div>
            <div class="small muted">{{ auth.user?.email }}</div>
          </div>
          <button class="btn small" @click="auth.logout()">Esci</button>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>Supermercati</h2>
      <p class="small muted">Per ogni supermercato puoi impostare l'ordine reale delle corsie.</p>
      <div class="card">
        <template v-for="m in catalog.supermarkets" :key="m.id">
          <div class="list-item">
            <div class="grow">{{ m.name }}</div>
            <button class="btn small" @click="orderingId === m.id ? (orderingId = null) : startOrdering(m.id)">Corsie</button>
            <button class="btn small" @click="renameSupermarket(m.id, m.name)">Rinomina</button>
            <button class="btn small danger" @click="deleteSupermarket(m.id, m.name)">Elimina</button>
          </div>
          <div v-if="orderingId === m.id" style="padding: 8px 12px; background: var(--surface-2)">
            <div v-for="(depId, i) in draftOrder" :key="depId" class="row" style="padding: 4px 0">
              <span class="muted small" style="width: 22px">{{ i + 1 }}.</span>
              <span style="flex: 1">{{ catalog.departmentById(depId)?.name }}</span>
              <button class="btn small icon" aria-label="Sposta su" :disabled="i === 0" @click="move(i, -1)">▲</button>
              <button class="btn small icon" aria-label="Sposta giù" :disabled="i === draftOrder.length - 1" @click="move(i, 1)">▼</button>
            </div>
            <div class="row" style="justify-content: flex-end; margin-top: 8px">
              <button class="btn small" @click="orderingId = null">Annulla</button>
              <button class="btn small primary" @click="saveOrder">Salva ordine</button>
            </div>
          </div>
        </template>
        <form class="list-item" @submit.prevent="guard(() => catalog.createSupermarket(newSupermarket.trim())).then(() => (newSupermarket = ''))">
          <input v-model="newSupermarket" class="input" placeholder="Nuovo supermercato" />
          <button class="btn primary" type="submit" :disabled="!newSupermarket.trim()">Aggiungi</button>
        </form>
      </div>
    </section>

    <section class="section">
      <h2>Reparti</h2>
      <p class="small muted">La descrizione è la legenda mostrata sotto il nome del reparto nella lista.</p>
      <div class="card">
        <div v-for="d in sortedDepartments" :key="d.id" class="list-item">
          <div class="grow">
            <div>{{ d.name }}</div>
            <div class="small muted">{{ d.description }}</div>
          </div>
          <button class="btn small" @click="editDepartment(d)">Modifica</button>
        </div>
        <div class="list-item">
          <button class="btn" style="width: 100%" @click="editDepartment(null)">Nuovo reparto</button>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>Dati</h2>
      <div class="card">
        <div class="list-item">
          <div class="grow small muted">
            {{ tasks.all.length }} elementi in locale<span v-if="tasks.pending"> · {{ tasks.pending }} modifiche in attesa</span>
          </div>
          <button class="btn small" @click="reload">Ricarica dal server</button>
        </div>
      </div>
    </section>

    <div v-if="editingDep" class="sheet-backdrop" @click.self="editingDep = null">
      <form class="sheet" @submit.prevent="saveDepartment">
        <h3>{{ editingDep.id ? 'Modifica reparto' : 'Nuovo reparto' }}</h3>
        <div class="field">
          <label for="dep-name">Nome</label>
          <input id="dep-name" v-model="depForm.name" class="input" required />
        </div>
        <div class="field">
          <label for="dep-desc">Legenda</label>
          <textarea id="dep-desc" v-model="depForm.description" class="textarea" />
        </div>
        <div class="field">
          <label for="dep-order">Ordine predefinito</label>
          <input id="dep-order" v-model.number="depForm.default_sort_order" class="input" type="number" />
        </div>
        <div class="actions">
          <button v-if="editingDep.id" type="button" class="btn danger" @click="deleteDepartment(editingDep)">Elimina</button>
          <span class="spacer" />
          <button type="button" class="btn" @click="editingDep = null">Annulla</button>
          <button type="submit" class="btn primary">Salva</button>
        </div>
      </form>
    </div>
  </main>
</template>
