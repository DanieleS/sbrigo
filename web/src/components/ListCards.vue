<script setup lang="ts">
import { computed, ref } from 'vue'
import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui'
import { useI18n } from 'vue-i18n'
import BottomSheet from './BottomSheet.vue'
import { formatDue } from '../format'
import { useSummaries } from '../listSummary'
import { useCatalogStore } from '../stores/catalog'
import { useUiStore } from '../stores/ui'
import type { ItemType } from '../types'

const props = defineProps<{ kind: ItemType; selectedId: string; stacked?: boolean }>()
const emit = defineEmits<{ select: [id: string] }>()
const { t } = useI18n()
const catalog = useCatalogStore()
const ui = useUiStore()

const lists = computed(() => catalog.listsOfKind(props.kind))
const summaries = useSummaries(lists)

const creating = ref(false)
const newName = ref('')

function onSelect(value: unknown) {
  if (typeof value === 'string' && value) emit('select', value)
}

function lines(id: string): { main: string; sub: string; overdue: number } {
  const s = summaries.value[id]!
  if (props.kind === 'grocery') {
    const main = s.open > 0 ? t('lists.toGet', { n: s.open }) : s.done > 0 ? t('lists.allDone') : t('lists.emptyList')
    return { main, sub: s.done > 0 ? t('lists.inCart', { n: s.done }) : '', overdue: 0 }
  }
  const main = s.open > 0 ? t('lists.open', { n: s.open }) : s.done > 0 ? t('lists.allDone') : t('lists.emptyList')
  const sub = s.open > 0 ? (s.nextDue ? t('lists.next', { when: formatDue(s.nextDue) }) : t('lists.noNext')) : ''
  return { main, sub, overdue: s.overdue }
}

async function create() {
  const name = newName.value.trim()
  if (!name) return
  const ok = await ui.guard(async () => {
    const created = await catalog.createList(name, props.kind)
    emit('select', created.id)
  })
  if (ok) {
    creating.value = false
    newName.value = ''
  }
}
</script>

<template>
  <ToggleGroupRoot
    :model-value="selectedId"
    type="single"
    class="list-cards"
    :class="{ stacked }"
    :aria-label="t('lists.title')"
    @update:model-value="onSelect"
  >
    <ToggleGroupItem v-for="l in lists" :key="l.id" :value="l.id" class="list-card">
      <span class="list-card-name">{{ l.name }}</span>
      <span class="list-card-main">{{ lines(l.id).main }}</span>
      <span class="list-card-sub">
        <span v-if="lines(l.id).overdue" class="chip overdue">{{
          t('lists.overdue', { n: lines(l.id).overdue })
        }}</span>
        <span v-else-if="lines(l.id).sub">{{ lines(l.id).sub }}</span>
      </span>
    </ToggleGroupItem>
    <button type="button" class="list-card add" :aria-label="t('lists.newList')" @click="creating = true">
      <svg
        width="22"
        height="22"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.4"
        stroke-linecap="round"
        aria-hidden="true"
      >
        <path d="M12 5v14M5 12h14" />
      </svg>
    </button>
  </ToggleGroupRoot>

  <BottomSheet
    :open="creating"
    :title="kind === 'grocery' ? t('lists.newGroceryList') : t('lists.newTaskList')"
    @close="creating = false"
  >
    <form style="display: flex; flex-direction: column; gap: 16px" @submit.prevent="create">
      <div class="field">
        <label for="new-list-name">{{ t('lists.name') }}</label>
        <input id="new-list-name" v-model="newName" class="input" required autocomplete="off" />
      </div>
      <div class="actions">
        <button type="button" class="btn" @click="creating = false">{{ t('common.cancel') }}</button>
        <button type="submit" class="btn primary" :disabled="!newName.trim()">{{ t('common.add') }}</button>
      </div>
    </form>
  </BottomSheet>
</template>
