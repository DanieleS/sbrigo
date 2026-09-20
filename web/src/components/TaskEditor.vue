<script setup lang="ts">
import { reactive } from 'vue'
import type { Task, TaskFields } from '../types'
import { fromDateInput, toDateInput } from '../format'
import { NO_DEPARTMENT, useCatalogStore } from '../stores/catalog'

const props = defineProps<{ task: Task }>()
const emit = defineEmits<{ save: [patch: Partial<TaskFields>]; remove: []; close: [] }>()
const catalog = useCatalogStore()

const form = reactive({
  title: props.task.title,
  notes: props.task.notes ?? '',
  department_id: props.task.department_id ?? NO_DEPARTMENT,
  assignee_id: props.task.assignee_id ?? '',
  due_date: toDateInput(props.task.due_date),
  item_type: props.task.item_type,
})

function save() {
  const title = form.title.trim()
  if (!title) return
  emit('save', {
    title,
    notes: form.notes.trim() || null,
    department_id: form.department_id === NO_DEPARTMENT ? null : form.department_id,
    assignee_id: form.assignee_id || null,
    due_date: fromDateInput(form.due_date),
    item_type: form.item_type,
  })
}
</script>

<template>
  <div class="sheet-backdrop" @click.self="emit('close')">
    <form class="sheet" @submit.prevent="save">
      <h3>Modifica</h3>
      <div class="field">
        <label for="te-title">Titolo</label>
        <input id="te-title" v-model="form.title" class="input" type="text" required />
      </div>
      <div class="field">
        <label for="te-type">Tipo</label>
        <select id="te-type" v-model="form.item_type" class="select">
          <option value="grocery">Spesa</option>
          <option value="general_task">Attività</option>
        </select>
      </div>
      <div v-if="form.item_type === 'grocery'" class="field">
        <label for="te-dep">Reparto</label>
        <select id="te-dep" v-model="form.department_id" class="select">
          <option :value="NO_DEPARTMENT">Nessun reparto</option>
          <option v-for="d in catalog.orderedDepartments" :key="d.id" :value="d.id">{{ d.name }}</option>
        </select>
      </div>
      <template v-else>
        <div class="field">
          <label for="te-due">Scadenza</label>
          <input id="te-due" v-model="form.due_date" class="input" type="date" />
        </div>
        <div class="field">
          <label for="te-assignee">Assegnato a</label>
          <select id="te-assignee" v-model="form.assignee_id" class="select">
            <option value="">Nessuno</option>
            <option v-for="u in catalog.users" :key="u.id" :value="u.id">{{ u.display_name || u.email }}</option>
          </select>
        </div>
      </template>
      <div class="field">
        <label for="te-notes">Note</label>
        <textarea id="te-notes" v-model="form.notes" class="textarea" />
      </div>
      <div class="actions">
        <button type="button" class="btn danger" @click="emit('remove')">Elimina</button>
        <span class="spacer" />
        <button type="button" class="btn" @click="emit('close')">Annulla</button>
        <button type="submit" class="btn primary">Salva</button>
      </div>
    </form>
  </div>
</template>
