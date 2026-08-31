import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useMacroStore } from './macro'
import { useUserStore } from './user'

vi.mock('../composables/useEmitter', () => ({
  useEmitter: () => ({ emit: vi.fn() })
}))

describe('macro store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('restores filtered actions when permissions change', () => {
    const macroStore = useMacroStore()
    const userStore = useUserStore()
    const setPermissions = (permissions) => {
      userStore.setCurrentUser({ id: 1, teams: [], permissions })
    }

    setPermissions(['messages:write'])
    macroStore.macroList = [
      {
        id: 1,
        name: 'Reply and note',
        visibility: 'all',
        message_content: '',
        actions: [{ type: 'send_reply' }, { type: 'send_private_note' }]
      }
    ]

    expect(macroStore.macroOptions[0].actions).toEqual([{ type: 'send_reply' }])
    expect(macroStore.macroList[0].actions).toHaveLength(2)

    setPermissions(['messages:write', 'messages:write_private'])

    expect(macroStore.macroOptions[0].actions).toEqual([
      { type: 'send_reply' },
      { type: 'send_private_note' }
    ])
  })
})
