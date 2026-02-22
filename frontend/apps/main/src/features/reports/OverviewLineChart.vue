<template>
  <LineChart
    :data="data"
    index="date"
    :categories="[t('report.chart.newConversations'), t('report.chart.resolvedConversations')]"
    :x-formatter="xFormatter"
    :y-formatter="yFormatter"
  />
</template>

<script setup>
import { LineChart } from '@shared-ui/components/ui/chart-line'
import { useI18n } from 'vue-i18n'
const props = defineProps({
  data: {
    type: Array,
    default: () => []
  }
})
const { t } = useI18n()

const xFormatter = (tick) => {
  return props.data[tick]?.date ?? ''
}

const yFormatter = (tick) => {
  return Number.isInteger(tick) ? tick : ''
}
</script>
