<template>
  <form class="mt-8 space-y-4 border-t pt-6" @submit.prevent="save">
    <h2 class="text-lg font-medium">{{ t('admin.resourcePolicy.title') }}</h2>
    <p class="text-sm text-muted-foreground">{{ t('admin.resourcePolicy.description') }}</p>
    <label class="block space-y-2">
      <span>{{ t('admin.resourcePolicy.mode') }}</span>
      <select v-model="mode" class="block w-full rounded border bg-background p-2" :disabled="loading || saving">
        <option value="block_all">{{ t('admin.resourcePolicy.blockAll') }}</option>
        <option value="allowlist">{{ t('admin.resourcePolicy.allowlist') }}</option>
        <option value="load_on_receipt">{{ t('admin.resourcePolicy.loadOnReceipt') }}</option>
      </select>
    </label>
    <p class="text-sm text-muted-foreground">{{ t(modeDescription) }}</p>
    <label v-if="mode === 'allowlist'" class="block space-y-2">
      <span>{{ t('admin.resourcePolicy.domains') }}</span>
      <Textarea v-model="domains" :disabled="loading || saving" rows="5" placeholder="images.example.com" />
    </label>
    <p v-if="mode === 'allowlist'" class="text-sm text-muted-foreground">{{ t('admin.resourcePolicy.domainHint') }}</p>
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
    <Button type="submit" :disabled="loading || saving || !loaded">{{ t('admin.resourcePolicy.save') }}</Button>
  </form>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Textarea } from '@shared-ui/components/ui/textarea'
import api from '@/api'
import { useConversationStore } from '@/stores/conversation'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter.js'

const { t } = useI18n()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const mode = ref('load_on_receipt')
const modeDescription = computed(() => ({
  block_all: 'admin.resourcePolicy.blockAllHint',
  allowlist: 'admin.resourcePolicy.allowlistHint',
  load_on_receipt: 'admin.resourcePolicy.loadOnReceiptHint'
})[mode.value])
const domains = ref('')
const loading = ref(true)
const loaded = ref(false)
const saving = ref(false)
const error = ref('')

onMounted(async () => {
  try {
    const response = await api.getResourcePolicy()
    mode.value = response.data.data.mode
    domains.value = response.data.data.allowed_domains.join('\n')
    loaded.value = true
  } catch {
    error.value = t('admin.resourcePolicy.loadFailed')
  } finally {
    loading.value = false
  }
})

const save = async () => {
  saving.value = true
  error.value = ''
  try {
    const response = await api.updateResourcePolicy({
      mode: mode.value,
      allowed_domains: domains.value.split('\n').map(domain => domain.trim()).filter(Boolean)
    })
    domains.value = response.data.data.allowed_domains.join('\n')
    conversationStore.invalidateImageDisplays()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch {
    error.value = t('admin.resourcePolicy.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>
