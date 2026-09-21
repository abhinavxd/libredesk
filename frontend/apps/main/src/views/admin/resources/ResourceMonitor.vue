<template>
  <AdminSplitLayout>
    <template #content>
      <div class="space-y-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h1 class="text-xl font-semibold">{{ t('admin.systemResources.title') }}</h1>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ t('admin.systemResources.description') }}
            </p>
          </div>
          <Button variant="outline" size="sm" :disabled="refreshing" @click="refresh">
            <RefreshCw class="mr-2 h-4 w-4" :class="{ 'animate-spin': refreshing }" />
            {{ t('admin.systemResources.refresh') }}
          </Button>
        </div>

        <p
          v-if="error"
          role="alert"
          class="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive"
        >
          {{ error }}
        </p>

        <form class="space-y-4 rounded-md border bg-muted/10 p-4" @submit.prevent="saveLimits">
          <div>
            <h2 class="text-lg font-medium">{{ t('admin.systemResources.limitsTitle') }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">
              {{ t('admin.systemResources.limitsDescription') }}
            </p>
          </div>
          <div class="grid gap-4 md:grid-cols-3">
            <label class="block space-y-2">
              <span>{{ t('admin.systemResources.maxEmailSize') }}</span>
              <Input
                v-model.number="maxEmailMiB"
                type="number"
                min="0"
                step="1"
                required
                :disabled="loading || saving || !limitsLoaded"
              />
              <span class="block text-xs text-muted-foreground">
                {{ t('admin.systemResources.maxEmailSizeHint') }}
              </span>
            </label>
            <label class="block space-y-2">
              <span>{{ t('admin.systemResources.maxDurableStorage') }}</span>
              <Input
                v-model.number="maxStorageGiB"
                type="number"
                min="0"
                step="0.1"
                required
                :disabled="loading || saving || !limitsLoaded"
              />
              <span class="block text-xs text-muted-foreground">
                {{ t('admin.systemResources.maxDurableStorageHint') }}
              </span>
            </label>
            <label class="block space-y-2">
              <span>{{ t('admin.systemResources.maxImageCache') }}</span>
              <Input
                v-model.number="maxImageCacheGiB"
                type="number"
                min="0.000000001"
                step="0.1"
                required
                :disabled="loading || saving || !limitsLoaded"
              />
              <span class="block text-xs text-muted-foreground">
                {{ t('admin.systemResources.maxImageCacheHint') }}
              </span>
            </label>
          </div>
          <p class="text-sm text-muted-foreground">
            {{ t('admin.systemResources.infrastructureLimitNote') }}
          </p>
          <p v-if="limitError" role="alert" class="text-sm text-destructive">{{ limitError }}</p>
          <p v-if="combinedConfiguredLimit" class="text-sm font-medium text-foreground">
            {{ t('admin.systemResources.combinedLimit') }}:
            {{ formatBytes(combinedConfiguredLimit) }}
          </p>
          <Button type="submit" :disabled="loading || saving || !limitsLoaded">
            {{ saving ? t('admin.systemResources.saving') : t('admin.systemResources.saveLimits') }}
          </Button>
        </form>

        <div v-if="usage" class="grid gap-4 md:grid-cols-2">
          <ResourceCard
            :title="t('admin.systemResources.memory')"
            :used="usage.memory.current_bytes"
            :limit="usage.memory.limit_bytes"
            :percent="usage.memory.usage_percent"
            :detail="
              usage.memory.peak_bytes
                ? `${t('admin.systemResources.used')}: ${formatBytes(usage.memory.current_bytes)} · Peak: ${formatBytes(usage.memory.peak_bytes)}`
                : ''
            "
          />
          <ResourceCard
            :title="t('admin.systemResources.storage')"
            :used="usage.storage.used_bytes"
            :limit="usage.storage.limit_bytes"
            :percent="usage.storage.usage_percent"
            :detail="t('admin.systemResources.used') + ': ' + formatBytes(usage.storage.used_bytes)"
          />
          <ResourceCard
            :title="t('admin.systemResources.imageCache')"
            :used="usage.image_cache.used_bytes"
            :limit="usage.image_cache.limit_bytes"
            :percent="usage.image_cache.usage_percent"
            :detail="
              t('admin.systemResources.used') + ': ' + formatBytes(usage.image_cache.used_bytes)
            "
          />
          <ResourceCard
            :title="t('admin.systemResources.uploadFilesystem')"
            :used="usage.disk.used_bytes"
            :limit="usage.disk.limit_bytes"
            :percent="usage.disk.usage_percent"
            :detail="
              usage.disk.error
                ? t('admin.systemResources.diskUnavailable', { error: usage.disk.error })
                : `${t('admin.systemResources.available')}: ${formatBytes(usage.disk.available_bytes)}`
            "
          />
        </div>

        <div v-if="usage" class="rounded-md border bg-muted/20 p-4 text-sm text-muted-foreground">
          <p>{{ t('admin.systemResources.help') }}</p>
          <p class="mt-2">{{ t('admin.systemResources.containerNote') }}</p>
          <p v-if="combinedLimit" class="mt-2 font-medium text-foreground">
            {{ t('admin.systemResources.combinedLimit') }}: {{ formatBytes(combinedLimit) }}
          </p>
          <p v-if="sampledAt" class="mt-2 text-xs">
            {{ t('admin.systemResources.lastSampled', { time: sampledAt }) }}
          </p>
        </div>
      </div>
    </template>

    <template #help>
      <p>{{ t('admin.systemResources.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import ResourceCard from './ResourceCard.vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter.js'

const { t } = useI18n()
const usage = ref(null)
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const limitError = ref('')
const limitsLoaded = ref(false)
const saving = ref(false)
const maxEmailMiB = ref(100)
const maxStorageGiB = ref(0)
const maxImageCacheGiB = ref(10)
const resourcePolicy = ref(null)
const emitter = useEmitter()
const GiB = 2 ** 30
const MiB = 2 ** 20
let refreshTimer

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

const sampledAt = computed(() => {
  if (!usage.value?.sampled_at) return ''
  return new Date(usage.value.sampled_at).toLocaleTimeString()
})

const combinedLimit = computed(() => {
  const storageLimit = usage.value?.storage?.limit_bytes
  const imageLimit = usage.value?.image_cache?.limit_bytes
  return storageLimit === null || storageLimit === undefined ? null : storageLimit + imageLimit
})

const combinedConfiguredLimit = computed(() => {
  const storage = Number(maxStorageGiB.value)
  const imageCache = Number(maxImageCacheGiB.value)
  if (
    !Number.isFinite(storage) ||
    storage <= 0 ||
    !Number.isFinite(imageCache) ||
    imageCache <= 0
  ) {
    return null
  }
  return Math.round(storage * GiB) + Math.round(imageCache * GiB)
})

const refresh = async () => {
  refreshing.value = true
  try {
    const response = await api.getResourceUsage()
    usage.value = response.data.data
    error.value = ''
  } catch {
    error.value = t('admin.systemResources.loadFailed')
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

// Only initialize the editable fields when the page opens. Metric polling
// must never replace a draft, including while a save request is in flight.
const loadLimits = async () => {
  try {
    const [limitsResponse, policyResponse] = await Promise.all([
      api.getResourceLimits(),
      api.getResourcePolicy()
    ])
    const limits = limitsResponse.data.data
    maxEmailMiB.value = limits.max_incoming_message_size / MiB
    maxStorageGiB.value = limits.max_storage_bytes / GiB
    resourcePolicy.value = policyResponse.data.data
    maxImageCacheGiB.value = resourcePolicy.value.max_cache_bytes / GiB
    limitsLoaded.value = true
  } catch {
    limitError.value = t('admin.systemResources.loadFailed')
  }
}

const saveLimits = async () => {
  limitError.value = ''
  const maxIncomingMessageSize = Math.round(Number(maxEmailMiB.value) * MiB)
  const maxStorageBytes = Math.round(Number(maxStorageGiB.value) * GiB)
  const maxCacheBytes = Math.round(Number(maxImageCacheGiB.value) * GiB)
  if (
    !Number.isSafeInteger(maxIncomingMessageSize) ||
    maxIncomingMessageSize < 0 ||
    !Number.isSafeInteger(maxStorageBytes) ||
    maxStorageBytes < 0 ||
    !Number.isSafeInteger(maxCacheBytes) ||
    maxCacheBytes <= 0
  ) {
    limitError.value = t('admin.systemResources.invalidLimits')
    return
  }
  saving.value = true
  try {
    await api.updateResourceLimits({
      max_incoming_message_size: maxIncomingMessageSize,
      max_storage_bytes: maxStorageBytes
    })
    const policyResponse = await api.updateResourcePolicy({
      max_cache_bytes: maxCacheBytes
    })
    resourcePolicy.value = policyResponse.data.data
    maxImageCacheGiB.value = policyResponse.data.data.max_cache_bytes / GiB
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    await refresh()
  } catch {
    limitError.value = t('admin.systemResources.saveFailed')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadLimits()
  refresh()
  refreshTimer = setInterval(refresh, 30000)
})

onUnmounted(() => clearInterval(refreshTimer))
</script>
