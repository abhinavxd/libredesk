<template>
  <LineChart
    :data="data"
    index="date"
    :categories="[i18n._('New conversations'), i18n._('Resolved conversations')]"
    :x-formatter="xFormatter"
    :y-formatter="yFormatter"
  />
</template>

<script setup>
import { LineChart } from '@shared-ui/components/ui/chart-line'
import { useI18n } from '@shared-ui/i18n'
const props = defineProps({
  data: {
    type: Array,
    default: () => []
  }
})
const i18n = useI18n()

const xFormatter = (tick) => {
  return props.data[tick]?.date ?? ''
}

const yFormatter = (tick) => {
  return Number.isInteger(tick) ? tick : ''
}
</script>
