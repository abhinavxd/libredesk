import api from '@main/api'
import { usePushNotifications } from '@main/composables/usePushNotifications'
import { clearNewConversationDrafts } from '@main/features/conversation/useNewConversationDraft.js'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const createLogout = (pushNotifications, browser = defaultBrowser, apiClient = api) => async () => {
  await Promise.resolve(pushNotifications.disable()).catch(() => {})
  clearNewConversationDrafts(browser.localStorage)
  await apiClient.logout().catch(() => {})
  browser.location.href = '/'
}

export const useLogout = () => createLogout(usePushNotifications())
