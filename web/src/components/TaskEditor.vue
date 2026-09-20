<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import AppSelect from './AppSelect.vue'
import BottomSheet from './BottomSheet.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import type { Task, TaskFields } from '../types'
import { dateInDays, daysFromToday, fromDateInput, initials, toDateInput } from '../format'
import { NO_DEPARTMENT, useCatalogStore } from '../stores/catalog'

const props = defineProps<{ task: Task }>()
const emit = defineEmits<{ save: [patch: Partial<TaskFields>]; remove: []; close: [] }>()
const { t } = useI18n()
const catalog = useCatalogStore()

const form = reactive({
  list_id: props.task.list_id,
  title: props.task.title,
  notes: props.task.notes ?? '',
  department_id: props.task.department_id ?? NO_DEPARTMENT,
  assignee_id: props.task.assignee_id ?? '',
  due_date: props.task.due_date as string | null,
  item_type: props.task.item_type,
})
const confirmDelete = ref(false)

const listOptions = computed(() =>
  [...catalog.lists]
    .sort((a, b) => a.kind.localeCompare(b.kind) || a.sort_order - b.sort_order)
    .map((l) => ({
      value: l.id,
      label: `${l.name} · ${l.kind === 'grocery' ? t('lists.kindGrocery') : t('lists.kindTasks')}`,
    })),
)
/** The item type follows the chosen list. */
const itemType = computed(() => catalog.listById(form.list_id)?.kind ?? form.item_type)

const departmentOptions = computed(() => [
  { value: NO_DEPARTMENT, label: t('editor.noDepartment') },
  ...catalog.orderedDepartments.map((d) => ({ value: d.id, label: d.name })),
])

/** Which quick chip is active: none / today / tomorrow / custom date. */
const dueChip = computed(() => {
  const diff = daysFromToday(form.due_date)
  if (diff === null) return 'none'
  if (diff === 0) return 'today'
  if (diff === 1) return 'tomorrow'
  return 'custom'
})
function setDueChip(value: unknown) {
  if (value === 'none') form.due_date = null
  else if (value === 'today') form.due_date = dateInDays(0)
  else if (value === 'tomorrow') form.due_date = dateInDays(1)
}

function save() {
  const title = form.title.trim()
  if (!title) return
  const grocery = itemType.value === 'grocery'
  emit('save', {
    list_id: form.list_id,
    title,
    notes: form.notes.trim() || null,
    item_type: itemType.value,
    department_id: grocery && form.department_id !== NO_DEPARTMENT ? form.department_id : null,
    assignee_id: grocery ? null : form.assignee_id || null,
    due_date: grocery ? null : form.due_date,
  })
}
</script>

<template>
  <BottomSheet :open="true" :title="t('editor.title')" @close="emit('close')">
    <form style="display: flex; flex-direction: column; gap: 18px" @submit.prevent="save">
      <div class="field">
        <label for="te-title">{{ t('editor.itemTitle') }}</label>
        <input id="te-title" v-model="form.title" class="input" type="text" required autocomplete="off" />
      </div>

      <div class="field">
        <span class="label">{{ t('lists.list') }}</span>
        <AppSelect v-model="form.list_id" :label="t('lists.list')" :options="listOptions" />
      </div>

      <div v-if="itemType === 'grocery'" class="field">
        <span class="label">{{ t('editor.department') }}</span>
        <AppSelect v-model="form.department_id" :label="t('editor.department')" :options="departmentOptions" />
      </div>

      <template v-else>
        <div class="field">
          <span id="te-due-label" class="label">{{ t('editor.due') }}</span>
          <ToggleGroupRoot
            :model-value="dueChip"
            type="single"
            aria-labelledby="te-due-label"
            class="pill-row"
            style="flex-wrap: wrap; overflow: visible"
            @update:model-value="setDueChip"
          >
            <ToggleGroupItem value="none" class="pill">{{ t('editor.dueNone') }}</ToggleGroupItem>
            <ToggleGroupItem value="today" class="pill">{{ t('editor.dueToday') }}</ToggleGroupItem>
            <ToggleGroupItem value="tomorrow" class="pill">{{ t('editor.dueTomorrow') }}</ToggleGroupItem>
            <label class="pill" :class="{ on: dueChip === 'custom' }" style="position: relative">
              <span class="sr-only">{{ t('editor.pickDate') }}</span>
              {{ dueChip === 'custom' ? toDateInput(form.due_date) : t('editor.pickDate') }}
              <input
                type="date"
                :value="toDateInput(form.due_date)"
                style="position: absolute; inset: 0; opacity: 0; width: 100%; height: 100%"
                @input="form.due_date = fromDateInput(($event.target as HTMLInputElement).value)"
              />
            </label>
          </ToggleGroupRoot>
        </div>
        <div class="field">
          <span id="te-who-label" class="label">{{ t('editor.assignee') }}</span>
          <ToggleGroupRoot
            v-model="form.assignee_id"
            type="single"
            aria-labelledby="te-who-label"
            class="pill-row"
            style="flex-wrap: wrap; overflow: visible"
          >
            <ToggleGroupItem
              v-for="u in catalog.users"
              :key="u.id"
              :value="u.id"
              class="pill lg"
              style="padding-left: 6px"
            >
              <span class="avatar sm">{{ initials(u.display_name || u.email) }}</span
              >{{ u.display_name || u.email }}
            </ToggleGroupItem>
            <ToggleGroupItem value="" class="pill lg" :class="{ on: !form.assignee_id }">{{
              t('common.none')
            }}</ToggleGroupItem>
          </ToggleGroupRoot>
        </div>
      </template>

      <div class="field">
        <label for="te-notes">{{ t('editor.notes') }}</label>
        <textarea id="te-notes" v-model="form.notes" class="textarea" rows="3" />
      </div>

      <div class="actions">
        <button type="button" class="btn danger" @click="confirmDelete = true">{{ t('common.delete') }}</button>
        <button type="button" class="btn" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" class="btn primary">{{ t('common.save') }}</button>
      </div>
    </form>
  </BottomSheet>

  <ConfirmDialog
    :open="confirmDelete"
    :title="t('editor.deleteTitle', { title: task.title })"
    :description="t('editor.deleteText')"
    @confirm="emit('remove')"
    @cancel="confirmDelete = false"
  />
</template>
