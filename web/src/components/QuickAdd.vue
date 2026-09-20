<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppSelect from './AppSelect.vue'
import { NO_DEPARTMENT, useCatalogStore } from '../stores/catalog'

const props = defineProps<{ placeholder: string; withDepartment?: boolean; defaultDepartment?: string }>()
const emit = defineEmits<{ add: [title: string, departmentId: string | null] }>()
const { t } = useI18n()
const catalog = useCatalogStore()

const title = ref('')
const department = ref(props.defaultDepartment ?? NO_DEPARTMENT)

const options = computed(() => [
  { value: NO_DEPARTMENT, label: t('grocery.noDepartment') },
  ...catalog.orderedDepartments.map((d) => ({ value: d.id, label: d.name })),
])

function submit() {
  const v = title.value.trim()
  if (!v) return
  emit('add', v, props.withDepartment && department.value !== NO_DEPARTMENT ? department.value : null)
  title.value = ''
}
</script>

<template>
  <form class="quickadd" @submit.prevent="submit">
    <div class="field-wrap">
      <label class="sr-only" for="quickadd-input">{{ placeholder }}</label>
      <input
        id="quickadd-input"
        v-model="title"
        type="text"
        :placeholder="placeholder"
        autocomplete="off"
        enterkeyhint="done"
      />
      <AppSelect
        v-if="withDepartment"
        v-model="department"
        chip
        side="top"
        :label="t('grocery.department')"
        :options="options"
        :placeholder="t('grocery.department')"
      />
    </div>
    <button class="btn primary" type="submit" :aria-label="t('common.add')" :disabled="!title.trim()">
      <svg
        width="22"
        height="22"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.6"
        stroke-linecap="round"
        aria-hidden="true"
      >
        <path d="M12 5v14M5 12h14" />
      </svg>
    </button>
  </form>
</template>
