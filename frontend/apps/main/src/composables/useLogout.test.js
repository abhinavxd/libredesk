import { describe, expect, it, vi } from 'vitest'
import { createLogout } from './useLogout'

describe('createLogout', () => {
  it('clears the push association before logging out', async () => {
    const pushNotifications = { clearServerSubscription: vi.fn().mockResolvedValue() }
    const browser = { location: { href: '' } }

    await createLogout(pushNotifications, browser)()

    expect(pushNotifications.clearServerSubscription).toHaveBeenCalled()
    expect(browser.location.href).toBe('/logout')
  })

  it('logs out when clearing the push association fails', async () => {
    const pushNotifications = { clearServerSubscription: vi.fn().mockRejectedValue(new Error('offline')) }
    const browser = { location: { href: '' } }

    await createLogout(pushNotifications, browser)()

    expect(browser.location.href).toBe('/logout')
  })
})
