import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAppSettingsStore } from './appSettings'

vi.mock('@/api', () => ({ default: {} }))

describe('site branding', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('falls back to Fernmail when the title and logo are unset', () => {
    const store = useAppSettingsStore()
    expect([store.siteTitle, store.siteLogo]).toEqual(['Fernmail', ''])

    store.setSettings({ 'app.site_name': '   ', 'app.logo_url': '' })
    expect([store.siteTitle, store.siteLogo]).toEqual(['Fernmail', ''])
  })

  it('uses the public config on the login screen', () => {
    const store = useAppSettingsStore()
    store.setPublicConfig({ 'app.site_name': 'Acme', 'app.logo_url': '/uploads/logo' })
    expect([store.siteTitle, store.siteLogo]).toEqual(['Acme', '/uploads/logo'])
  })

  it('prefers saved settings, so clearing the title or logo takes effect', () => {
    const store = useAppSettingsStore()
    store.setPublicConfig({ 'app.site_name': 'Acme', 'app.logo_url': '/uploads/logo' })
    store.setSettings({ 'app.site_name': '', 'app.logo_url': '' })
    expect([store.siteTitle, store.siteLogo]).toEqual(['Fernmail', ''])
  })
})
