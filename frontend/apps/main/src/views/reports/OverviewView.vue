<template>
  <div class="overflow-y-auto">
    <div
      class="p-6 w-[calc(100%-3rem)]"
      :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    >
      <Spinner v-if="isLoading" />

      <div class="space-y-6">
        <div class="text-sm text-gray-500 text-left">
          {{ $t('globals.terms.lastUpdated') }}: {{ lastUpdateFormatted }}
        </div>

        <!-- Row 1: Open Conversations and Agent Status -->
        <div class="flex w-full space-x-4">
          <Card
            class="flex-1"
            :title="$t('report.openConversations')"
            :counts="cardCounts"
            :labels="conversationCountLabels"
            size="large"
          />
          <Card
            class="flex-1"
            :title="$t('report.agentStatus')"
            :counts="agentStatusCounts"
            :labels="agentStatusLabels"
            size="large"
          />
        </div>

        <!-- Row 2: CSAT and Message Volume -->
        <div class="flex w-full space-x-4">
          <!-- CSAT Card -->
          <div class="flex-1 box p-5">
            <div class="flex justify-between items-center mb-4">
              <p class="card-title">{{ $t('report.csat.cardTitle', { days: csatDays }) }}</p>
              <DateFilter @filter-change="handleCSATFilterChange" :label="''" />
            </div>
            <div class="grid grid-cols-3 gap-6">
              <div class="metric-item">
                <span class="metric-value">{{ formatRating(csatData.average_rating) }}</span>
                <span class="metric-label">{{ $t('report.csat.avgRating') }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-value">{{ formatPercent(csatData.response_rate) }}</span>
                <span class="metric-label">{{ $t('report.csat.responseRate') }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-value">{{
                  formatCompactNumber(csatData.total_responses || 0)
                }}</span>
                <span class="metric-label">{{ $t('report.csat.responses') }}</span>
              </div>
            </div>
          </div>

          <!-- Message Volume Card -->
          <div class="flex-1 box p-5">
            <div class="flex justify-between items-center mb-4">
              <p class="card-title">
                {{ $t('report.messages.cardTitle', { days: messageVolumeDays }) }}
              </p>
              <DateFilter @filter-change="handleMessageVolumeFilterChange" :label="''" />
            </div>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-6">
              <div class="metric-item">
                <span class="metric-value">{{
                  formatCompactNumber(messageVolumeData.total_messages || 0)
                }}</span>
                <span class="metric-label">{{ $t('report.messages.total') }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-value">{{
                  formatCompactNumber(messageVolumeData.incoming_messages || 0)
                }}</span>
                <span class="metric-label">{{ $t('report.messages.incoming') }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-value">{{
                  formatCompactNumber(messageVolumeData.outgoing_messages || 0)
                }}</span>
                <span class="metric-label">{{ $t('report.messages.outgoing') }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-value">{{
                  messageVolumeData.messages_per_conversation || 0
                }}</span>
                <span class="metric-label">{{ $t('report.messages.perConversation') }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Row 3: SLA Card with Compliance Percentages -->
        <div class="w-full rounded box p-5">
          <div class="flex justify-between items-center mb-6">
            <p class="card-title">{{ slaCardTitle }}</p>
            <DateFilter @filter-change="handleSlaFilterChange" :label="''" />
          </div>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
            <!-- First Response -->
            <div class="space-y-4">
              <p class="section-title">{{ $t('report.sla.firstResponse') }}</p>
              <div class="metric-item">
                <span class="metric-value text-green-600"
                  >{{ slaCounts.first_response_compliance_percent || 0 }}%</span
                >
                <span class="metric-label">{{ $t('report.sla.compliance') }}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-center pt-2">
                <div>
                  <span class="text-2xl font-semibold text-green-600">{{
                    slaCounts.first_response_met_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.met') }}</p>
                </div>
                <div>
                  <span class="text-2xl font-semibold text-red-600">{{
                    slaCounts.first_response_breached_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.breached') }}</p>
                </div>
              </div>
              <div class="text-center pt-2">
                <span class="text-lg font-medium">{{
                  formattedSlaCounts.avg_first_response_time_sec
                }}</span>
                <p class="text-xs text-muted-foreground">{{ $t('report.sla.avgFirstResp') }}</p>
              </div>
            </div>

            <!-- Next Response -->
            <div class="space-y-4 border-l border-r px-8">
              <p class="section-title">{{ $t('report.sla.nextResponse') }}</p>
              <div class="metric-item">
                <span class="metric-value text-green-600"
                  >{{ slaCounts.next_response_compliance_percent || 0 }}%</span
                >
                <span class="metric-label">{{ $t('report.sla.compliance') }}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-center pt-2">
                <div>
                  <span class="text-2xl font-semibold text-green-600">{{
                    slaCounts.next_response_met_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.met') }}</p>
                </div>
                <div>
                  <span class="text-2xl font-semibold text-red-600">{{
                    slaCounts.next_response_breached_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.breached') }}</p>
                </div>
              </div>
              <div class="text-center pt-2">
                <span class="text-lg font-medium">{{
                  formattedSlaCounts.avg_next_response_time_sec
                }}</span>
                <p class="text-xs text-muted-foreground">{{ $t('report.sla.avgNextResp') }}</p>
              </div>
            </div>

            <!-- Resolution -->
            <div class="space-y-4">
              <p class="section-title">{{ $t('report.sla.resolution') }}</p>
              <div class="metric-item">
                <span class="metric-value text-green-600"
                  >{{ slaCounts.resolution_compliance_percent || 0 }}%</span
                >
                <span class="metric-label">{{ $t('report.sla.compliance') }}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-center pt-2">
                <div>
                  <span class="text-2xl font-semibold text-green-600">{{
                    slaCounts.resolution_met_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.met') }}</p>
                </div>
                <div>
                  <span class="text-2xl font-semibold text-red-600">{{
                    slaCounts.resolution_breached_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.breached') }}</p>
                </div>
              </div>
              <div class="text-center pt-2">
                <span class="text-lg font-medium">{{
                  formattedSlaCounts.avg_resolution_time_sec
                }}</span>
                <p class="text-xs text-muted-foreground">{{ $t('report.sla.avgResolution') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Row 4: Tag Distribution -->
        <div class="w-full rounded box p-5">
          <div class="flex justify-between items-center mb-4">
            <p class="card-title">
              {{ $t('report.tags.cardTitle', { days: tagDistributionDays }) }}
            </p>
            <DateFilter @filter-change="handleTagDistributionFilterChange" :label="''" />
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <!-- Tagged percentage metric -->
            <div class="metric-item justify-center p-4">
              <span class="metric-value">{{ tagDistributionData.tagged_percentage || 0 }}%</span>
              <span class="metric-label mt-2">{{ $t('report.tags.tagged') }}</span>
              <span class="text-sm text-muted-foreground mt-1">
                {{ tagDistributionData.tagged_conversations || 0 }} /
                {{
                  (tagDistributionData.tagged_conversations || 0) +
                  (tagDistributionData.untagged_conversations || 0)
                }}
              </span>
            </div>

            <!-- Top tags list -->
            <div class="space-y-3">
              <p class="section-title mb-3 text-left">{{ $t('report.tags.topTags') }}</p>
              <div
                v-for="tag in (tagDistributionData.top_tags || []).slice(0, 5)"
                :key="tag.tag_id"
                class="flex justify-between items-center py-1"
              >
                <span class="text-sm">{{ tag.tag_name }}</span>
                <span class="text-sm font-semibold">{{ formatCompactNumber(tag.count) }}</span>
              </div>
              <p v-if="!tagDistributionData.top_tags?.length" class="text-sm text-muted-foreground">
                {{
                  $t('globals.messages.noResults', {
                    item: $t('globals.terms.tags').toLowerCase()
                  })
                }}
              </p>
            </div>
          </div>
        </div>

        <!-- Row 5: Line Chart -->
        <div class="rounded box w-full p-5">
          <div class="flex justify-between items-center mb-4">
            <p class="card-title">{{ $t('report.chart.title') }}</p>
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
  avg_resolution_time_sec: 0,
  first_response_compliance_percent: 0,
  next_response_compliance_percent: 0,
  resolution_compliance_percent: 0
})

// New data refs
const csatData = ref({
  average_rating: 0,
  response_rate: 0,
  total_responses: 0,
  total_sent: 0
})

const messageVolumeData = ref({
  total_messages: 0,
  incoming_messages: 0,
  outgoing_messages: 0,
  messages_per_conversation: 0
})

const tagDistributionData = ref({
  top_tags: [],
  tagged_conversations: 0,
  untagged_conversations: 0,
  tagged_percentage: 0
})

// Date filter state
const slaDays = ref(30)
const chartDays = ref(90)
const csatDays = ref(30)
const messageVolumeDays = ref(30)
const tagDistributionDays = ref(30)

// Format helpers
const formatRating = (value) => {
  if (!value) return '0.0'
  return Number(value).toFixed(1)
}

const formatPercent = (value) => {
  if (!value) return '0%'
  return `${Math.round(value)}%`
}

const formatCompactNumber = (value) => {
  if (!value || value < 1000) return value
  return new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 }).format(
    value
  )
}

const formattedSlaCounts = computed(() => ({
  ...slaCounts.value,
  avg_first_response_time_sec: formatDuration(slaCounts.value.avg_first_response_time_sec, false),
  avg_next_response_time_sec: formatDuration(slaCounts.value.avg_next_response_time_sec, false),
  avg_resolution_time_sec: formatDuration(slaCounts.value.avg_resolution_time_sec, false)
}))

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

const fetchCSATStats = async (days = csatDays.value) => {
  try {
    const { data } = await api.getOverviewCSAT({ days })
    csatData.value = { ...csatData.value, ...data.data }
  } catch (error) {
    showError(error)
  }
}

const fetchMessageVolumeStats = async (days = messageVolumeDays.value) => {
  try {
    const { data } = await api.getOverviewMessageVolume({ days })
    messageVolumeData.value = { ...messageVolumeData.value, ...data.data }
  } catch (error) {
    showError(error)
  }
}

const fetchTagDistributionStats = async (days = tagDistributionDays.value) => {
  try {
    const { data } = await api.getOverviewTagDistribution({ days })
    tagDistributionData.value = { ...tagDistributionData.value, ...data.data }
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

const handleCSATFilterChange = async (days) => {
  csatDays.value = days
  isLoading.value = true
  try {
    await fetchCSATStats(days)
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const handleMessageVolumeFilterChange = async (days) => {
  messageVolumeDays.value = days
  isLoading.value = true
  try {
    await fetchMessageVolumeStats(days)
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const handleTagDistributionFilterChange = async (days) => {
  tagDistributionDays.value = days
  isLoading.value = true
  try {
    await fetchTagDistributionStats(days)
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const loadDashboardData = async () => {
  isLoading.value = true
  try {
    await Promise.allSettled([
      fetchCardStats(),
      fetchSLAStats(),
      fetchChartData(),
      fetchCSATStats(),
      fetchMessageVolumeStats(),
      fetchTagDistributionStats()
    ])
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

<style scoped>
.metric-value {
  @apply text-3xl font-bold tracking-tight;
}

.metric-value {
  @apply text-3xl font-bold tracking-tight;
}

.metric-label {
  @apply text-xs text-muted-foreground uppercase tracking-wider;
}

.card-title {
  @apply text-xl font-medium;
}

.metric-item {
  @apply flex flex-col items-center gap-1 text-center;
}

.section-title {
  @apply text-sm font-medium text-center text-muted-foreground uppercase tracking-wider;
}
</style>
