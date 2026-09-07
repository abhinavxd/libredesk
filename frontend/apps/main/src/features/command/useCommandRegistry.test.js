import { describe, it, expect, vi } from 'vitest'
import { computed, ref } from 'vue'

const userMock = vi.hoisted(() => ({ allowed: [] }))

vi.mock('@main/stores/user', () => ({
  useUserStore: () => ({
    can: (permission) => userMock.allowed.includes(permission)
  })
}))

import { useCommandRegistry, commandMatches } from '@main/features/command/useCommandRegistry'

const command = (id, overrides = {}) => ({ id, label: id, ...overrides })

describe('useCommandRegistry', () => {
  it('drops commands whose permission the user lacks', () => {
    userMock.allowed = ['a:read']
    const registry = useCommandRegistry([
      computed(() => [
        command('open'),
        command('allowed', { permission: 'a:read' }),
        command('denied', { permission: 'b:write' }),
        command('any-of', { permission: ['b:write', 'a:read'] })
      ])
    ])
    expect(registry.commands.value.map((c) => c.id)).toEqual(['open', 'allowed', 'any-of'])
  })

  it('hides children whose parent is hidden, even without their own permission', () => {
    userMock.allowed = []
    const registry = useCommandRegistry([
      computed(() => [
        command('group', { permission: 'x:manage', group: true }),
        command('child', { parent: 'group' }),
        command('orphan', { parent: 'missing' })
      ])
    ])
    expect(registry.commands.value).toEqual([])
    expect(registry.childrenOf('group')).toEqual([])
  })

  it('lists root commands and children separately', () => {
    userMock.allowed = []
    const registry = useCommandRegistry([
      computed(() => [command('root'), command('group', { group: true }), command('child', { parent: 'group' })])
    ])
    expect(registry.childrenOf(null).map((c) => c.id)).toEqual(['root', 'group'])
    expect(registry.childrenOf('group').map((c) => c.id)).toEqual(['child'])
    expect(registry.get('child').parent).toBe('group')
    expect(registry.get('nope')).toBeNull()
  })

  it('recomputes when a provider changes', () => {
    userMock.allowed = []
    const source = ref([command('one')])
    const registry = useCommandRegistry([computed(() => source.value)])
    expect(registry.commands.value).toHaveLength(1)
    source.value = [command('one'), command('two')]
    expect(registry.commands.value).toHaveLength(2)
  })
})

describe('commandMatches', () => {
  const resolve = command('resolve', { label: 'Resolve', keywords: ['close', 'done'] })

  it('matches label and keywords case-insensitively', () => {
    expect(commandMatches(resolve, 'RES')).toBe(true)
    expect(commandMatches(resolve, 'close')).toBe(true)
    expect(commandMatches(resolve, 'reopen')).toBe(false)
  })

  it('requires every typed word to match', () => {
    const agents = command('agents', { label: 'Agents', keywords: ['Teammates'] })
    expect(commandMatches(agents, 'team agent')).toBe(true)
    expect(commandMatches(agents, 'team inbox')).toBe(false)
    expect(commandMatches(agents, '   ')).toBe(true)
  })
})
