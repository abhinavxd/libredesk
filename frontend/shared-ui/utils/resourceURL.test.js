// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { avatarURL, localMediaURL, isPreviewImage, isPreviewAudio } from './resourceURL'
import { downloadUrl } from './file'

const id = '12345678-1234-4234-9234-123456789abc'
const path = '/uploads/' + id

describe('resource URLs', () => {
  it('keeps stored media and its signature on LibreDesk', () => {
    expect(localMediaURL(window.location.origin + path + '?sig=abc&exp=42')).toBe(path + '?sig=abc&exp=42')
    expect(downloadUrl(path + '?sig=abc&exp=42')).toBe(path + '?sig=abc&exp=42&download=1')
    expect(localMediaURL('/api/v1/conversations/' + id + '/messages/' + id + '/images/' + 'a'.repeat(64))).not.toBe('')
  })

  it.each([
    'https://tracker.example' + path,
    '//tracker.example' + path,
    '/\\tracker.example' + path,
    'javascript:alert(1)',
    'data:image/svg+xml,<svg/>',
    '/api/v1/redirect?url=https://tracker.example',
    '/uploads/invalid',
    '\n' + path,
    'https://user:pass@localhost' + path
  ])('rejects non-media resource %s', value => {
    expect(localMediaURL(value)).toBe('')
  })

  it('sends external avatars only to the local gateway', () => {
    const source = 'https://avatars.example/person?a=1&b=2'
    expect(avatarURL(source)).toBe('/api/v1/resource-images/avatar?source=' + encodeURIComponent(source))
    expect(avatarURL(path)).toBe(path)
    expect(avatarURL('blob:' + window.location.origin + '/preview')).toContain('blob:')
    expect(avatarURL('http://avatars.example/person')).toBe('')
    expect(avatarURL('https://user:pass@avatars.example/person')).toBe('')
  })

  it('excludes active images and audio playlists from previews', () => {
    expect(isPreviewImage('image/png')).toBe(true)
    expect(isPreviewImage('image/svg+xml')).toBe(false)
    expect(isPreviewAudio('audio/mpeg')).toBe(true)
    expect(isPreviewAudio('audio/x-mpegurl')).toBe(false)
  })
})
