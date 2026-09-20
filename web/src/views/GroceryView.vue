<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { dragAndDrop } from '@formkit/drag-and-drop/vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuTrigger,
  RadioGroupItem,
  RadioGroupRoot,
} from 'reka-ui'
import { useI18n } from 'vue-i18n'
import BottomSheet from '../components/BottomSheet.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import DepartmentGroup from '../components/DepartmentGroup.vue'
import ListSwitcher from '../components/ListSwitcher.vue'
import PageHeader from '../components/PageHeader.vue'
import QuickAdd from '../components/QuickAdd.vue'
import TaskEditor from '../components/TaskEditor.vue'
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
const supermarketSheet = ref(false)
const reordering = ref(false)
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

// Department groups can be dragged by their grip (reorder mode) to change the aisle order.
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
const supermarketLabel = computed(() => catalog.selectedSupermarket?.name ?? t('grocery.defaultOrder'))
function selectSupermarket(value: unknown) {
  if (typeof value !== 'string' || value === '') return
  void catalog.selectSupermarket(value === DEFAULT_ORDER ? '' : value)
  supermarketSheet.value = false
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
  <main class="page" style="--dock-h: 70px">
    <PageHeader :subtitle="t('grocery.summary', { open: openCount, done: doneCount })">
      <template #title>
        <ListSwitcher kind="grocery" />
      </template>
      <template #actions>
        <DropdownMenuRoot>
          <DropdownMenuTrigger class="btn ghost icon" :aria-label="t('grocery.options')">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <circle cx="5" cy="12" r="2" />
              <circle cx="12" cy="12" r="2" />
              <circle cx="19" cy="12" r="2" />
            </svg>
          </DropdownMenuTrigger>
          <DropdownMenuPortal>
            <DropdownMenuContent class="menu" align="end" :side-offset="6" :collision-padding="12">
              <DropdownMenuItem class="menu-item" @select="supermarketSheet = true">
                <span class="grow">{{ t('grocery.chooseSupermarket') }}</span>
                <span class="muted small">{{ supermarketLabel }}</span>
              </DropdownMenuItem>
              <DropdownMenuItem class="menu-item" @select="reordering = true">{{
                t('grocery.reorder')
              }}</DropdownMenuItem>
              <DropdownMenuItem class="menu-item danger" :disabled="doneCount === 0" @select="confirmClear = true">
                {{ t('grocery.clearCart') }}<span v-if="doneCount" class="muted small"> · {{ doneCount }}</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenuPortal>
        </DropdownMenuRoot>
      </template>
    </PageHeader>

    <div v-if="reordering" class="mode-bar">
      <span class="grow small">{{ t('grocery.reorderHint') }}</span>
      <button class="btn small dark" @click="reordering = false">{{ t('grocery.done') }}</button>
    </div>

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
        :reorderable="reordering"
        @toggle="tasks.toggle"
        @open="editing = $event"
        @moved="moveItem"
        @add="add"
      />
    </div>
    <DepartmentGroup
      v-if="noDepartmentGroup"
      :department="null"
      :items="noDepartmentGroup.items"
      :reorderable="reordering"
      @toggle="tasks.toggle"
      @open="editing = $event"
      @moved="moveItem"
      @add="add"
    />

    <div class="dock">
      <div class="dock-inner">
        <QuickAdd
          :placeholder="t('grocery.placeholder')"
          with-department
          :default-department="lastDepartment"
          @add="add"
        />
      </div>
    </div>

    <BottomSheet
      :open="supermarketSheet"
      :title="t('grocery.chooseSupermarket')"
      :description="t('grocery.supermarketHint')"
      @close="supermarketSheet = false"
    >
      <p class="muted small" style="margin: -8px 0 0">{{ t('grocery.supermarketHint') }}</p>
      <RadioGroupRoot
        :model-value="supermarketValue"
        class="card"
        :aria-label="t('grocery.supermarket')"
        @update:model-value="selectSupermarket"
      >
        <RadioGroupItem :value="DEFAULT_ORDER" class="radio-row">
          <span class="grow">{{ t('grocery.defaultOrder') }}</span
          ><span class="radio-dot" />
        </RadioGroupItem>
        <RadioGroupItem v-for="m in catalog.supermarkets" :key="m.id" :value="m.id" class="radio-row">
          <span class="grow">{{ m.name }}</span
          ><span class="radio-dot" />
        </RadioGroupItem>
      </RadioGroupRoot>
    </BottomSheet>

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
