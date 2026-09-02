<template>
  <div class="flex items-center gap-2 text-xs text-muted-foreground">
    <span class="h-px w-8 bg-border" aria-hidden="true"></span>
    <button
      type="button"
      :class="[
        'inline-flex cursor-pointer items-center rounded-md border border-border bg-background font-semibold uppercase tracking-wide text-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        props.variant === 'group' ? 'min-h-8 px-3 text-sm' : 'min-h-7 px-2.5 text-xs'
      ]"
      :title="t('filter.toggleConnector')"
      @click.stop="toggle"
    >
      {{ connectorLabel }}
    </button>
    <span class="h-px w-8 bg-border" aria-hidden="true"></span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { LOGIC } from '@/constants/filterConfig'

const modelValue = defineModel('modelValue', { required: true })
const props = defineProps({
  mode: { type: String, default: 'and-or' },
  variant: { type: String, default: 'condition' }
})
const { t } = useI18n()

const connectorLabel = computed(() => {
  if (props.mode === 'all-any') {
    return modelValue.value === LOGIC.OR ? t('admin.automation.any') : t('admin.automation.all')
  }
  return modelValue.value === LOGIC.OR ? t('admin.automation.or') : t('admin.automation.and')
})

const toggle = () => {
  modelValue.value = modelValue.value === LOGIC.OR ? LOGIC.AND : LOGIC.OR
}
</script>
