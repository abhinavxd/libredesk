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

  it('clears drafts and redirects when the session is already gone', async () => {
    const pushNotifications = { disable: vi.fn().mockResolvedValue() }
    const error = Object.assign(new Error('session expired'), { response: { status: 401 } })
    const apiClient = { logout: vi.fn().mockRejectedValue(error) }
    const browser = {
      location: { href: '/inboxes/assigned' },
      localStorage: { removeItem: vi.fn() }
    }

    await createLogout(pushNotifications, browser, apiClient)()

    expect(browser.location.href).toBe('/')
    expect(browser.localStorage.removeItem.mock.calls).toEqual([
      ['newConversationDraftEmail'],
      ['newConversationDraftWhatsApp']
    ])
  })

  it.each([
    ['network', undefined],
    ['forbidden', 403],
    ['server', 500]
  ])('preserves the page and drafts on a %s failure', async (_, status) => {
    const pushNotifications = { disable: vi.fn().mockResolvedValue() }
    const error = new Error('logout failed')
    if (status !== undefined) error.response = { status }
    const apiClient = { logout: vi.fn().mockRejectedValue(error) }
    const browser = {
      location: { href: '/inboxes/assigned' },
      localStorage: { removeItem: vi.fn() }
    }

    await expect(createLogout(pushNotifications, browser, apiClient)()).rejects.toBe(error)

    expect(browser.location.href).toBe('/inboxes/assigned')
    expect(browser.localStorage.removeItem).not.toHaveBeenCalled()
  })
})
