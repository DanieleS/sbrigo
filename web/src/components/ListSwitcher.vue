<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BottomSheet from './BottomSheet.vue'
import ListCards from './ListCards.vue'
import { useCatalogStore } from '../stores/catalog'
import type { ItemType } from '../types'

const props = defineProps<{ kind: ItemType }>()
const { t } = useI18n()
const catalog = useCatalogStore()
const open = ref(false)

const current = computed(() => catalog.currentList(props.kind))

async function select(id: string) {
  await catalog.selectList(props.kind, id)
  open.value = false
}
</script>

<template>
  <button type="button" class="list-title" :aria-label="t('lists.switch')" @click="open = true">
    <h1>{{ current?.name ?? '…' }}</h1>
    <svg
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2.6"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="M6 9l6 6 6-6" />
    </svg>
  </button>

  <BottomSheet :open="open" :title="t('lists.title')" @close="open = false">
    <ListCards :kind="kind" :selected-id="current?.id ?? ''" stacked @select="select" />
  </BottomSheet>
</template>
