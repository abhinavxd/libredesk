import { usePushNotifications } from '@main/composables/usePushNotifications'

const defaultBrowser = typeof window === 'undefined' ? {} : window

export const createLogout = (pushNotifications, browser = defaultBrowser) => async () => {
  await Promise.resolve(pushNotifications.disable()).catch(() => {})
  browser.location.href = '/logout'
}

export const useLogout = () => createLogout(usePushNotifications())
