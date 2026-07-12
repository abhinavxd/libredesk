<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading">
    <AssistantForm
      v-if="!id"
      :initial-values="assistant"
      :is-editing="false"
      :submit-form="submitForm"
    />

    <Tabs v-else default-value="settings">
      <TabsList class="grid w-full grid-cols-3 mb-5">
        <TabsTrigger value="settings">{{ t('admin.ai.assistant.tabs.settings') }}</TabsTrigger>
        <TabsTrigger value="test">{{ t('admin.ai.assistant.tabs.test') }}</TabsTrigger>
        <TabsTrigger value="performance">{{ t('admin.ai.assistant.tabs.performance') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="settings">
        <AssistantForm :initial-values="assistant" :is-editing="true" :submit-form="submitForm" />
      </TabsContent>

      <TabsContent value="test">
        <Card>
          <CardHeader>
            <CardTitle>{{ t('admin.ai.assistant.preview.title') }}</CardTitle>
            <CardDescription>{{ t('admin.ai.assistant.preview.description') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <Textarea
              v-model="previewMessage"
              rows="3"
              :placeholder="t('admin.ai.assistant.preview.placeholder')"
              @keydown.enter.ctrl.prevent="submitPreview"
              @keydown.enter.meta.prevent="submitPreview"
            />
            <div class="flex items-center justify-end gap-3">
              <span class="text-xs text-muted-foreground">{{
                t('admin.ai.assistant.preview.shortcutHint')
              }}</span>
              <Button
                :isLoading="previewLoading"
                :disabled="!previewMessage.trim()"
                @click="runPreview"
              >
                {{ t('admin.ai.assistant.preview.run') }}
              </Button>
            </div>
            <div class="space-y-2">
              <div class="text-sm font-medium text-foreground">
                {{ t('admin.ai.assistant.preview.replyLabel') }}
              </div>
              <div class="rounded-md border border-border bg-muted p-3 text-sm whitespace-pre-wrap">
                <span v-if="previewReply">{{ previewReply }}</span>
                <span v-else class="text-muted-foreground">
                  {{ t('admin.ai.assistant.preview.empty') }}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="performance">
        <Card>
          <CardHeader class="flex flex-row items-center justify-between space-y-0">
            <CardTitle>{{ t('admin.ai.assistant.stats.title') }}</CardTitle>
            <div class="flex gap-1">
              <Button
                v-for="option in rangeOptions"
                :key="option"
                size="sm"
                :variant="range === option ? 'default' : 'outline'"
                :disabled="statsLoading"
                @click="selectRange(option)"
              >
                {{ t('admin.ai.assistant.stats.rangeDays', { days: option }) }}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div
              class="grid gap-4 grid-cols-2 md:grid-cols-3 xl:grid-cols-5 transition-opacity"
              :class="{ 'opacity-50 pointer-events-none': statsLoading }"
            >
              <div v-for="tile in tiles" :key="tile.label">
                <div class="text-sm text-muted-foreground">{{ tile.label }}</div>
                <div class="text-2xl font-semibold text-foreground">{{ tile.value }}</div>
                <div v-if="tile.delta !== null" class="text-xs" :class="tile.deltaClass">
                  {{ tile.deltaText }}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  </LoadingOverlay>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import AssistantForm from '@/features/admin/ai/AssistantForm.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@shared-ui/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  id: { type: String, required: false }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const assistant = ref({})
const isLoading = ref(false)
const stats = ref({})
const range = ref(30)
const statsLoading = ref(false)
const rangeOptions = [7, 30, 90]
const previewMessage = ref('')
const previewReply = ref('')
const previewLoading = ref(false)

const fmtNumber = (value) => Number(value ?? 0).toLocaleString()
const fmtPercent = (value) => `${Number(value ?? 0).toFixed(1)}%`
const fmtDecimal = (value) => Number(value ?? 0).toFixed(1)

const buildDelta = (raw, unit, increaseIsGood) => {
  const value = raw ?? 0
  const suffix = unit === 'percent' ? '%' : ''
  if (value === 0) {
    return { deltaText: '-', deltaClass: 'text-muted-foreground' }
  }
  const arrow = value > 0 ? '▲' : '▼'
  const good = value > 0 ? increaseIsGood : !increaseIsGood
  return {
    deltaText: `${arrow} ${Math.abs(value).toFixed(1)}${suffix}`,
    deltaClass: good ? 'text-foreground' : 'text-destructive'
  }
}

const tiles = computed(() => {
  const s = stats.value
  const trends = s.trends ?? {}
  const hasCsat = (s.csat_count ?? 0) > 0
  const tile = (label, value, deltaRaw, unit, increaseIsGood) => {
    const base = { label: t(label), value, delta: null }
    if (deltaRaw === undefined || deltaRaw === null) return base
    return { ...base, delta: deltaRaw, ...buildDelta(deltaRaw, unit, increaseIsGood) }
  }
  return [
    tile('admin.ai.assistant.stats.conversations', fmtNumber(s.conversations), trends.conversations, 'percent', true),
    tile('admin.ai.assistant.stats.replies', fmtNumber(s.replies), trends.replies, 'percent', true),
    tile('admin.ai.assistant.stats.resolutionRate', fmtPercent(s.resolution_rate), trends.resolution_rate, 'point', true),
    tile('admin.ai.assistant.stats.handoffRate', fmtPercent(s.handoff_rate), trends.handoff_rate, 'point', false),
    tile('admin.ai.assistant.stats.reopenRate', fmtPercent(s.reopen_rate), trends.reopen_rate, 'point', false),
    tile('admin.ai.assistant.stats.depth', fmtDecimal(s.depth), null),
    tile('admin.ai.assistant.stats.csat', hasCsat ? fmtDecimal(s.csat_avg) : '-', hasCsat ? trends.csat_avg : null, 'point', true),
    tile('admin.ai.assistant.stats.csatPositive', hasCsat ? fmtPercent(s.csat_positive) : '-', null)
  ]
})

const breadcrumbLinks = [
  { path: 'ai-assistants', label: t('admin.ai.assistants') },
  { path: '', label: props.id ? t('admin.ai.assistant.edit') : t('admin.ai.assistant.new') }
]

const submitForm = async (values) => {
  try {
    if (props.id) {
      await api.updateAIAssistant(props.id, values)
    } else {
      await api.createAIAssistant(values)
      router.push({ name: 'ai-assistants' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const submitPreview = () => {
  if (previewLoading.value || !previewMessage.value.trim()) return
  runPreview()
}

const runPreview = async () => {
  try {
    previewLoading.value = true
    const resp = await api.previewAIAssistant(props.id, { message: previewMessage.value })
    previewReply.value = resp.data.data.reply
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    previewLoading.value = false
  }
}

const fetchStats = async () => {
  try {
    statsLoading.value = true
    const resp = await api.getAIAssistantStats(props.id, range.value)
    stats.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    statsLoading.value = false
  }
}

const selectRange = (option) => {
  if (option === range.value) return
  range.value = option
  fetchStats()
}

onMounted(async () => {
  if (!props.id) return
  try {
    isLoading.value = true
    const [assistantResp] = await Promise.all([api.getAIAssistant(props.id), fetchStats()])
    assistant.value = assistantResp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})
</script>
