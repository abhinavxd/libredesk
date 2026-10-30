<template>
  <div class="h-full relative">
    <div class="flex flex-col space-y-5 max-w-2xl">
      <div class="space-y-1">
        <span class="sub-title">{{ i18n.ngettext('Notification', 'Notifications', 2) }}</span>
      </div>

      <Spinner v-if="isLoading" />

      <template v-else>
        <p v-if="!emailEnabled" class="text-muted-foreground text-xs">
          {{ i18n._('Email notifications are turned off for this workspace. Contact your administrator to enable them.') }}
        </p>

        <div
          class="flex items-center justify-between gap-4 border border-border rounded-md px-4 py-3"
        >
          <div class="space-y-1">
            <p class="text-sm text-foreground">
              {{ i18n.ngettext('Browser notification', 'Browser notifications', 2) }}
            </p>
            <p v-if="browserPushDescription" class="text-xs text-muted-foreground">
              {{ browserPushDescription }}
            </p>
          </div>
          <DotLoader
            v-if="isUpdatingBrowserPush"
            role="status"
            :aria-label="i18n._('Loading')"
          />
          <Switch
            v-else
            :checked="pushNotifications.enabled.value"
            :disabled="
              !pushNotifications.supported.value ||
              pushNotifications.permission.value === 'denied' ||
              !vapidPublicKey
            "
            :aria-label="i18n.ngettext('Browser notification', 'Browser notifications', 2)"
            @update:checked="updateBrowserPush"
          />
        </div>

        <div class="border border-border rounded-md divide-y divide-border">
          <div
            class="grid grid-cols-3 items-center px-4 py-2 text-xs font-medium text-muted-foreground sm:grid-cols-[minmax(0,1fr)_repeat(3,5rem)]"
          >
            <div class="hidden sm:block" />
            <div v-for="channel in channels" :key="channel.key" class="text-center">
              {{ i18n._(channel.labelMessage) }}
            </div>
          </div>

          <div
            v-for="row in rows"
            :key="row.type"
            class="grid grid-cols-3 items-center gap-y-3 px-4 py-3 sm:grid-cols-[minmax(0,1fr)_repeat(3,5rem)]"
          >
            <div class="col-span-3 min-w-0 sm:col-span-1 sm:pr-4">
              <p class="text-sm text-foreground">{{ typeLabel(row.type) }}</p>
            </div>
            <div v-for="channel in channels" :key="channel.key" class="flex justify-center">
              <Switch
                :checked="row[channel.key]"
                :disabled="
                  (channel.key === 'email' && !emailEnabled) ||
                  pendingUpdates.has(preferenceKey(row.type, channel.key))
                "
                :aria-label="
                  i18n._('{channel} notifications for {type}', {
                    channel: i18n._(channel.labelMessage),
                    type: typeLabel(row.type)
                  })
                "
                @update:checked="update(row, channel.key, $event)"
              />
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

import { computed, ref, onMounted } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { Switch } from '@shared-ui/components/ui/switch'
import { DotLoader } from '@shared-ui/components/ui/loader'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'
import { usePushNotifications } from '@/composables/usePushNotifications'

const i18n = useI18n()
const emitter = useEmitter()
const rows = ref([])
const isLoading = ref(true)
const emailEnabled = ref(true)
const vapidPublicKey = ref('')
const pendingUpdates = ref(new Set())
const isUpdatingBrowserPush = ref(false)
const pushNotifications = usePushNotifications()

const channels = [
  { key: 'in_app', labelMessage: msg({ message: 'In-app' }) },
  { key: 'email', labelMessage: msg({ message: '{count, plural, one {Email} other {Emails}}', context: 'Communication channel' }) },
  { key: 'push', labelMessage: msg({ message: 'Browser' }) }
]

const typeLabels = {
  assignment: msg({ message: 'Conversation assigned to me' }),
  mention: msg({ message: 'Mentioned in a conversation' }),
  conversation_reopened: msg({ message: 'Conversation assigned to me is reopened by a reply' }),
  new_reply: msg({ message: 'New reply on a conversation assigned to me' }),
  new_reply_participating: msg({ message: 'New reply on a conversation I\'\'m participating in' }),
  sla_first_response_warning: msg({ message: 'First response SLA breach warning' }),
  sla_first_response_breach: msg({ message: 'First response SLA breached' }),
  sla_next_response_warning: msg({ message: 'Next response SLA breach warning' }),
  sla_next_response_breach: msg({ message: 'Next response SLA breached' }),
  sla_resolution_warning: msg({ message: 'Resolution SLA breach warning' }),
  sla_resolution_breach: msg({ message: 'Resolution SLA breached' })
}

const typeLabel = (type) => (typeLabels[type] ? i18n._(typeLabels[type]) : type)
const preferenceKey = (type, channel) => `${type}:${channel}`

const fetchPreferences = async () => {
  try {
    const { data } = await api.getNotificationPreferences()
    emailEnabled.value = data.data.email_enabled
    vapidPublicKey.value = data.data.vapid_public_key
    await pushNotifications.refresh(vapidPublicKey.value, data.data.push_endpoints)
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
  } finally {
    isLoading.value = false
  }
}

const browserPushDescription = computed(() => {
  if (!pushNotifications.supported.value) return i18n._('This browser does not support push notifications. Use a supported browser to receive notifications when libredesk is closed.')
  if (pushNotifications.permission.value === 'denied') return i18n._('Browser notifications are blocked. Allow them in your browser settings to receive notifications on this device.')
  if (!vapidPublicKey.value) return i18n._('Browser notifications are unavailable because push is not configured. Ask an administrator to configure push notifications.')
  return ''
})

const updateBrowserPush = async (value) => {
  if (isUpdatingBrowserPush.value) return

  try {
    isUpdatingBrowserPush.value = true
    if (value) {
      const granted = await pushNotifications.enable(vapidPublicKey.value)
      if (!granted) {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          variant: 'destructive',
          description: i18n._('Browser notifications are blocked. Allow them in your browser settings to receive notifications on this device.')
        })
      }
    } else {
      await pushNotifications.disable()
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isUpdatingBrowserPush.value = false
  }
}

const update = async (row, channel, value) => {
  const key = preferenceKey(row.type, channel)
  if (pendingUpdates.value.has(key)) return

  const previous = row[channel]
  row[channel] = value
  pendingUpdates.value.add(key)
  try {
    await api.updateNotificationPreferences([
      { notification_type: row.type, channel, enabled: value }
    ])
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
  } catch (error) {
    row[channel] = previous
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    pendingUpdates.value.delete(key)
  }
}

onMounted(fetchPreferences)
</script>
