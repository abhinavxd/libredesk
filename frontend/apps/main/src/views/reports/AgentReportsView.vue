<template>
  <div class="p-4 md:p-6 space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold">{{ t('report.agents') }}</h1>
      <div class="flex gap-1">
        <Button
          v-for="option in [7, 30, 90]"
          :key="option"
          size="sm"
          :variant="days === option ? 'default' : 'outline'"
          @click="days = option"
        >
          {{ t('globals.messages.nDays', { days: option }) }}
        </Button>
      </div>
    </div>
    <table class="w-full text-sm">
      <thead>
        <tr class="text-left text-muted-foreground border-b">
          <th class="py-2">{{ t('globals.terms.agent') }}</th>
          <th class="py-2">{{ t('report.ticketsAssigned') }}</th>
          <th class="py-2">{{ t('report.ticketsResolved') }}</th>
          <th class="py-2">{{ t('report.replies') }}</th>
          <th class="py-2">{{ t('report.avgFirstReply') }}</th>
          <th class="py-2">{{ t('globals.terms.csatRating') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="row.id" class="border-b">
          <td class="py-2">{{ row.first_name }} {{ row.last_name }}</td>
          <td class="py-2">{{ row.tickets_assigned }}</td>
          <td class="py-2">{{ row.tickets_resolved }}</td>
          <td class="py-2">{{ row.replies }}</td>
          <td class="py-2">{{ formatSeconds(row.avg_first_reply_seconds) }}</td>
          <td class="py-2">{{ Number(row.csat_avg || 0).toFixed(1) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import api from '@/api'

const { t } = useI18n()
const days = ref(30)
const rows = ref([])

async function load() {
  const resp = await api.getAgentReports({ days: days.value })
  rows.value = resp.data.data || []
}

function formatSeconds(sec) {
  if (!sec) return '—'
  const m = Math.round(sec / 60)
  return `${m}m`
}

onMounted(load)
watch(days, load)
</script>
