import { ref } from 'vue'
import api from '@main/api'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const urlBase64ToUint8Array = (value) => {
  const padding = '='.repeat((4 - value.length % 4) % 4)
  const raw = atob((value + padding).replace(/-/g, '+').replace(/_/g, '/'))
  return Uint8Array.from(raw, character => character.charCodeAt(0))
}

export const createPushNotifications = (apiClient = api, browser = defaultBrowser) => {
  const supported = ref(Boolean(browser.Notification && browser.navigator?.serviceWorker && browser.PushManager))
  const permission = ref(browser.Notification?.permission || 'unsupported')
  const enabled = ref(false)
  let registration = null

  const registerWorker = async () => {
    registration ??= await browser.navigator.serviceWorker.register('/sw.js')
    return registration
  }

  const save = async (subscription) => {
    const value = subscription.toJSON()
    await apiClient.createPushSubscription({
      endpoint: value.endpoint,
      p256dh: value.keys.p256dh,
      auth: value.keys.auth
    })
  }

  const refresh = async (vapidPublicKey) => {
    if (!supported.value || !vapidPublicKey) return
    const worker = await registerWorker()
    const subscription = await worker.pushManager.getSubscription()
    enabled.value = Boolean(subscription)
    if (subscription) await save(subscription)
  }

  const enable = async (vapidPublicKey) => {
    if (!supported.value || !vapidPublicKey) return false
    permission.value = await browser.Notification.requestPermission()
    if (permission.value !== 'granted') return false
    const worker = await registerWorker()
    const existing = await worker.pushManager.getSubscription()
    const subscription = existing || await worker.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(vapidPublicKey)
    })
    await save(subscription)
    enabled.value = true
    return true
  }

  const disable = async () => {
    if (!supported.value) return
    const worker = await registerWorker()
    const subscription = await worker.pushManager.getSubscription()
    if (!subscription) {
      enabled.value = false
      return
    }
    await subscription.unsubscribe()
    await apiClient.deletePushSubscription(subscription.endpoint)
    enabled.value = false
  }

  const clearServerSubscription = async () => {
    if (!supported.value) return
    const worker = await registerWorker()
    const subscription = await worker.pushManager.getSubscription()
    if (subscription) await apiClient.deletePushSubscription(subscription.endpoint)
  }

  return { supported, permission, enabled, refresh, enable, disable, clearServerSubscription }
}

const pushNotifications = createPushNotifications()

export const usePushNotifications = () => pushNotifications
