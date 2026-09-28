// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import AudioReplyRecorder from './AudioReplyRecorder.vue'
import { encodeAudioReply } from './audioReply'

vi.mock('./audioReply', () => ({ encodeAudioReply: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@shared-ui/components/ui/tooltip', () => ({
  Tooltip: {
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.())
  },
  TooltipTrigger: {
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.())
  },
  TooltipContent: { setup: () => () => null }
}))
vi.mock('@shared-ui/components/ui/button', () => ({
  Button: {
    setup:
      (_, { slots }) =>
      () =>
        h('button', slots.default?.())
  }
}))
vi.mock('@shared-ui/components/ui/popover', () => ({
  Popover: {
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.())
  },
  PopoverTrigger: {
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.())
  },
  PopoverContent: {
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.())
  }
}))
let app, root, track, getUserMedia, recorder, recorded, busy
const click = async (label) => {
  const button = [...root.querySelectorAll('button')].find(
    (el) => el.getAttribute('aria-label') === label || el.textContent === label
  )
  expect(button).toBeTruthy()
  button.click()
  await nextTick()
}
beforeEach(() => {
  track = { stop: vi.fn() }
  getUserMedia = vi.fn().mockResolvedValue({ getTracks: () => [track] })
  vi.stubGlobal('navigator', { mediaDevices: { getUserMedia } })
  vi.stubGlobal(
    'MediaRecorder',
    class {
      static isTypeSupported = () => true
      mimeType = 'audio/webm'
      state = 'inactive'
      constructor() {
        recorder = this
      }
      start() {
        this.state = 'recording'
      }
      stop() {
        this.state = 'inactive'
        this.ondataavailable?.({ data: new Blob(['sound']) })
        this.onstop?.()
      }
    }
  )
  vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:recording'), revokeObjectURL: vi.fn() })
  encodeAudioReply.mockResolvedValue(new File(['mp3'], 'recording.mp3', { type: 'audio/mpeg' }))
  recorded = vi.fn()
  busy = vi.fn()
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(AudioReplyRecorder, { onRecorded: recorded, onBusy: busy })
  app.config.globalProperties.$t = (key) => key
  app.mount(root)
})
afterEach(() => {
  app.unmount()
  root.remove()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

it('records, releases the microphone, previews, and attaches the encoded file', async () => {
  await click('globals.messages.recordAudio')
  await vi.waitFor(() => expect(recorder?.state).toBe('recording'))
  await click('globals.messages.stopRecording')
  await vi.waitFor(() => expect(root.querySelector('audio')).toBeTruthy())
  expect(track.stop).toHaveBeenCalledOnce()
  await click('globals.messages.attachRecording')
  expect(recorded).toHaveBeenCalledWith([expect.objectContaining({ name: 'recording.mp3' })])
  expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:recording')
  expect(busy).toHaveBeenLastCalledWith(false)
})
it('stops recording without attaching when cancelled', async () => {
  await click('globals.messages.recordAudio')
  await vi.waitFor(() => expect(recorder?.state).toBe('recording'))
  await click('globals.messages.cancel')
  expect(track.stop).toHaveBeenCalledOnce()
  expect(recorded).not.toHaveBeenCalled()
  expect(encodeAudioReply).not.toHaveBeenCalled()
})
it('releases a microphone granted after cancellation', async () => {
  let grant
  getUserMedia.mockImplementation(
    () =>
      new Promise((resolve) => {
        grant = resolve
      })
  )
  await click('globals.messages.recordAudio')
  await click('globals.messages.cancel')
  grant({ getTracks: () => [track] })
  await vi.waitFor(() => expect(track.stop).toHaveBeenCalledOnce())
  expect(recorded).not.toHaveBeenCalled()
})
it('shows a useful error when microphone permission is denied', async () => {
  getUserMedia.mockRejectedValue(new Error('denied'))
  await click('globals.messages.recordAudio')
  await vi.waitFor(() => expect(root.textContent).toContain('replyBox.microphoneUnavailable'))
  await click('globals.messages.cancel')
  expect(busy).toHaveBeenLastCalledWith(false)
})
it('discards an encoding result that finishes after cancellation', async () => {
  let finish
  encodeAudioReply.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve
      })
  )
  await click('globals.messages.recordAudio')
  await vi.waitFor(() => expect(recorder?.state).toBe('recording'))
  await click('globals.messages.stopRecording')
  await click('globals.messages.cancel')
  finish(new File(['mp3'], 'late.mp3'))
  await nextTick()
  expect(URL.createObjectURL).not.toHaveBeenCalled()
  expect(recorded).not.toHaveBeenCalled()
})
it('releases microphone resources when the conversation unmounts', async () => {
  await click('globals.messages.recordAudio')
  await vi.waitFor(() => expect(recorder?.state).toBe('recording'))
  app.unmount()
  expect(track.stop).toHaveBeenCalledOnce()
  expect(recorded).not.toHaveBeenCalled()
})
it('reports encoding failures and allows cancellation', async () => {
  encodeAudioReply.mockRejectedValueOnce(new Error('bad encoding'))
  await click('globals.messages.recordAudio')
  await vi.waitFor(() => expect(recorder?.state).toBe('recording'))
  await click('globals.messages.stopRecording')
  await vi.waitFor(() => expect(root.textContent).toContain('replyBox.audioEncodingFailed'))
  await click('globals.messages.cancel')
  expect(busy).toHaveBeenLastCalledWith(false)
})
