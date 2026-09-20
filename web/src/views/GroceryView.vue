<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { dragAndDrop } from '@formkit/drag-and-drop/vue'
import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import ListCards from '../components/ListCards.vue'
import PageHeader from '../components/PageHeader.vue'
import QuickAdd from '../components/QuickAdd.vue'
import TaskEditor from '../components/TaskEditor.vue'
import DepartmentGroup from '../components/DepartmentGroup.vue'
import { useCatalogStore } from '../stores/catalog'
import { useTasksStore } from '../stores/tasks'
import { useUiStore } from '../stores/ui'
import type { Department, Task, TaskFields } from '../types'

const DEFAULT_ORDER = '__default__'
const { t } = useI18n()
const catalog = useCatalogStore()
const tasks = useTasksStore()
const ui = useUiStore()
const editing = ref<Task | null>(null)
const confirmClear = ref(false)
const lastDepartment = ref(readLastDepartment())

const currentList = computed(() => catalog.currentList('grocery'))
const items = computed(() => (currentList.value ? tasks.byList(currentList.value.id) : []))
const openCount = computed(() => items.value.filter((x) => !x.is_completed).length)
const doneCount = computed(() => items.value.length - openCount.value)

interface Group {
  department: Department | null
  items: Task[]
}

/** Departments in aisle order for the selected supermarket; empty departments are hidden. */
const groups = computed<Group[]>(() => {
  const byDep = new Map<string | null, Task[]>()
  for (const x of items.value) {
    const key = x.department_id && catalog.departmentById(x.department_id) ? x.department_id : null
    byDep.set(key, [...(byDep.get(key) ?? []), x])
  }
  const sortItems = (list: Task[]) =>
    list.sort(
      (a, b) =>
        Number(a.is_completed) - Number(b.is_completed) ||
        a.position - b.position ||
        a.created_at.localeCompare(b.created_at),
    )
  const out: Group[] = []
  for (const d of catalog.orderedDepartments) {
    const list = byDep.get(d.id)
    if (list) out.push({ department: d, items: sortItems(list) })
  }
  const none = byDep.get(null)
  if (none) out.push({ department: null, items: sortItems(none) })
  return out
})

// Department groups can be dragged by their grip to change the aisle order of the current context.
const groupsEl = ref<HTMLElement>()
const groupValues = ref<Group[]>([])
watch(groups, (g) => (groupValues.value = g.filter((x) => x.department)), { immediate: true })
const noDepartmentGroup = computed(() => groups.value.find((g) => !g.department))
onMounted(() => {
  dragAndDrop<Group>({
    parent: groupsEl,
    values: groupValues,
    group: 'grocery-groups',
    dragHandle: '.group-grip',
    draggingClass: 'dragging',
    synthDraggingClass: 'dragging',
    onSort({ values }) {
      const ids = values.map((g) => g.department?.id).filter((id): id is string => !!id)
      void ui.guard(() => catalog.reorderVisibleDepartments(ids))
    },
  })
})

async function moveItem(id: string, departmentId: string | null, position: number) {
  await tasks.patch(id, { department_id: departmentId, position })
}

const supermarketValue = computed(() => catalog.selectedSupermarketId || DEFAULT_ORDER)
function selectSupermarket(value: unknown) {
  if (typeof value !== 'string' || value === '') return // ignore deselect: one option is always active
  void catalog.selectSupermarket(value === DEFAULT_ORDER ? '' : value)
}

function readLastDepartment(): string | undefined {
  try {
    return localStorage.getItem('sbrigo.lastDepartment') ?? undefined
  } catch {
    return undefined
  }
}

async function add(title: string, departmentId: string | null) {
  try {
    if (departmentId) localStorage.setItem('sbrigo.lastDepartment', departmentId)
  } catch {
    // storage unavailable
  }
  lastDepartment.value = departmentId ?? undefined
  if (!currentList.value) return
  await tasks.add({ title, item_type: 'grocery', list_id: currentList.value.id, department_id: departmentId })
}

async function save(patch: Partial<TaskFields>) {
  if (editing.value) await tasks.patch(editing.value.id, patch)
  editing.value = null
}

async function remove() {
  if (editing.value) await tasks.removeTask(editing.value.id)
  editing.value = null
}

async function clearCart() {
  confirmClear.value = false
  if (currentList.value) await tasks.clearCompleted(currentList.value.id)
}
</script>

<template>
  <main class="page">
    <PageHeader :title="t('grocery.title')" :subtitle="t('grocery.summary', { open: openCount, done: doneCount })" />

    <ListCards
      kind="grocery"
      :selected-id="currentList?.id ?? ''"
      @select="(id) => catalog.selectList('grocery', id)"
    />

    <div v-if="items.length === 0" class="empty">
      <strong>{{ t('grocery.empty') }}</strong
      >{{ t('grocery.emptyHint') }}
    </div>

    <div ref="groupsEl" class="groups">
      <DepartmentGroup
        v-for="g in groupValues"
        :key="g.department?.id ?? 'none'"
        :department="g.department"
        :items="g.items"
        @toggle="tasks.toggle"
        @open="editing = $event"
        @moved="moveItem"
      />
    </div>
    <DepartmentGroup
      v-if="noDepartmentGroup"
      :department="null"
      :items="noDepartmentGroup.items"
      @toggle="tasks.toggle"
      @open="editing = $event"
      @moved="moveItem"
    />

    <div v-if="doneCount > 0" class="row" style="justify-content: center; margin-top: 4px">
      <button class="btn small" @click="confirmClear = true">{{ t('grocery.clearCart') }} · {{ doneCount }}</button>
    </div>

    <div class="dock">
      <div class="dock-inner">
        <ToggleGroupRoot
          :model-value="supermarketValue"
          type="single"
          class="pill-row"
          :aria-label="t('grocery.supermarket')"
          @update:model-value="selectSupermarket"
        >
          <ToggleGroupItem :value="DEFAULT_ORDER" class="pill">{{ t('grocery.defaultOrder') }}</ToggleGroupItem>
          <ToggleGroupItem v-for="m in catalog.supermarkets" :key="m.id" :value="m.id" class="pill">{{
            m.name
          }}</ToggleGroupItem>
        </ToggleGroupRoot>
        <QuickAdd
          :placeholder="t('grocery.placeholder')"
          with-department
          :default-department="lastDepartment"
          @add="add"
        />
      </div>
    </div>

    <TaskEditor v-if="editing" :task="editing" @save="save" @remove="remove" @close="editing = null" />
    <ConfirmDialog
      :open="confirmClear"
      :title="t('grocery.clearCartTitle', { n: doneCount })"
      :description="t('grocery.clearCartText')"
      @confirm="clearCart"
      @cancel="confirmClear = false"
    />
  </main>
</template>
