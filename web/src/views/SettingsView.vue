<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppSelect from '../components/AppSelect.vue'
import BottomSheet from '../components/BottomSheet.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import { LOCALES, LOCALE_NAMES, setLocale, type Locale } from '../i18n'
import { useAuthStore } from '../stores/auth'
import { useCatalogStore } from '../stores/catalog'
import { useTasksStore } from '../stores/tasks'
import { useUiStore, type Theme } from '../stores/ui'
import type { Department } from '../types'

const { t, locale } = useI18n()
const auth = useAuthStore()
const catalog = useCatalogStore()
const tasks = useTasksStore()
const ui = useUiStore()

// Preferences
const localeOptions = LOCALES.map((l) => ({ value: l, label: LOCALE_NAMES[l] }))
const themeOptions = computed(() => [
  { value: 'system', label: t('settings.themeSystem') },
  { value: 'light', label: t('settings.themeLight') },
  { value: 'dark', label: t('settings.themeDark') },
])

// Supermarkets
const newSupermarket = ref('')
const orderingId = ref<string | null>(null)
const draftOrder = ref<string[]>([])
const renaming = ref<{ id: string; name: string } | null>(null)
const deletingSupermarket = ref<{ id: string; name: string } | null>(null)

function startOrdering(id: string) {
  orderingId.value = id
  const explicit = (catalog.orders[id] ?? []).filter((depId) => catalog.departmentById(depId))
  const rest = catalog.departments
    .filter((d) => !explicit.includes(d.id))
    .sort((a, b) => a.default_sort_order - b.default_sort_order)
    .map((d) => d.id)
  draftOrder.value = [...explicit, ...rest]
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
  if (await ui.guard(() => catalog.setDepartmentOrder(id, draftOrder.value))) orderingId.value = null
}
async function addSupermarket() {
  const name = newSupermarket.value.trim()
  if (!name) return
  if (await ui.guard(() => catalog.createSupermarket(name))) newSupermarket.value = ''
}
async function saveRename() {
  const r = renaming.value
  if (!r || !r.name.trim()) return
  if (await ui.guard(() => catalog.renameSupermarket(r.id, r.name.trim()))) renaming.value = null
}
async function deleteSupermarket() {
  const s = deletingSupermarket.value
  if (!s) return
  await ui.guard(() => catalog.deleteSupermarket(s.id))
  deletingSupermarket.value = null
}

// Departments
const editingDep = ref<Department | null>(null)
const deletingDep = ref<Department | null>(null)
const depForm = reactive({ name: '', description: '', default_sort_order: 0 })
const sortedDepartments = computed(() =>
  [...catalog.departments].sort((a, b) => a.default_sort_order - b.default_sort_order || a.name.localeCompare(b.name)),
)
function editDepartment(d: Department | null) {
  editingDep.value = d ?? {
    id: '',
    name: '',
    description: '',
    default_sort_order: (sortedDepartments.value.at(-1)?.default_sort_order ?? 0) + 10,
  }
  Object.assign(depForm, {
    name: editingDep.value.name,
    description: editingDep.value.description,
    default_sort_order: editingDep.value.default_sort_order,
  })
}
async function saveDepartment() {
  const d = editingDep.value
  if (!d || !depForm.name.trim()) return
  const payload = {
    name: depForm.name.trim(),
    description: depForm.description.trim(),
    default_sort_order: Number(depForm.default_sort_order) || 0,
  }
  const ok = await ui.guard(() =>
    d.id ? catalog.updateDepartment({ id: d.id, ...payload }) : catalog.createDepartment(payload),
  )
  if (ok) editingDep.value = null
}
async function deleteDepartment() {
  const d = deletingDep.value
  if (!d) return
  await ui.guard(() => catalog.deleteDepartment(d.id))
  deletingDep.value = null
  editingDep.value = null
}

async function reload() {
  await ui.guard(async () => {
    await catalog.refresh()
    await tasks.refreshFromServer()
  })
}
</script>

