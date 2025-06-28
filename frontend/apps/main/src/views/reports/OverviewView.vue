<template>
  <div class="overflow-y-auto">
    <div
      class="p-6 w-[calc(100%-3rem)]"
      :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    >
      <Spinner v-if="isLoading" />

      <div class="space-y-4">
        <div class="text-sm text-gray-500 text-left">
          {{ $t('globals.terms.lastUpdated') }}: {{ lastUpdateFormatted }}
        </div>

        <!-- First row -->
        <div class="mt-7 space-y-4">
          <!-- Cards for Open Conversations and Agent Status -->
          <div class="flex w-full space-x-4">
            <Card
              class="flex-1"
              title="Open conversations"
              :counts="cardCounts"
              :labels="conversationCountLabels"
            />
            <Card
              class="flex-1"
              title="Agent status"
              :counts="agentStatusCounts"
              :labels="agentStatusLabels"
            />
          </div>

          <!-- SLA Card with Date Filter -->
          <div class="w-full rounded box p-5">
            <div class="flex justify-between items-center mb-4">
              <p class="text-2xl font-medium">{{ slaCardTitle }}</p>
              <DateFilter @filter-change="handleSlaFilterChange" :label="''" />
            </div>
            <div class="grid grid-cols-2 md:grid-cols-5 gap-6">
              <div
                v-for="(item, key) in filteredSlaCounts"
                :key="key"
                class="flex flex-col items-center gap-2 text-center"
              >
                <span class="text-sm text-muted-foreground">{{ slaLabels[key] }}</span>
                <span class="text-2xl font-semibold">{{ item }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Line Chart with Date Filter -->
        <div class="rounded box w-full p-5">
          <div class="flex justify-between items-center mb-4">
            <p class="text-2xl font-medium">{{ $t('report.chart.title') }}</p>
            <DateFilter @filter-change="handleChartFilterChange" :label="''" />
          </div>
          <LineChart :data="processedLineData" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useEmitter } from '../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { handleHTTPError } from '../../utils/http'
import { formatDuration } from '../../utils/datetime'
import Card from '@/features/reports/OverviewCard.vue'
import LineChart from '@/features/reports/OverviewLineChart.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { DateFilter } from '@shared-ui/components/ui/date-filter'
import { useI18n } from 'vue-i18n'
import api from '../../api'

const emitter = useEmitter()
const { t } = useI18n()
const isLoading = ref(false)
const lastUpdate = ref(new Date())
const cardCounts = ref({})
const chartData = ref({ status_summary: [] })
let updateInterval = null

const agentStatusCounts = ref({
  agents_online: 0,
  agents_offline: 0,
  agents_away: 0,
  agents_reassigning: 0
})

const slaCounts = ref({
  first_response_met_count: 0,
  first_response_breached_count: 0,
  next_response_met_count: 0,
  next_response_breached_count: 0,
  resolution_met_count: 0,
  resolution_breached_count: 0,
  avg_first_response_time_sec: 0,
  avg_next_response_time_sec: 0,
  avg_resolution_time_sec: 0
})

// Date filter state
const slaDays = ref(30)
const chartDays = ref(90)

const formattedSlaCounts = computed(() => ({
  ...slaCounts.value,
  avg_first_response_time_sec: formatDuration(slaCounts.value.avg_first_response_time_sec, false),
  avg_next_response_time_sec: formatDuration(slaCounts.value.avg_next_response_time_sec, false),
  avg_resolution_time_sec: formatDuration(slaCounts.value.avg_resolution_time_sec, false)
}))

// Filter out counts that don't have a label.
const filteredSlaCounts = computed(() => {
  return Object.fromEntries(
    Object.entries(formattedSlaCounts.value).filter(([key]) => slaLabels.value[key])
  )
})

// Dynamic SLA card title based on selected days
const slaCardTitle = computed(() => t('report.sla.cardTitle', { days: slaDays.value }))

const lastUpdateFormatted = computed(() => lastUpdate.value.toLocaleTimeString())

