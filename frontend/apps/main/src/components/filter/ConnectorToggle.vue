<template>
  <button
    type="button"
    class="inline-flex cursor-pointer items-center rounded-full border border-border bg-background px-3 py-0.5 text-xs font-medium uppercase tracking-wide text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
    :title="t('filter.toggleConnector')"
    @click.stop="toggle"
  >
    {{ modelValue === LOGIC.OR ? t('admin.automation.or') : t('admin.automation.and') }}
  </button>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { LOGIC } from '@/constants/filterConfig'

const modelValue = defineModel('modelValue', { required: true })
const { t } = useI18n()

const toggle = () => {
  modelValue.value = modelValue.value === LOGIC.OR ? LOGIC.AND : LOGIC.OR
}
</script>
