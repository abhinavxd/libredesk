import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { usePushNotifications } from '@main/composables/usePushNotifications'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { clearNewConversationDrafts } from '@main/features/conversation/useNewConversationDraft.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const createLogout = (pushNotifications, browser = defaultBrowser, apiClient = api) => async () => {
  await Promise.resolve(pushNotifications.disable()).catch(() => {})
  try {
    await apiClient.logout()
  } catch (error) {
    if (error?.response?.status !== 401) throw error
  }
  clearNewConversationDrafts(browser.localStorage)
  browser.location.href = '/'
}

export const useLogout = () => {
  const emitter = useEmitter()
  const logout = createLogout(usePushNotifications())
  return async () => {
    try {
      await logout()
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }
}
