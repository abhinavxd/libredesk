<template>
  <div class="bg-muted/30 box p-4 space-y-4">
    <div>
      <p class="text-base font-semibold text-foreground">
        {{ $t('globals.terms.apiKey', 2) }}
      </p>
      <p class="text-sm text-muted-foreground">
        {{ description }}
      </p>
    </div>

    <div v-if="apiKey" class="space-y-3">
      <div class="space-y-2">
        <Label class="text-sm font-medium">{{ $t('globals.terms.apiKey') }}</Label>
        <div class="flex items-center gap-2">
          <Input :model-value="apiKey" readonly class="font-mono text-sm" />
          <CopyButton :text="apiKey" variant="outline" size="sm" :show-text="false" />
        </div>
      </div>

      <div v-if="secret" class="space-y-2">
        <Label class="text-sm font-medium">{{ $t('globals.terms.secret') }}</Label>
        <div class="flex items-center gap-2">
          <Input
            :model-value="revealSecret ? secret : '•'.repeat(24)"
            readonly
            class="font-mono text-sm"
          />
          <Button type="button" variant="outline" size="sm" @click="revealSecret = !revealSecret">
            <EyeOff v-if="revealSecret" class="w-4 h-4" />
            <Eye v-else class="w-4 h-4" />
            <span class="sr-only">
              {{ revealSecret ? $t('account.apiKey.hideSecret') : $t('account.apiKey.showSecret') }}
            </span>
          </Button>
          <CopyButton :text="secret" variant="outline" size="sm" :show-text="false" />
        </div>
      </div>
      <p v-else class="text-xs text-muted-foreground">
        {{ $t('account.apiKey.secretUnavailable') }}
      </p>

      <div v-if="lastUsedAt" class="text-xs text-muted-foreground">
        {{ $t('globals.messages.lastUsed') }}:
        {{ format(new Date(lastUsedAt), 'PPpp') }}
      </div>

      <div class="flex gap-2">
        <Button type="button" variant="outline" size="sm" @click="generate" :disabled="loading">
          <RotateCcw class="w-4 h-4 mr-1" />
          {{ $t('globals.messages.regenerate') }}
        </Button>
        <Button type="button" variant="destructive" size="sm" @click="revoke" :disabled="loading">
          <Trash2 class="w-4 h-4 mr-1" />
          {{ $t('globals.messages.revoke') }}
        </Button>
      </div>
    </div>

    <div v-else class="text-center py-6">
      <Key class="w-8 h-8 text-muted-foreground mx-auto mb-2" />
      <p class="text-sm text-muted-foreground mb-3">{{ emptyLabel }}</p>
      <Button type="button" @click="generate" :disabled="loading">
        <Plus class="w-4 h-4" />
        {{ $t('agent.generateApiKey') }}
      </Button>
    </div>

    <div v-if="apiKey && secret" class="border-t pt-4 space-y-3">
      <div>
        <p class="text-sm font-semibold">{{ $t('account.apiKey.mcpTitle') }}</p>
        <p class="text-xs text-muted-foreground">{{ $t('account.apiKey.mcpDescription') }}</p>
      </div>
      <div class="space-y-2">
        <Label class="text-sm font-medium">{{ $t('account.apiKey.mcpUrl') }}</Label>
        <div class="flex items-center gap-2">
          <Input :model-value="mcpUrl" readonly class="font-mono text-sm" />
          <CopyButton :text="mcpUrl" variant="outline" size="sm" :show-text="false" />
        </div>
      </div>
      <p class="text-xs text-muted-foreground">{{ $t('account.apiKey.mcpAuth') }}</p>
      <div class="flex items-start gap-2">
        <textarea
          :value="mcpConfig"
          readonly
          rows="10"
          class="flex-1 min-h-[10rem] rounded-md border bg-background px-3 py-2 font-mono text-xs"
        />
        <CopyButton :text="mcpConfig" variant="outline" size="sm" :show-text="false" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import { Key, RotateCcw, Trash2, Plus, Eye, EyeOff } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import CopyButton from '@/components/button/CopyButton.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const props = defineProps({
  apiKey: { type: String, default: '' },
  lastUsedAt: { type: [String, Date], default: null },
  description: { type: String, required: true },
  emptyLabel: { type: String, required: true },
  generateFn: { type: Function, required: true },
  revokeFn: { type: Function, required: true },
  fetchFn: { type: Function, default: null }
})

const emit = defineEmits(['updated'])
const { t } = useI18n()
const emitter = useEmitter()
const loading = ref(false)
const apiKey = ref(props.apiKey || '')
const secret = ref('')
const lastUsedAt = ref(props.lastUsedAt || null)
const revealSecret = ref(false)

const mcpUrl = computed(() => `${window.location.origin}/mcp`)
const mcpConfig = computed(() =>
  JSON.stringify(
    {
      mcpServers: {
        libredesk: {
          url: mcpUrl.value,
          headers: {
            Authorization: `Basic ${btoa(`${apiKey.value}:${secret.value}`)}`
          }
        }
      }
    },
    null,
    2
  )
)

watch(
  () => [props.apiKey, props.lastUsedAt],
  ([key, used]) => {
    apiKey.value = key || ''
    lastUsedAt.value = used || null
  }
)

const applyCreds = (data) => {
  if (!data) return
  apiKey.value = data.api_key || ''
  secret.value = data.secret_available ? data.api_secret || '' : ''
  lastUsedAt.value = data.api_key_last_used_at || null
}

onMounted(async () => {
  if (!props.fetchFn) return
  try {
    const response = await props.fetchFn()
    applyCreds(response?.data?.data)
  } catch {
    /* keep props */
  }
})

const generate = async () => {
  try {
    loading.value = true
    const response = await props.generateFn()
    const data = response?.data?.data
    if (!data) throw new Error('No API key returned')
    apiKey.value = data.api_key
    secret.value = data.api_secret || ''
    lastUsedAt.value = null
    revealSecret.value = true
    emit('updated', { api_key: data.api_key, api_key_last_used_at: null })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('agent.apiKeyGenerated')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}

const revoke = async () => {
  try {
    loading.value = true
    await props.revokeFn()
    apiKey.value = ''
    secret.value = ''
    lastUsedAt.value = null
    revealSecret.value = false
    emit('updated', { api_key: '', api_key_last_used_at: null })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('agent.apiKeyRevoked')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
}
</script>
