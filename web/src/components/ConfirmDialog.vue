<script setup lang="ts">
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
} from 'reka-ui'
import { useI18n } from 'vue-i18n'

withDefaults(
  defineProps<{ open: boolean; title: string; description?: string; confirmLabel?: string; destructive?: boolean }>(),
  {
    description: '',
    confirmLabel: '',
    destructive: true,
  },
)
const emit = defineEmits<{ confirm: []; cancel: [] }>()
const { t } = useI18n()
</script>

<template>
  <AlertDialogRoot :open="open" @update:open="(v) => !v && emit('cancel')">
    <AlertDialogPortal>
      <AlertDialogOverlay class="overlay" />
      <AlertDialogContent class="alert">
        <AlertDialogTitle as="h2">{{ title }}</AlertDialogTitle>
        <AlertDialogDescription as="p">{{ description }}</AlertDialogDescription>
        <div class="actions">
          <AlertDialogCancel class="btn" @click="emit('cancel')">{{ t('common.cancel') }}</AlertDialogCancel>
          <AlertDialogAction class="btn" :class="destructive ? 'danger' : 'primary'" @click="emit('confirm')">
            {{ confirmLabel || (destructive ? t('common.delete') : t('common.confirm')) }}
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