const conversationCountLabels = computed(() => ({
  open: t('globals.terms.open'),
  awaiting_response: t('globals.terms.awaitingResponse'),
  unassigned: t('globals.terms.unassigned'),
  pending: t('globals.terms.pending')
}))

const agentStatusLabels = computed(() => ({
  agents_online: t('globals.terms.online'),
  agents_offline: t('globals.terms.offline'),
  agents_away: t('globals.terms.away'),
  agents_reassigning: t('globals.messages.reassigning')
}))

const slaLabels = computed(() => ({
  first_response_met_count: t('report.sla.firstRespMet'),
  first_response_breached_count: t('report.sla.firstRespBreached'),
  avg_first_response_time_sec: t('report.sla.avgFirstResp'),
  next_response_met_count: t('report.sla.nextRespMet'),
  next_response_breached_count: t('report.sla.nextRespBreached'),
  avg_next_response_time_sec: t('report.sla.avgNextResp'),
  resolution_met_count: t('report.sla.resolutionMet'),
  resolution_breached_count: t('report.sla.resolutionBreached'),
  avg_resolution_time_sec: t('report.sla.avgResolution')
}))

const processedLineData = computed(() => {
  const { new_conversations = [], resolved_conversations = [] } = chartData.value

  const dateMap = new Map()

  new_conversations.forEach((item) => {
    dateMap.set(item.date, {
      date: item.date,
      [t('report.chart.newConversations')]: item.count,
      [t('report.chart.resolvedConversations')]: 0
    })
  })

  resolved_conversations.forEach((item) => {
    const existing = dateMap.get(item.date)
    if (existing) {
      existing[t('report.chart.resolvedConversations')] = item.count
    } else {
      dateMap.set(item.date, {
        date: item.date,
        [t('report.chart.newConversations')]: 0,
        [t('report.chart.resolvedConversations')]: item.count
      })
    }
  })
  return Array.from(dateMap.values()).sort((a, b) => new Date(a.date) - new Date(b.date))
})

const showError = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

const fetchCardStats = async () => {
  try {
    const { data } = await api.getOverviewCounts()
    cardCounts.value = data.data
    agentStatusCounts.value = {
      agents_online: data.data.agents_online || 0,
      agents_offline: data.data.agents_offline || 0,
      agents_away: data.data.agents_away || 0,
      agents_reassigning: data.data.agents_reassigning || 0
    }
  } catch (error) {
    showError(error)
  }
}

const fetchSLAStats = async (days = slaDays.value) => {
  try {
    const { data } = await api.getOverviewSLA({ days })
    slaCounts.value = { ...slaCounts.value, ...data.data }
  } catch (error) {
    showError(error)
  }
}

const fetchChartData = async (days = chartDays.value) => {
  try {
    const { data } = await api.getOverviewCharts({ days })
    chartData.value = {
      new_conversations: data.data.new_conversations || [],
      resolved_conversations: data.data.resolved_conversations || [],
      messages_sent: data.data.messages_sent || []
    }
  } catch (error) {
    showError(error)
  }
}

// Date filter handlers
const handleSlaFilterChange = async (days) => {
  slaDays.value = days
  isLoading.value = true
  try {
    await fetchSLAStats(days)
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const handleChartFilterChange = async (days) => {
  chartDays.value = days
  isLoading.value = true
  try {
    await fetchChartData(days)
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const loadDashboardData = async () => {
  isLoading.value = true
  try {
    await Promise.allSettled([fetchCardStats(), fetchSLAStats(), fetchChartData()])
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const startRealtimeUpdates = () => {
  if (updateInterval) clearInterval(updateInterval)
  updateInterval = setInterval(loadDashboardData, 60000)
}

const stopRealtimeUpdates = () => {
  if (updateInterval) {
    clearInterval(updateInterval)
    updateInterval = null
  }
}

onMounted(() => {
  loadDashboardData()
  startRealtimeUpdates()
})

onUnmounted(() => {
  stopRealtimeUpdates()
})
</script>
