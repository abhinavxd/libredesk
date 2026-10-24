import { describe, expect, it, vi } from 'vitest'
import { createLogout } from './useLogout'

describe('createLogout', () => {
  it('clears the push association before logging out', async () => {
    const pushNotifications = { disable: vi.fn().mockResolvedValue() }
    const apiClient = { logout: vi.fn().mockResolvedValue() }
    const browser = { location: { href: '' } }

    await createLogout(pushNotifications, browser, apiClient)()

    expect(pushNotifications.disable).toHaveBeenCalled()
    expect(apiClient.logout).toHaveBeenCalled()
    expect(browser.location.href).toBe('/')
  })

  it('logs out when clearing the push association fails', async () => {
    const pushNotifications = { disable: vi.fn().mockRejectedValue(new Error('offline')) }
    const apiClient = { logout: vi.fn().mockResolvedValue() }
    const browser = { location: { href: '' } }

    await createLogout(pushNotifications, browser, apiClient)()

    expect(apiClient.logout).toHaveBeenCalled()
    expect(browser.location.href).toBe('/')
  })
})
