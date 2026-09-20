<script setup lang="ts">
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  VisuallyHidden,
} from 'reka-ui'

defineProps<{ open: boolean; title: string; description?: string }>()
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <DialogRoot :open="open" @update:open="(v) => !v && emit('close')">
    <DialogPortal>
      <DialogOverlay class="overlay" />
      <DialogContent class="sheet">
        <div class="grabber" aria-hidden="true" />
        <DialogTitle as="h2">{{ title }}</DialogTitle>
        <VisuallyHidden>
          <DialogDescription>{{ description ?? title }}</DialogDescription>
        </VisuallyHidden>
        <slot />
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
