<script setup lang="ts">
import { computed } from 'vue'
import { CheckboxIndicator, CheckboxRoot } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import type { Task } from '../types'
import { formatDue, initials, isOverdue } from '../format'
import { useCatalogStore } from '../stores/catalog'

const props = defineProps<{ task: Task; showDetails?: boolean; draggable?: boolean }>()
const emit = defineEmits<{ toggle: []; open: [] }>()
const { t } = useI18n()
const catalog = useCatalogStore()

const assignee = computed(() => catalog.userById(props.task.assignee_id))
const assigneeName = computed(() => assignee.value?.display_name || assignee.value?.email || '')
const due = computed(() => formatDue(props.task.due_date))
const overdue = computed(() => !props.task.is_completed && isOverdue(props.task.due_date))
</script>

<template>
  <div class="task" :class="{ done: task.is_completed }">
    <span v-if="draggable" class="task-grip" :aria-label="t('grocery.dragItem')" role="img">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <circle cx="9" cy="6" r="1.7" />
        <circle cx="15" cy="6" r="1.7" />
        <circle cx="9" cy="12" r="1.7" />
        <circle cx="15" cy="12" r="1.7" />
        <circle cx="9" cy="18" r="1.7" />
        <circle cx="15" cy="18" r="1.7" />
      </svg>
    </span>
    <button class="body" :aria-label="t('common.open', { title: task.title })" @click="emit('open')">
      <span class="title">{{ task.title }}</span>
      <span v-if="(showDetails && (due || assignee)) || task.notes" class="meta">
        <span v-if="showDetails && due" class="chip" :class="{ overdue }">{{ due }}</span>
        <span v-if="showDetails && assignee" class="avatar sm" :title="assigneeName">{{ initials(assigneeName) }}</span>
        <span v-if="task.notes" class="note">{{ task.notes }}</span>
      </span>
    </button>
    <CheckboxRoot
      class="check"
      :model-value="task.is_completed"
      :aria-label="
        task.is_completed ? t('common.markOpen', { title: task.title }) : t('common.markDone', { title: task.title })
      "
      @update:model-value="emit('toggle')"
    >
      <span class="box">
        <CheckboxIndicator>
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="3.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M5 13l4 4L19 7" />
          </svg>
        </CheckboxIndicator>
      </span>
    </CheckboxRoot>
  </div>
</template>
