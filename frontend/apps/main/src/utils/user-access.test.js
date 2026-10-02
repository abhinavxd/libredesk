import { describe, expect, it } from 'vitest'
import {
  accessDraft,
  describeAccessChanges,
  filterUsers,
  hasAccessChanges,
  initials,
  legacyRoles,
  primaryRole
} from './user-access'

const casey = { id: 4, first_name: 'Casey', last_name: 'Lee', email: 'casey@example.test', roles: ['Agent'], address_ids: [3, 1], enabled: true }

describe('user access helpers', () => {
  it('picks the built-in role and flags roles from older releases', () => {
    expect(primaryRole(['Billing', 'Agent'])).toBe('Agent')
    expect(primaryRole(['Billing'])).toBe('Billing')
    expect(legacyRoles(['Agent'])).toEqual([])
    expect(legacyRoles(['Billing', 'Agent'])).toEqual(['Billing'])
    expect(legacyRoles(['Billing'])).toEqual(['Billing'])
  })

  it('describes exactly what a save will change', () => {
    const label = (id) => `addr-${id}`
    expect(describeAccessChanges(casey, accessDraft(casey), label)).toEqual([])
    expect(hasAccessChanges(casey, { ...accessDraft(casey), address_ids: [1, 3] })).toBe(false)
    expect(describeAccessChanges(casey, { role: 'Contributor', address_ids: [1, 2], enabled: false }, label)).toEqual([
      { kind: 'role', from: 'Agent', to: 'Contributor' },
      { kind: 'added', list: ['addr-2'] },
      { kind: 'removed', list: ['addr-3'] },
      { kind: 'disabled' }
    ])
  })

  it('shows that saving replaces a legacy role even when the primary role stays', () => {
    const legacy = { ...casey, roles: ['Agent', 'Billing'] }
    expect(describeAccessChanges(legacy, accessDraft(legacy))).toEqual([{ kind: 'role', from: 'Agent, Billing', to: 'Agent' }])
  })

  it('searches names, emails and roles', () => {
    const users = [casey, { id: 5, first_name: 'Robin', last_name: '', email: 'robin@example.test', roles: ['Contributor'] }]
    expect(filterUsers(users, '').length).toBe(2)
    expect(filterUsers(users, 'ROBIN').map((u) => u.id)).toEqual([5])
    expect(filterUsers(users, 'contrib').map((u) => u.id)).toEqual([5])
    expect(filterUsers(users, 'lee').map((u) => u.id)).toEqual([4])
  })

  it('makes initials', () => {
    expect(initials(casey)).toBe('CL')
    expect(initials({ first_name: 'robin' })).toBe('R')
    expect(initials({})).toBe('?')
  })
})
