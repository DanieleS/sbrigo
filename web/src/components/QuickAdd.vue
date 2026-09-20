<script setup lang="ts">
import { ref } from 'vue'
import { NO_DEPARTMENT, useCatalogStore } from '../stores/catalog'

const props = defineProps<{ placeholder: string; withDepartment?: boolean; defaultDepartment?: string }>()
const emit = defineEmits<{ add: [title: string, departmentId: string | null] }>()
const catalog = useCatalogStore()

const title = ref('')
const department = ref(props.defaultDepartment ?? NO_DEPARTMENT)

function submit() {
  const t = title.value.trim()
  if (!t) return
  emit('add', t, props.withDepartment && department.value !== NO_DEPARTMENT ? department.value : null)
  title.value = ''
}
</script>

<template>
  <div class="quickadd">
    <form @submit.prevent="submit">
      <input v-model="title" class="input" type="text" :placeholder="placeholder" autocomplete="off" enterkeyhint="done" />
      <select v-if="withDepartment" v-model="department" class="select" aria-label="Reparto">
        <option :value="NO_DEPARTMENT">Reparto…</option>
        <option v-for="d in catalog.orderedDepartments" :key="d.id" :value="d.id">{{ d.name }}</option>
      </select>
      <button class="btn primary icon" type="submit" aria-label="Aggiungi" :disabled="!title.trim()">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>
      </button>
    </form>
  </div>
</template>