<template>
  <main class="page" style="--dock-h: 0px">
    <header class="page-head">
      <h1>{{ t('settings.title') }}</h1>
    </header>

    <section class="section">
      <h2>{{ t('settings.account') }}</h2>
      <div class="card">
        <div class="list-item">
          <div class="grow">
            <div class="primary-text">{{ auth.user?.display_name || auth.user?.email || t('settings.user') }}</div>
            <div class="small muted">{{ auth.user?.email }}</div>
          </div>
          <button class="btn small" @click="auth.logout()">{{ t('common.logout') }}</button>
        </div>
        <div class="list-item">
          <button class="btn small ghost grow" style="justify-content: flex-start" @click="auth.logoutAll()">
            {{ t('common.logoutAll') }}
          </button>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>{{ t('settings.preferences') }}</h2>
      <div class="card">
        <div class="list-item">
          <span class="grow primary-text">{{ t('settings.language') }}</span>
          <div style="width: 160px">
            <AppSelect
              :model-value="locale"
              :label="t('settings.language')"
              :options="localeOptions"
              @update:model-value="(v) => setLocale(v as Locale)"
            />
          </div>
        </div>
        <div class="list-item">
          <span class="grow primary-text">{{ t('settings.theme') }}</span>
          <div style="width: 160px">
            <AppSelect
              :model-value="ui.theme"
              :label="t('settings.theme')"
              :options="themeOptions"
              @update:model-value="(v) => ui.setTheme(v as Theme)"
            />
          </div>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>{{ t('settings.supermarkets') }}</h2>
      <p class="hint">{{ t('settings.supermarketsHint') }}</p>
      <div class="card">
        <template v-for="m in catalog.supermarkets" :key="m.id">
          <div class="list-item">
            <span class="grow primary-text">{{ m.name }}</span>
            <button
              class="btn small"
              :class="{ dark: orderingId === m.id }"
              @click="orderingId === m.id ? (orderingId = null) : startOrdering(m.id)"
            >
              {{ t('settings.aisles') }}
            </button>
            <button class="btn small" @click="renaming = { id: m.id, name: m.name }">{{ t('common.rename') }}</button>
            <button
              class="btn small danger"
              :aria-label="`${t('common.delete')} ${m.name}`"
              @click="deletingSupermarket = { id: m.id, name: m.name }"
            >
              {{ t('common.delete') }}
            </button>
          </div>
          <div v-if="orderingId === m.id" class="sub-panel">
            <div v-for="(depId, i) in draftOrder" :key="depId" class="order-row">
              <span class="num">{{ i + 1 }}.</span>
              <span class="grow">{{ catalog.departmentById(depId)?.name }}</span>
              <button
                class="btn small icon"
                :aria-label="t('settings.moveUp')"
                :disabled="i === 0"
                @click="move(i, -1)"
              >
                ▲
              </button>
              <button
                class="btn small icon"
                :aria-label="t('settings.moveDown')"
                :disabled="i === draftOrder.length - 1"
                @click="move(i, 1)"
              >
                ▼
              </button>
            </div>
            <div class="row" style="justify-content: flex-end; margin-top: 8px">
              <button class="btn small" @click="orderingId = null">{{ t('common.cancel') }}</button>
              <button class="btn small primary" @click="saveOrder">{{ t('settings.saveOrder') }}</button>
            </div>
          </div>
        </template>
        <form class="list-item" @submit.prevent="addSupermarket">
          <label class="sr-only" for="new-supermarket">{{ t('settings.newSupermarket') }}</label>
          <input
            id="new-supermarket"
            v-model="newSupermarket"
            class="input"
            :placeholder="t('settings.newSupermarket')"
          />
          <button class="btn primary" type="submit" :disabled="!newSupermarket.trim()">{{ t('common.add') }}</button>
        </form>
      </div>
    </section>

    <section class="section">
      <h2>{{ t('settings.departments') }}</h2>
      <p class="hint">{{ t('settings.departmentsHint') }}</p>
      <div class="card">
        <div v-for="d in sortedDepartments" :key="d.id" class="list-item">
          <div class="grow">
            <div class="primary-text">{{ d.name }}</div>
            <div class="small muted">{{ d.description }}</div>
          </div>
          <button class="btn small" :aria-label="`${t('common.edit')} ${d.name}`" @click="editDepartment(d)">
            {{ t('common.edit') }}
          </button>
        </div>
        <div class="list-item">
          <button class="btn block" @click="editDepartment(null)">{{ t('settings.newDepartment') }}</button>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>{{ t('settings.data') }}</h2>
      <div class="card">
        <div class="list-item">
          <div class="grow small muted">
            {{ t('settings.localItems', { n: tasks.all.length })
            }}<span v-if="tasks.pending"> · {{ t('status.pending', { n: tasks.pending }) }}</span>
          </div>
          <button class="btn small" @click="reload">{{ t('settings.reload') }}</button>
        </div>
      </div>
    </section>

    <BottomSheet
      :open="!!editingDep"
      :title="editingDep?.id ? t('settings.editDepartment') : t('settings.newDepartment')"
      @close="editingDep = null"
    >
      <form v-if="editingDep" style="display: flex; flex-direction: column; gap: 16px" @submit.prevent="saveDepartment">
        <div class="field">
          <label for="dep-name">{{ t('settings.name') }}</label>
          <input id="dep-name" v-model="depForm.name" class="input" required />
        </div>
        <div class="field">
          <label for="dep-desc">{{ t('settings.legend') }}</label>
          <textarea id="dep-desc" v-model="depForm.description" class="textarea" rows="2" />
        </div>
        <div class="field">
          <label for="dep-order">{{ t('settings.defaultOrder') }}</label>
          <input id="dep-order" v-model.number="depForm.default_sort_order" class="input" type="number" />
        </div>
        <div class="actions">
          <button v-if="editingDep.id" type="button" class="btn danger" @click="deletingDep = editingDep">
            {{ t('common.delete') }}
          </button>
          <button type="button" class="btn" @click="editingDep = null">{{ t('common.cancel') }}</button>
          <button type="submit" class="btn primary">{{ t('common.save') }}</button>
        </div>
      </form>
    </BottomSheet>

    <BottomSheet :open="!!renaming" :title="t('common.rename')" @close="renaming = null">
      <form v-if="renaming" style="display: flex; flex-direction: column; gap: 16px" @submit.prevent="saveRename">
        <div class="field">
          <label for="rename-input">{{ t('settings.name') }}</label>
          <input id="rename-input" v-model="renaming.name" class="input" required />
        </div>
        <div class="actions">
          <button type="button" class="btn" @click="renaming = null">{{ t('common.cancel') }}</button>
          <button type="submit" class="btn primary">{{ t('common.save') }}</button>
        </div>
      </form>
    </BottomSheet>

    <ConfirmDialog
      :open="!!deletingSupermarket"
      :title="t('settings.deleteSupermarketTitle', { name: deletingSupermarket?.name ?? '' })"
      @confirm="deleteSupermarket"
      @cancel="deletingSupermarket = null"
    />
    <ConfirmDialog
      :open="!!deletingDep"
      :title="t('settings.deleteDepartmentTitle', { name: deletingDep?.name ?? '' })"
      :description="t('settings.deleteDepartmentText')"
      @confirm="deleteDepartment"
      @cancel="deletingDep = null"
    />
  </main>
</template>
