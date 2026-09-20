<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { dragAndDrop } from '@formkit/drag-and-drop/vue'
import { useI18n } from 'vue-i18n'
import TaskRow from './TaskRow.vue'
import { positionBetween } from '../ordering'
import { useTasksStore } from '../stores/tasks'
import type { Department, Task } from '../types'

const props = defineProps<{ department: Department | null; items: Task[] }>()
const emit = defineEmits<{
  toggle: [id: string]
  open: [task: Task]
  /** An item was dropped here: new department and manual position. */
  moved: [id: string, departmentId: string | null, position: number]
}>()
const { t } = useI18n()

const listEl = ref<HTMLElement>()
const values = ref<Task[]>([...props.items])
watch(
  () => props.items,
  (v) => {
    values.value = [...v]
  },
)

onMounted(() => {
  dragAndDrop<Task>({
    parent: listEl,
    values,
    group: 'grocery-items',
    dragHandle: '.task-grip',
    draggingClass: 'dragging',
    synthDraggingClass: 'dragging',
    dropZoneClass: 'drop-target',
    synthDropZoneClass: 'drop-target',
    dragPlaceholderClass: 'placeholder',
    onDragend({ draggedNode }) {
      // Read the final placement from the DOM: it is the same for sorts and transfers, whatever
      // parent the event is reported on.
      const el = draggedNode.el as HTMLElement
      const list = el.closest<HTMLElement>('.drop-list')
      if (!list) return
      const ids = Array.from(list.querySelectorAll<HTMLElement>('.task-wrap')).map((w) => w.dataset.id ?? '')
      const moved = draggedNode.data.value
      const index = ids.indexOf(moved.id)
      if (index < 0) return
      const targetDepartment = list.dataset.departmentId || null
      const tasks = useTasksStore()
      const prev = ids[index - 1] ? tasks.tasks[ids[index - 1]!]?.position : undefined
      const next = ids[index + 1] ? tasks.tasks[ids[index + 1]!]?.position : undefined
      const unchanged =
        moved.department_id === targetDepartment &&
        (prev === undefined || prev < moved.position) &&
        (next === undefined || moved.position < next)
      if (unchanged) return
      emit('moved', moved.id, targetDepartment, positionBetween(prev, next))
    },
  })
})
</script>

<template>
  <section class="group" :class="{ 'no-drag': !department }">
    <div class="group-head">
      <span v-if="department" class="group-grip" :aria-label="t('grocery.dragGroup')" role="img">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <circle cx="9" cy="6" r="1.6" />
          <circle cx="15" cy="6" r="1.6" />
          <circle cx="9" cy="12" r="1.6" />
          <circle cx="15" cy="12" r="1.6" />
          <circle cx="9" cy="18" r="1.6" />
          <circle cx="15" cy="18" r="1.6" />
        </svg>
      </span>
      <div class="grow">
        <h2>{{ department?.name ?? t('grocery.noDepartment') }}</h2>
        <p v-if="department?.description">{{ department.description }}</p>
      </div>
      <span class="count">{{ items.filter((x) => !x.is_completed).length }}</span>
    </div>
    <div ref="listEl" class="card drop-list" :data-department-id="department?.id ?? ''">
      <div v-for="x in values" :key="x.id" class="task-wrap" :data-id="x.id">
        <TaskRow :task="x" draggable @toggle="emit('toggle', x.id)" @open="emit('open', x)" />
      </div>
    </div>
  </section>
</template>
