<script setup lang="ts">
import { computed } from 'vue'
import {
  SelectContent,
  SelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectPortal,
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectViewport,
} from 'reka-ui'

export interface SelectOption {
  value: string
  label: string
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    options: SelectOption[]
    label: string
    placeholder?: string
    chip?: boolean
    side?: 'top' | 'bottom'
  }>(),
  { placeholder: '', chip: false, side: 'bottom' },
)
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const selectedLabel = computed(() => props.options.find((o) => o.value === props.modelValue)?.label ?? '')
</script>

<template>
  <SelectRoot :model-value="modelValue" @update:model-value="(v) => emit('update:modelValue', String(v ?? ''))">
    <SelectTrigger class="select-trigger" :class="{ 'chip-like': chip }" :aria-label="label">
      <SelectValue :placeholder="placeholder">{{ selectedLabel || placeholder }}</SelectValue>
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.4"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M6 9l6 6 6-6" />
      </svg>
    </SelectTrigger>
    <SelectPortal>
      <SelectContent class="select-content" position="popper" :side="side" :side-offset="6" :collision-padding="12">
        <SelectViewport>
          <SelectItem v-for="o in options" :key="o.value" :value="o.value" class="select-item">
            <span class="indicator">
              <SelectItemIndicator>
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"
                >
                  <path d="M5 13l4 4L19 7" />
                </svg>
              </SelectItemIndicator>
            </span>
            <SelectItemText>{{ o.label }}</SelectItemText>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
