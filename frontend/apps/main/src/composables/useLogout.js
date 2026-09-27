import { usePushNotifications } from '@main/composables/usePushNotifications'
import { clearNewConversationDrafts } from '@main/features/conversation/useNewConversationDraft.js'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const createLogout = (pushNotifications, browser = defaultBrowser) => async () => {
  await Promise.resolve(pushNotifications.disable()).catch(() => {})
  clearNewConversationDrafts(browser.localStorage)
  browser.location.href = '/logout'
}

export const useLogout = () => createLogout(usePushNotifications())
