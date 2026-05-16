import { describe, test, expect } from 'vitest'
import { stripConvUUID, extractEmail, computeRecipientsFromMessage } from './email-recipients'

describe('stripConvUUID', () => {
    test('returns email unchanged when no plus addressing', () => {
        expect(stripConvUUID('support@domain.com')).toBe('support@domain.com')
    })

    test('strips valid UUID v4 from email', () => {
        expect(stripConvUUID('support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com'))
            .toBe('support@domain.com')
    })

    test('preserves non-UUID conv pattern (user email)', () => {
        expect(stripConvUUID('support+conv-21321@domain.com')).toBe('support+conv-21321@domain.com')
    })

    test('preserves invalid UUID format', () => {
        expect(stripConvUUID('support+conv-abc123-def456@domain.com')).toBe('support+conv-abc123-def456@domain.com')
    })

    test('keeps non-conv plus addressing unchanged', () => {
        expect(stripConvUUID('support+other@domain.com')).toBe('support+other@domain.com')
    })

    test('handles empty string', () => {
        expect(stripConvUUID('')).toBe('')
    })

    test('handles null/undefined', () => {
        expect(stripConvUUID(null)).toBe(null)
        expect(stripConvUUID(undefined)).toBe(undefined)
    })

    test('handles uppercase UUID v4', () => {
        expect(stripConvUUID('support+conv-13216CF7-6626-4B0D-A938-46CE65A20701@domain.com'))
            .toBe('support@domain.com')
    })

    test('preserves UUID missing the 4 version marker', () => {
        expect(stripConvUUID('support+conv-13216cf7-6626-ab0d-a938-46ce65a20701@domain.com'))
            .toBe('support+conv-13216cf7-6626-ab0d-a938-46ce65a20701@domain.com')
    })
})

describe('extractEmail', () => {
    test('returns bare email unchanged', () => {
        expect(extractEmail('support@domain.com')).toBe('support@domain.com')
    })

    test('strips display name format', () => {
        expect(extractEmail('Acme <support@mailer.acme.com>')).toBe('support@mailer.acme.com')
    })

    test('trims whitespace', () => {
        expect(extractEmail('  support@domain.com  ')).toBe('support@domain.com')
    })

    test('handles empty/null', () => {
        expect(extractEmail('')).toBe('')
        expect(extractEmail(null)).toBe(null)
        expect(extractEmail(undefined)).toBe(undefined)
    })
})

describe('computeRecipientsFromMessage', () => {
    const inboxAddresses = ['support@domain.com']
    const contactEmail = 'customer@example.com'

    describe('filters inbox email variants', () => {
        test('removes exact inbox email from recipients', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support@domain.com', 'other@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.cc).not.toContain('support@domain.com')
            expect(result.cc).toContain('other@domain.com')
        })

        test('removes UUID v4 plus-addressed variant', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com', 'other@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.cc).not.toContain('support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com')
            expect(result.cc).toContain('other@domain.com')
        })

        test('keeps non-UUID conv addresses (user email preserved)', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support+conv-21321@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.cc).toContain('support+conv-21321@domain.com')
        })

        test('keeps non-conv plus addresses', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support+other@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.cc).toContain('support+other@domain.com')
        })

        test('removes any address from a multi-inbox dedupe list', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support@domain.com', 'sales@domain.com', 'billing@domain.com', 'external@partner.com']
                }
            }
            const result = computeRecipientsFromMessage(
                message,
                contactEmail,
                ['support@domain.com', 'sales@domain.com', 'billing@domain.com']
            )
            expect(result.cc).not.toContain('support@domain.com')
            expect(result.cc).not.toContain('sales@domain.com')
            expect(result.cc).not.toContain('billing@domain.com')
            expect(result.cc).toContain('external@partner.com')
        })

        test('dedupes when inbox address is in display-name format', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support@mailer.acme.com', 'other@partner.com']
                }
            }
            const result = computeRecipientsFromMessage(
                message,
                contactEmail,
                ['Acme <support@mailer.acme.com>']
            )
            expect(result.cc).not.toContain('support@mailer.acme.com')
            expect(result.cc).toContain('other@partner.com')
        })

        test('dedupes when recipient is in display-name format', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['Support Team <support@domain.com>', 'other@partner.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, ['support@domain.com'])
            expect(result.cc).not.toContain('Support Team <support@domain.com>')
            expect(result.cc).toContain('other@partner.com')
        })

        test('accepts a single string for backwards compatibility', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support@domain.com', 'other@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, 'support@domain.com')
            expect(result.cc).not.toContain('support@domain.com')
            expect(result.cc).toContain('other@domain.com')
        })

        test('preserves all emails when dedupe list is empty', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support@domain.com', 'other@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, [])
            expect(result.cc).toContain('support@domain.com')
            expect(result.cc).toContain('other@domain.com')
        })

        test('preserves all emails when dedupe list is undefined', () => {
            const message = {
                type: 'incoming',
                meta: {
                    from: ['customer@example.com'],
                    to: ['support@domain.com', 'other@domain.com']
                }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, undefined)
            expect(result.cc).toContain('support@domain.com')
            expect(result.cc).toContain('other@domain.com')
        })
    })

    describe('incoming message handling', () => {
        test('sets from as to field for incoming', () => {
            const message = {
                type: 'incoming',
                meta: { from: ['sender@example.com'] }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.to).toEqual(['sender@example.com'])
        })

        test('falls back to contactEmail when no from', () => {
            const message = {
                type: 'incoming',
                meta: {}
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.to).toEqual([contactEmail])
        })
    })

    describe('outgoing message handling', () => {
        test('preserves to field for outgoing', () => {
            const message = {
                type: 'outgoing',
                meta: { to: ['recipient@example.com'] }
            }
            const result = computeRecipientsFromMessage(message, contactEmail, inboxAddresses)
            expect(result.to).toEqual(['recipient@example.com'])
        })
    })
})
