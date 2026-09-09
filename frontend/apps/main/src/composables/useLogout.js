import { usePushNotifications } from '@main/composables/usePushNotifications'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const createLogout = (pushNotifications, browser = defaultBrowser) => async () => {
  try {
    await pushNotifications.clearServerSubscription()
  } catch {
    browser.location.href = '/logout'
    return
  }
  browser.location.href = '/logout'
}

export const useLogout = () => createLogout(usePushNotifications())
