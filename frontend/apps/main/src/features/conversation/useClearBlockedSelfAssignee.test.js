// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useUserStore } from '@main/stores/user'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { useClearBlockedSelfAssignee } from './useClearBlockedSelfAssignee'

vi.mock('@main/composables/useEmitter', () => ({ useEmitter: () => ({ emit: vi.fn() }) }))

let scopes
const setup = ({ blockSetting, status, agentId }) => {
  const userStore = useUserStore()
  userStore.user.id = 7
  userStore.user.availability_status = status
  useAppSettingsStore().setSettings({ 'app.block_assign_away_reassigning': blockSetting })
  const assignee = ref(agentId)
  const scope = effectScope()
  scopes.push(scope)
  scope.run(() => useClearBlockedSelfAssignee(() => assignee.value, () => (assignee.value = 'none')))
  return assignee
}

beforeEach(() => {
  scopes = []
  localStorage.clear()
  setActivePinia(createPinia())
})
afterEach(() => scopes.forEach((scope) => scope.stop()))

describe('clearing a blocked self assignee', () => {
  it('clears the current user when the setting blocks them', () => {
    expect(setup({ blockSetting: true, status: 'away_and_reassigning', agentId: '7' }).value).toBe('none')
  })

  it('keeps the current user when the setting is off or they are not reassigning', () => {
    expect(setup({ blockSetting: false, status: 'away_and_reassigning', agentId: '7' }).value).toBe('7')
    expect(setup({ blockSetting: true, status: 'away_manual', agentId: '7' }).value).toBe('7')
  })

  it('keeps another agent and an empty assignee', () => {
    expect(setup({ blockSetting: true, status: 'away_and_reassigning', agentId: '8' }).value).toBe('8')
    expect(setup({ blockSetting: true, status: 'away_and_reassigning', agentId: null }).value).toBe(null)
  })

  it('clears the current user again when a draft restores them', async () => {
    const assignee = setup({ blockSetting: true, status: 'away_and_reassigning', agentId: 'none' })
    assignee.value = '7'
    await nextTick()
    expect(assignee.value).toBe('none')
  })

  it('clears the current user once they switch to away and reassigning', async () => {
    const assignee = setup({ blockSetting: true, status: 'online', agentId: '7' })
    useUserStore().user.availability_status = 'away_and_reassigning'
    await nextTick()
    expect(assignee.value).toBe('none')
  })
})
