<template>
  <div class="flex items-center gap-4">
    <div
      class="flex h-16 w-16 shrink-0 items-center justify-center rounded-md border bg-muted/40 p-2"
    >
      <img :src="modelValue || '/images/fern.svg'" alt="" class="max-h-full max-w-full object-contain" />
    </div>
    <div class="space-y-1.5">
      <div class="flex flex-wrap gap-2">
        <Button type="button" variant="outline" size="sm" :isLoading="uploading" @click="fileInput.click()">
          {{ modelValue ? t('admin.general.siteLogo.replace') : t('admin.general.siteLogo.upload') }}
        </Button>
        <Button
          v-if="modelValue"
          type="button"
          variant="ghost"
          size="sm"
          :disabled="uploading"
          @click="emit('update:modelValue', '')"
        >
          {{ t('admin.general.siteLogo.remove') }}
        </Button>
      </div>
      <p v-if="!modelValue" class="text-xs text-muted-foreground">
        {{ t('admin.general.siteLogo.default') }}
      </p>
    </div>
    <input ref="fileInput" type="file" class="hidden" :accept="ACCEPT" @change="onFileChange" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter.js'
import api from '../../../api'

// Matches the formats the server accepts; it sniffs the bytes regardless.
const ACCEPT = 'image/png,image/jpeg,image/webp,image/x-icon,.ico'

defineProps({ modelValue: { type: String, default: '' } })
const emit = defineEmits(['update:modelValue'])

const { t } = useI18n()
const emitter = useEmitter()
const fileInput = ref(null)
const uploading = ref(false)

// The upload returns the logo's URL; it takes effect when the settings are saved.
const onFileChange = async (event) => {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  const data = new FormData()
  data.append('files', file)
  uploading.value = true
  try {
    const resp = await api.uploadSiteLogo(data)
    emit('update:modelValue', resp.data.data.url)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    uploading.value = false
  }
}
</script>
