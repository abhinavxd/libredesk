<template>
  <div class="h-full">
    <div class="flex flex-col space-y-5 max-w-2xl">
      <div class="space-y-1">
        <span class="sub-title">{{ $t('globals.terms.notification', 2) }}</span>
        <p class="text-muted-foreground text-xs">{{ $t('account.notifications.description') }}</p>
      </div>

      <p v-if="!emailEnabled" class="text-muted-foreground text-xs">
        {{ $t('notification.emailChannelDisabled') }}
      </p>

      <div class="flex items-center justify-between gap-4 border border-border rounded-md px-4 py-3">
        <div class="space-y-1">
          <p class="text-sm text-foreground">{{ $t('notification.browserPush.title') }}</p>
          <p class="text-xs text-muted-foreground">{{ browserPushDescription }}</p>
        </div>
        <Switch
          :checked="pushNotifications.enabled.value"
          :disabled="!pushNotifications.supported.value || pushNotifications.permission.value === 'denied'"
          :aria-label="$t('notification.browserPush.title')"
          @update:checked="updateBrowserPush"
        />
      </div>

      <div class="border border-border rounded-md divide-y divide-border">
        <div class="flex items-center px-4 py-2 text-xs font-medium text-muted-foreground">
          <div class="flex-grow" />
          <div v-for="channel in channels" :key="channel.key" class="w-20 text-center">
            {{ $t(channel.labelKey) }}
          </div>
        </div>

        <div v-for="row in rows" :key="row.type" class="flex items-center px-4 py-3">
          <div class="flex-grow pr-4">
            <p class="text-sm text-foreground">{{ typeLabel(row.type) }}</p>
          </div>
          <div v-for="channel in channels" :key="channel.key" class="w-20 flex justify-center">
            <Switch
              :checked="row[channel.key]"
              :disabled="channel.key === 'email' && !emailEnabled"
              :aria-label="
                $t('notification.channelToggleLabel', {
                  channel: $t(channel.labelKey),
                  type: typeLabel(row.type)
                })
              "
              @update:checked="update(row, channel.key, $event)"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Switch } from '@shared-ui/components/ui/switch'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'
import { usePushNotifications } from '@/composables/usePushNotifications'

const { t } = useI18n()
const emitter = useEmitter()
const rows = ref([])
const emailEnabled = ref(true)
const vapidPublicKey = ref('')
const pushNotifications = usePushNotifications()

const channels = [
  { key: 'in_app', labelKey: 'notification.channel.inApp' },
  { key: 'email', labelKey: 'globals.terms.email' },
  { key: 'push', labelKey: 'notification.channel.push' }
]

const typeLabels = {
  assignment: 'notification.type.assignment',
  mention: 'notification.type.mention',
  conversation_reopened: 'notification.type.conversationReopened',
  new_reply: 'notification.type.newReply',
  new_reply_participating: 'notification.type.newReplyParticipating',
  sla_first_response_warning: 'notification.type.slaFirstResponseWarning',
  sla_first_response_breach: 'notification.type.slaFirstResponseBreach',
  sla_next_response_warning: 'notification.type.slaNextResponseWarning',
  sla_next_response_breach: 'notification.type.slaNextResponseBreach',
  sla_resolution_warning: 'notification.type.slaResolutionWarning',
  sla_resolution_breach: 'notification.type.slaResolutionBreach'
}

const typeLabel = (type) => (typeLabels[type] ? t(typeLabels[type]) : type)

const fetchPreferences = async () => {
  try {
    const { data } = await api.getNotificationPreferences()
    emailEnabled.value = data.data.email_enabled
    vapidPublicKey.value = data.data.vapid_public_key
    await pushNotifications.refresh(vapidPublicKey.value)
    const byType = {}
    for (const pref of data.data.preferences) {
      byType[pref.notification_type] ??= { type: pref.notification_type }
      byType[pref.notification_type][pref.channel] = pref.enabled
    }
    rows.value = Object.values(byType)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const browserPushDescription = computed(() => {
  if (!pushNotifications.supported.value) return t('notification.browserPush.unsupported')
  if (pushNotifications.permission.value === 'denied') return t('notification.browserPush.blocked')
  return t('notification.browserPush.description')
})

const updateBrowserPush = async (value) => {
  try {
    if (value) {
      await pushNotifications.enable(vapidPublicKey.value)
    } else {
      await pushNotifications.disable()
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const update = async (row, channel, value) => {
  const previous = row[channel]
  row[channel] = value
  try {
    await api.updateNotificationPreferences([
      { notification_type: row.type, channel, enabled: value }
    ])
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    row[channel] = previous
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

onMounted(fetchPreferences)
</script>
