<template>
  <section class="rounded-md border p-4">
    <div class="flex items-center justify-between gap-3">
      <h2 class="font-medium">{{ title }}</h2>
      <span v-if="percent !== null && percent !== undefined" class="text-sm text-muted-foreground">
        {{ Math.round(percent) }}%
      </span>
    </div>
    <div class="mt-4 h-2 overflow-hidden rounded-full bg-muted">
      <div
        class="h-full rounded-full bg-primary transition-all"
        :style="{ width: `${Math.min(Math.max(percent || 0, 0), 100)}%` }"
      />
    </div>
    <div class="mt-3 flex justify-between gap-3 text-sm">
      <span>{{ formatBytes(used) }}</span>
      <span class="text-muted-foreground">{{
        limit === null || limit === undefined
          ? t('admin.systemResources.noLimit')
          : formatBytes(limit)
      }}</span>
    </div>
    <p v-if="detail" class="mt-2 text-xs text-muted-foreground">{{ detail }}</p>
  </section>
</template>

<script setup>
import { useI18n } from 'vue-i18n'

defineProps({
  title: { type: String, required: true },
  used: { type: Number, required: true },
  limit: { type: Number, default: null },
  percent: { type: Number, default: null },
  detail: { type: String, default: '' }
})

const { t } = useI18n()

const formatBytes = (value) => {
  if (value === null || value === undefined) return t('admin.systemResources.noLimit')
  if (value < 1024) return `${value} B`
  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let amount = value
  let unit = 'B'
  for (const nextUnit of units) {
    amount /= 1024
    unit = nextUnit
    if (amount < 1024 || nextUnit === units[units.length - 1]) break
  }
  return `${amount.toFixed(amount >= 10 ? 1 : 2)} ${unit}`
}
</script>
