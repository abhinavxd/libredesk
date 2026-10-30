<template>
  <AdminSplitLayout>
    <template #content>
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
          <AdminToolbar class="mb-5">
            <template #left>
              <TabsList class="w-max justify-start">
                <TabsTrigger value="settings">{{
                  i18n._('Settings')
                }}</TabsTrigger>
                <TabsTrigger value="test">{{ i18n.pgettext('Assistant test tab', 'Test') }}</TabsTrigger>
                <TabsTrigger value="performance">{{
                  i18n.pgettext('Assistant navigation tab', 'Performance')
                }}</TabsTrigger>
              </TabsList>
            </template>
          </AdminToolbar>

          <TabsContent value="settings">
            <AssistantForm
              :initial-values="assistant"
              :is-editing="true"
              :submit-form="submitForm"
            />
          </TabsContent>

          <TabsContent value="test">
            <Card>
              <CardHeader>
                <CardTitle>{{ i18n._('Test assistant') }}</CardTitle>
              </CardHeader>
              <CardContent class="space-y-4">
                <Textarea
                  v-model="previewMessage"
                  rows="3"
                  :placeholder="i18n._('Type a sample customer message')"
                  @keydown.enter.ctrl.prevent="submitPreview"
                  @keydown.enter.meta.prevent="submitPreview"
                />
                <div class="flex items-center justify-end gap-3">
                  <span class="text-xs text-muted-foreground">{{
                    i18n._('Ctrl + Enter to test')
                  }}</span>
                  <Button
                    :isLoading="previewLoading"
                    :disabled="!previewMessage.trim()"
                    @click="runPreview"
                  >
                    {{ i18n.pgettext('Run assistant test', 'Test') }}
                  </Button>
                </div>
                <div class="space-y-2">
                  <div class="text-sm font-medium text-foreground">
                    {{ i18n._('Drafted reply') }}
                  </div>
                  <div
                    class="rounded-md border border-border bg-muted p-3 text-sm whitespace-pre-wrap"
                  >
                    <span v-if="previewReply">{{ previewReply }}</span>
                    <span v-else class="text-muted-foreground">
                      {{ i18n._('The drafted reply will appear here.') }}
                    </span>
                  </div>
                </div>
                <div v-if="previewReferences.length" class="space-y-2">
                  <div class="text-sm font-medium text-foreground">
                    {{ i18n.ngettext('Article reference', 'Article references', 2) }}
                  </div>
                  <ul class="space-y-1">
                    <li v-for="source in previewReferences" :key="source.id">
                      <Button
                        as="a"
                        variant="link"
                        :href="source.url"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="h-auto px-0 whitespace-normal text-left"
                      >
                        {{ source.title }}
                      </Button>
                    </li>
                  </ul>
                </div>
                <div v-if="previewSources.length" class="space-y-2">
                  <div class="text-sm font-medium text-foreground">
                    {{ i18n._('Knowledge used') }}
                  </div>
                  <div class="rounded-md border border-border p-3 space-y-1">
                    <div
                      v-for="source in previewSources"
                      :key="`${source.type}-${source.id}`"
                      class="flex items-center justify-between text-sm"
                    >
                      <span class="text-foreground">
                        {{ source.title }}
                        <span class="ml-2 text-xs text-muted-foreground">
                          {{
                            source.type === 'help_article'
                              ? i18n.ngettext('Article', 'Articles', 1)
                              : i18n.ngettext('Snippet', 'Snippets', 1)
                          }}
                        </span>
                      </span>
                      <span class="text-xs text-muted-foreground">
                        {{ Math.round(source.score * 100) }}%
                      </span>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="performance">
            <Card>
              <CardHeader class="flex flex-row items-center justify-between space-y-0">
                <CardTitle>{{ i18n.pgettext('Assistant statistics heading', 'Performance') }}</CardTitle>
                <div class="flex gap-1">
                  <Button
                    v-for="option in rangeOptions"
                    :key="option"
                    size="sm"
                    :variant="range === option ? 'default' : 'outline'"
                    :disabled="statsLoading"
                    @click="selectRange(option)"
                  >
                    {{ i18n._('{days} days', { days: option }) }}
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

    <template #help>
      <p>{{ i18n._('Assistants reply to conversations you route to them with an automation rule, and hand off to a human when they can\'\'t help. Each one has its own persona, instructions and tools, and they all read from the same knowledge base.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai#assistants"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ i18n._('Learn more') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import AssistantForm from '@/features/admin/ai/AssistantForm.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import { Card, CardContent, CardHeader, CardTitle } from '@shared-ui/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useAIAssistantStore } from '@/stores/aiAssistant'
import { useI18n } from '@shared-ui/i18n'

const props = defineProps({
  id: { type: String, required: false }
})

const i18n = useI18n()
const router = useRouter()
const emitter = useEmitter()
const aiAssistantStore = useAIAssistantStore()
const assistant = ref({})
const isLoading = ref(false)
const stats = ref({})
const range = ref(30)
const statsLoading = ref(false)
const rangeOptions = [7, 30, 90]
const previewMessage = ref('')
const previewReply = ref('')
const previewSources = ref([])
const previewLoading = ref(false)
const previewReferences = computed(() => previewSources.value.filter((source) => source.url))

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
    const base = { label: i18n._(label), value, delta: null }
    if (deltaRaw === undefined || deltaRaw === null) return base
    return { ...base, delta: deltaRaw, ...buildDelta(deltaRaw, unit, increaseIsGood) }
  }
  return [
    tile(
      msg({ message: 'Conversations' }),
      fmtNumber(s.conversations),
      trends.conversations,
      'percent',
      true
    ),
    tile(msg({ message: 'Replies' }), fmtNumber(s.replies), trends.replies, 'percent', true),
    tile(
      msg({ message: 'Resolution rate' }),
      fmtPercent(s.resolution_rate),
      trends.resolution_rate,
      'point',
      true
    ),
    tile(
      msg({ message: 'Handoff rate' }),
      fmtPercent(s.handoff_rate),
      trends.handoff_rate,
      'point',
      false
    ),
    tile(
      msg({ message: 'Reopen rate' }),
      fmtPercent(s.reopen_rate),
      trends.reopen_rate,
      'point',
      false
    ),
    tile(msg({ message: 'Replies per conversation' }), fmtDecimal(s.depth), null),
    tile(
      msg({ message: 'CSAT average' }),
      hasCsat ? fmtDecimal(s.csat_avg) : '-',
      hasCsat ? trends.csat_avg : null,
      'point',
      true
    ),
    tile(msg({ message: 'CSAT positive' }), hasCsat ? fmtPercent(s.csat_positive) : '-', null)
  ]
})

const breadcrumbLinks = [
  { path: 'ai-assistants', label: i18n._('Assistants') },
  { path: '', label: props.id ? i18n._('Edit assistant') : i18n._('New assistant') }
]

const submitForm = async (values) => {
  try {
    if (props.id) {
      await api.updateAIAssistant(props.id, values)
    } else {
      await api.createAIAssistant(values)
      router.push({ name: 'ai-assistants' })
    }
    aiAssistantStore.invalidate()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
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
    previewSources.value = resp.data.data.sources || []
  } catch (error) {
    previewReply.value = ''
    previewSources.value = []
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

const loadAssistant = async () => {
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
}

onMounted(loadAssistant)

// The router reuses this view across ai/assistants/:id/edit, so refetch when the id changes.
watch(
  () => props.id,
  () => {
    previewMessage.value = ''
    previewReply.value = ''
    previewSources.value = []
    loadAssistant()
  }
)
</script>
