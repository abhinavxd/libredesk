<template>
  <div class="h-full">
    <div class="flex flex-col space-y-5 max-w-2xl">
      <div class="space-y-1">
        <span class="sub-title">{{ $t('globals.terms.preferences') }}</span>
      </div>

      <div class="border border-border rounded-md px-4 py-3 flex items-center justify-between gap-4">
        <div class="space-y-1">
          <p class="text-sm text-foreground">{{ $t('account.preferences.sendDelay') }}</p>
          <p class="text-xs text-muted-foreground">{{ $t('account.preferences.sendDelay.description') }}</p>
        </div>
        <Select :modelValue="String(sendDelaySeconds)" @update:modelValue="updateSendDelay">
          <SelectTrigger class="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="0">{{ $t('globals.terms.off') }}</SelectItem>
              <SelectItem value="5">5s</SelectItem>
              <SelectItem value="10">10s</SelectItem>
              <SelectItem value="15">15s</SelectItem>
              <SelectItem value="20">20s</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useUserStore } from '@/stores/user'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const userStore = useUserStore()

const sendDelaySeconds = computed(() => userStore.preferences.send_delay_seconds ?? 0)

const updateSendDelay = async (val) => {
  const updated = { ...userStore.preferences, send_delay_seconds: Number(val) }
  try {
    const { data } = await api.updateCurrentUserPreferences(updated)
    userStore.user.preferences = data.data.preferences
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
</script>
