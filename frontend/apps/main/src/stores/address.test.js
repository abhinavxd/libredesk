import { describe, expect, it } from 'vitest'
import { orderAddresses } from './address'

describe('orderAddresses', () => {
  it('lists mailbox addresses before aliases while preserving alphabetical order within each group', () => {
    const addresses = orderAddresses([
      { id: 1, kind: 'alias', address: 'billing@example.test' },
      { id: 2, kind: 'mailbox', address: 'support@example.test' },
      { id: 3, kind: 'alias', address: 'contact@example.test' },
      { id: 4, kind: 'mailbox', address: 'director@example.test' }
    ])

    expect(addresses.map((address) => address.address)).toEqual([
      'director@example.test',
      'support@example.test',
      'billing@example.test',
      'contact@example.test'
    ])
  })
})
