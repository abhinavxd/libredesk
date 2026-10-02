import { afterEach, beforeEach, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useReplyAwareness } from './useReplyAwareness'
let scope, args, awareness
const message = (uuid, sender_id, created_at, extra = {}) => ({
  uuid,
  sender_id,
  created_at,
  type: 'outgoing',
  ...extra
})
beforeEach(() => {
  args = {
    uuid: ref('A'),
    messageType: ref('reply'),
    hasDraft: ref(false),
    ready: ref(true),
    userID: ref(1),
    messages: ref([message('initial', 2, '2026-10-01T10:00:00Z')])
  }
  scope = effectScope()
  scope.run(() => {
    awareness = useReplyAwareness(args)
  })
})
afterEach(() => scope.stop())
it('warns about a new teammate reply while drafting and allows acknowledgement', async () => {
  args.hasDraft.value = true
  await nextTick()
  args.messages.value = [...args.messages.value, message('colleague', 2, '2026-10-01T10:01:00Z')]
  await nextTick()
  expect(awareness.newerReply.value).toBe(true)
  awareness.acknowledgeReply()
  expect(awareness.newerReply.value).toBe(false)
  args.messages.value = [...args.messages.value]
  await nextTick()
  expect(awareness.newerReply.value).toBe(false)
})
it('ignores own echoes, private notes and loaded history; resets on navigation', async () => {
  args.hasDraft.value = true
  await nextTick()
  args.messages.value = [
    ...args.messages.value,
    message('own', 1, '2026-10-01T10:01:00Z'),
    message('history', 2, '2026-10-01T09:00:00Z'),
    message('note', 2, '2026-10-01T10:02:00Z', { private: true })
  ]
  await nextTick()
  expect(awareness.newerReply.value).toBe(false)
  args.messages.value = [...args.messages.value, message('new', 2, '2026-10-01T10:03:00Z')]
  await nextTick()
  expect(awareness.newerReply.value).toBe(true)
  args.uuid.value = 'B'
  args.messages.value = [message('B-initial', 2, '2026-10-02T10:03:00Z')]
  await nextTick()
  expect(awareness.newerReply.value).toBe(false)
})
