import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useStorage } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { useUserStore } from './user'
import { getI18n } from '@main/i18n'

export function mailNotificationContent(message, fallbackSender, fallbackMessage) {
  const sender = message.sender || message.conversation?.correspondent || {}
  const name = [sender.first_name, sender.last_name].filter(Boolean).join(' ').trim()
  const preview =
    typeof message.preview === 'string' ? message.preview.replace(/\s+/g, ' ').trim() : ''
  return {
    title: name || sender.email || fallbackSender,
    body: preview ? (preview.length > 240 ? `${preview.slice(0, 240)}…` : preview) : fallbackMessage
  }
}

export const useBrowserNotificationsStore = defineStore('browserNotifications', () => {
  const userStore = useUserStore()
  const router = useRouter()
  // Opt-in is specific to this account and browser, and is synchronized across tabs.
  const preferences = useStorage('libredesk.mail-notifications', {})
  const supported =
    typeof window !== 'undefined' && window.isSecureContext && 'Notification' in window
  const permission = ref(supported ? Notification.permission : 'unsupported')
  const requesting = ref(false)
  const error = ref(false)
  const enabled = computed(() => Boolean(userStore.userID && preferences.value[userStore.userID]))
  const active = computed(() => enabled.value && permission.value === 'granted')
  const shown = new Set()
  const openNotifications = new Set()

  function refreshPermission() {
    permission.value = supported ? Notification.permission : 'unsupported'
  }

  function closeAll() {
    for (const notification of openNotifications) notification.close()
    openNotifications.clear()
  }

  watch(
    () => userStore.userID,
    () => {
      closeAll()
      shown.clear()
    }
  )
  watch(enabled, (value) => {
    if (!value) closeAll()
  })

  async function enable() {
    if (!supported || requesting.value || !userStore.userID) return
    const userID = userStore.userID
    requesting.value = true
    error.value = false
    try {
      // Called only from the enable button, so the permission request has a user gesture.
      permission.value = await Notification.requestPermission()
      if (permission.value === 'granted' && userID === userStore.userID) {
        preferences.value = { ...preferences.value, [userID]: true }
      }
    } catch {
      error.value = true
      refreshPermission()
    } finally {
      requesting.value = false
    }
  }

  function disable() {
    preferences.value = { ...preferences.value, [userStore.userID]: false }
    closeAll()
  }

  function notifyNewMessage(message) {
    refreshPermission()
    if (
      !active.value ||
      !message?.uuid ||
      !message.conversation_uuid ||
      message.type !== 'incoming' ||
      message.sender_type !== 'contact' ||
      message.private ||
      shown.has(message.uuid)
    )
      return

    const t = getI18n().global.t
    const content = mailNotificationContent(
      message,
      t('mailNotifications.newMail'),
      t('mailNotifications.emptyMessage')
    )
    try {
      const notification = new Notification(content.title, {
        body: content.body,
        // The browser replaces duplicates of the same event across open tabs.
        tag: `libredesk-mail:${userStore.userID}:${message.uuid}`,
        icon: '/images/pwa-icon-192.png'
      })
      shown.add(message.uuid)
      if (shown.size > 200) shown.delete(shown.values().next().value)
      openNotifications.add(notification)
      notification.onclose = () => openNotifications.delete(notification)
      notification.onclick = () => {
        notification.close()
        openNotifications.delete(notification)
        window.focus()
        router.push({
          name: 'inbox-conversation',
          params: { type: 'all', uuid: message.conversation_uuid }
        })
      }
      notification.onerror = () => {
        error.value = true
      }
      error.value = false
    } catch {
      // Unsupported browsers or OS policy must not interrupt live mailbox updates.
      error.value = true
    }
  }

  return {
    closeAll,
    supported,
    permission,
    requesting,
    error,
    enabled,
    active,
    refreshPermission,
    enable,
    disable,
    notifyNewMessage
  }
})
