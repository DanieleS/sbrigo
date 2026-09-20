<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../types'
import { formatDue, isOverdue } from '../format'
import { useCatalogStore } from '../stores/catalog'

const props = defineProps<{ task: Task; showDetails?: boolean }>()
const emit = defineEmits<{ toggle: []; open: [] }>()
const catalog = useCatalogStore()

const assignee = computed(() => catalog.userById(props.task.assignee_id))
const due = computed(() => formatDue(props.task.due_date))
const dueClass = computed(() => {
  if (props.task.is_completed || !props.task.due_date) return ''
  if (isOverdue(props.task.due_date)) return 'overdue'
  return due.value === 'Oggi' ? 'today' : ''
})
</script>

<template>
  <div class="task" :class="{ done: task.is_completed }">
    <button class="check" :aria-label="task.is_completed ? 'Segna da fare' : 'Segna fatto'" @click="emit('toggle')">
      <span>
        <svg
          v-if="task.is_completed"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="3.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M5 13l4 4L19 7" />
        </svg>
      </span>
    </button>
    <button class="body" @click="emit('open')">
      <span class="title">{{ task.title }}</span>
      <span v-if="showDetails && (due || assignee || task.notes)" class="meta">
        <span v-if="due" class="chip" :class="dueClass">{{ due }}</span>
        <span v-if="assignee" class="chip">{{ assignee.display_name || assignee.email }}</span>
        <span v-if="task.notes" class="muted">{{ task.notes }}</span>
      </span>
      <span v-else-if="task.notes" class="meta"
        ><span class="muted">{{ task.notes }}</span></span
      >
    </button>
  </div>
</template>
