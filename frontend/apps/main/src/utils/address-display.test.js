import { describe, expect, it } from 'vitest'
import { addressLabel } from './address-display'

describe('addressLabel', () => {
  it('prefers a configured display name', () => {
    expect(
      addressLabel({
        address: 'contact@antimuonstudios.com',
        display_name: '  General Enquiries  '
      })
    ).toBe('General Enquiries')
  })

  it('uses a readable email local part when no display name exists', () => {
    expect(addressLabel({ address: 'director@antimuonstudios.com' })).toBe('Director')
    expect(addressLabel({ address: 'press.office@antimuonstudios.com' })).toBe('Press Office')
  })

  it('does not fail on an incomplete address', () => {
    expect(addressLabel({ address: '' })).toBe('')
  })
})
