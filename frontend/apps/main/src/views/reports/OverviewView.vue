<template>
  <div class="overflow-y-auto">
    <div
      class="p-6 w-[calc(100%-3rem)]"
      :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    >
      <Spinner v-if="isLoading" />

      <div class="space-y-6">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="text-sm text-muted-foreground">
              {{ $t('globals.terms.lastUpdated') }}: {{ lastUpdateFormatted }}
            </span>
            <template v-if="autoRefreshPaused">
              <Separator orientation="vertical" class="h-4" />
              <span class="text-sm text-muted-foreground">{{ $t('report.autoRefreshPaused') }}</span>
              <Button size="sm" variant="outline" :disabled="isLoading" @click="manualRefresh">
                {{ $t('globals.terms.refresh') }}
              </Button>
            </template>
          </div>
          <div class="flex items-center gap-1">
            <Button
              v-for="option in rangeOptions"
              :key="option"
              size="sm"
              :variant="!isCustom && range === option ? 'default' : 'outline'"
              :disabled="isLoading"
              @click="selectRange(option)"
            >
              {{ $t('globals.messages.nDays', { days: option }) }}
            </Button>
            <Separator orientation="vertical" class="h-6 mx-1" />
            <Button
              size="sm"
              :variant="isCustom ? 'default' : 'outline'"
              :disabled="isLoading"
              @click="enableCustom"
            >
              {{ $t('globals.terms.custom') }}
            </Button>
            <Input
              v-if="isCustom"
              v-model="customDays"
              type="number"
              min="1"
              max="365"
              class="w-20 h-8"
              :disabled="isLoading"
              @blur="applyCustom"
              @keyup.enter="applyCustom"
            />
          </div>
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
            <p class="card-title mb-4">{{ $t('report.csat.cardTitle') }}</p>
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
            <p class="card-title mb-4">{{ $t('report.messages.cardTitle') }}</p>
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
          <p class="card-title mb-4">{{ $t('report.sla.cardTitle') }}</p>

          <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
            <!-- First Response -->
            <div class="space-y-4">
              <p class="section-title">{{ $t('report.sla.firstResponse') }}</p>
              <div class="metric-item">
                <span class="metric-value text-primary"
                  >{{ slaCounts.first_response_compliance_percent || 0 }}%</span
                >
                <span class="metric-label">{{ $t('report.sla.compliance') }}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-center pt-2">
                <div>
                  <span class="text-2xl font-semibold tabular-nums text-primary">{{
                    slaCounts.first_response_met_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.met') }}</p>
                </div>
                <div>
                  <span class="text-2xl font-semibold tabular-nums text-destructive">{{
                    slaCounts.first_response_breached_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.breached') }}</p>
                </div>
              </div>
              <div class="text-center pt-2">
                <span class="text-lg font-medium tabular-nums">{{
                  formattedSlaCounts.avg_first_response_time_sec
                }}</span>
                <p class="metric-label">{{ $t('report.sla.avgFirstResp') }}</p>
              </div>
            </div>

            <!-- Next Response -->
            <div class="space-y-4 border-l border-r px-8">
              <p class="section-title">{{ $t('report.sla.nextResponse') }}</p>
              <div class="metric-item">
                <span class="metric-value text-primary"
                  >{{ slaCounts.next_response_compliance_percent || 0 }}%</span
                >
                <span class="metric-label">{{ $t('report.sla.compliance') }}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-center pt-2">
                <div>
                  <span class="text-2xl font-semibold tabular-nums text-primary">{{
                    slaCounts.next_response_met_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.met') }}</p>
                </div>
                <div>
                  <span class="text-2xl font-semibold tabular-nums text-destructive">{{
                    slaCounts.next_response_breached_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.breached') }}</p>
                </div>
              </div>
              <div class="text-center pt-2">
                <span class="text-lg font-medium tabular-nums">{{
                  formattedSlaCounts.avg_next_response_time_sec
                }}</span>
                <p class="metric-label">{{ $t('report.sla.avgNextResp') }}</p>
              </div>
            </div>

            <!-- Resolution -->
            <div class="space-y-4">
              <p class="section-title">{{ $t('report.sla.resolution') }}</p>
              <div class="metric-item">
                <span class="metric-value text-primary"
                  >{{ slaCounts.resolution_compliance_percent || 0 }}%</span
                >
                <span class="metric-label">{{ $t('report.sla.compliance') }}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-center pt-2">
                <div>
                  <span class="text-2xl font-semibold tabular-nums text-primary">{{
                    slaCounts.resolution_met_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.met') }}</p>
                </div>
                <div>
                  <span class="text-2xl font-semibold tabular-nums text-destructive">{{
                    slaCounts.resolution_breached_count || 0
                  }}</span>
                  <p class="metric-label">{{ $t('report.sla.breached') }}</p>
                </div>
              </div>
              <div class="text-center pt-2">
                <span class="text-lg font-medium tabular-nums">{{
                  formattedSlaCounts.avg_resolution_time_sec
                }}</span>
                <p class="metric-label">{{ $t('report.sla.avgResolution') }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Row 4: Tag Distribution -->
        <div class="w-full rounded box p-5">
          <p class="card-title mb-4">{{ $t('report.tags.cardTitle') }}</p>

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
                {{ $t('report.noTagsFound') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Row 5: Line Chart -->
        <div class="rounded box w-full p-5">
          <p class="card-title mb-4">{{ $t('report.chart.title') }}</p>
          <LineChart :data="processedLineData" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useIntervalFn, useDocumentVisibility } from '@vueuse/core'
import { useEmitter } from '../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { formatDuration } from '@shared-ui/utils/datetime.js'
import Card from '@/features/reports/OverviewCard.vue'
import LineChart from '@/features/reports/OverviewLineChart.vue'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input'
import { Separator } from '@shared-ui/components/ui/separator'
import { useI18n } from 'vue-i18n'
import api from '../../api'

const emitter = useEmitter()
const { t } = useI18n()
const isLoading = ref(false)
const lastUpdate = ref(new Date())
const rangeOptions = [7, 30, 90]
const range = ref(7)
const isCustom = ref(false)
const customDays = ref('')
const REFRESH_INTERVAL_MS = 5 * 60 * 1000
const MAX_POLL_DURATION_MS = 60 * 60 * 1000
const autoRefreshPaused = ref(false)
const cardCounts = ref({})
const chartData = ref({ status_summary: [] })

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

const sections = {
  sla: async (days) => {
    const { data } = await api.getOverviewSLA({ days })
    slaCounts.value = { ...slaCounts.value, ...data.data }
  },
  chart: async (days) => {
    const { data } = await api.getOverviewCharts({ days })
    chartData.value = {
      new_conversations: data.data.new_conversations || [],
      resolved_conversations: data.data.resolved_conversations || [],
      messages_sent: data.data.messages_sent || []
    }
  },
  csat: async (days) => {
    const { data } = await api.getOverviewCSAT({ days })
    csatData.value = { ...csatData.value, ...data.data }
  },
  messageVolume: async (days) => {
    const { data } = await api.getOverviewMessageVolume({ days })
    messageVolumeData.value = { ...messageVolumeData.value, ...data.data }
  },
  tagDistribution: async (days) => {
    const { data } = await api.getOverviewTagDistribution({ days })
    tagDistributionData.value = { ...tagDistributionData.value, ...data.data }
  }
}

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

const runSectionFetch = async (key, days) => {
  try {
    await sections[key](days)
  } catch (error) {
    showError(error)
  }
}

const fetchSections = async () => {
  isLoading.value = true
  try {
    await Promise.allSettled(Object.keys(sections).map((key) => runSectionFetch(key, range.value)))
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

const selectRange = (option) => {
  isCustom.value = false
  if (option === range.value) return
  markActive()
  range.value = option
  fetchSections()
}

const enableCustom = () => {
  isCustom.value = true
  customDays.value = String(range.value)
}

const applyCustom = () => {
  let days = parseInt(customDays.value)
  if (!days || days < 1 || days > 365) {
    days = 30
    customDays.value = '30'
  }
  if (days === range.value) return
  markActive()
  range.value = days
  fetchSections()
}

const loadDashboardData = async () => {
  isLoading.value = true
  try {
    await Promise.allSettled([
      fetchCardStats(),
      ...Object.keys(sections).map((key) => runSectionFetch(key, range.value))
    ])
  } finally {
    isLoading.value = false
    lastUpdate.value = new Date()
  }
}

let pollingSince = Date.now()

const scheduledRefresh = () => {
  if (Date.now() - pollingSince >= MAX_POLL_DURATION_MS) {
    pause()
    autoRefreshPaused.value = true
    return
  }
  loadDashboardData()
}

const visibility = useDocumentVisibility()
const { pause, resume } = useIntervalFn(scheduledRefresh, REFRESH_INTERVAL_MS)

const markActive = () => {
  pollingSince = Date.now()
  autoRefreshPaused.value = false
  resume()
}

const manualRefresh = () => {
  markActive()
  loadDashboardData()
}

watch(visibility, (v) => {
  if (v === 'visible') {
    markActive()
    if (Date.now() - lastUpdate.value.getTime() >= REFRESH_INTERVAL_MS) loadDashboardData()
  } else {
    pause()
  }
})

onMounted(loadDashboardData)
</script>

<style scoped>
.metric-value {
  @apply text-3xl font-bold tracking-tight tabular-nums;
}

.metric-label {
  @apply text-xs text-muted-foreground uppercase tracking-wider;
}

.card-title {
  @apply text-sm font-medium text-muted-foreground;
}

.metric-item {
  @apply flex flex-col items-center gap-1 text-center;
}

.section-title {
  @apply text-sm font-medium text-center text-muted-foreground uppercase tracking-wider;
}
</style>
