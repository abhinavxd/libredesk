<template>
  <div class="flex items-center gap-3">
    <div class="h-px flex-1 bg-border"></div>
    <span class="text-xs text-muted-foreground">{{ label }}</span>
    <div class="h-px flex-1 bg-border"></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { isToday, isYesterday, format } from 'date-fns'

const props = defineProps({
  date: {
    type: [String, Date],
    required: true
  }
})

const i18n = useI18n()

const label = computed(() => {
  const date = new Date(props.date)
  if (isToday(date)) return i18n._('Today')
  if (isYesterday(date)) return i18n._('Yesterday')
  return format(date, 'd MMM yyyy')
})
</script>
