<script setup lang="ts">
import { ToastDescription, ToastProvider, ToastRoot, ToastViewport } from 'reka-ui'
import { useUiStore } from '../stores/ui'

const ui = useUiStore()
</script>

<template>
  <ToastProvider :duration="3500" swipe-direction="down">
    <ToastRoot
      v-for="toast in ui.toasts"
      :key="toast.id"
      class="toast"
      :class="toast.tone"
      :open="true"
      @update:open="(v) => !v && ui.dismiss(toast.id)"
    >
      <ToastDescription>{{ toast.message }}</ToastDescription>
    </ToastRoot>
    <ToastViewport class="toasts" />
  </ToastProvider>
</template>
