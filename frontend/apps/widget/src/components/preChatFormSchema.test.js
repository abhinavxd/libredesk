import { describe, test, expect } from 'vitest'
import { createPreChatFormSchema } from './preChatFormSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`

const field = (overrides) => ({ enabled: true, required: false, ...overrides })
const build = (...fields) => createPreChatFormSchema(mockT, fields)

describe('Pre Chat Form Schema', () => {
    test('no fields yields an empty schema', () => {
        expect(build().parse({})).toEqual({})
    })

    test('disabled fields are skipped', () => {
        const schema = build(field({ key: 'name', type: 'text', enabled: false, required: true }))
        expect(() => schema.parse({})).not.toThrow()
    })

    test('valid payload across field types', () => {
        const schema = build(
            field({ key: 'name', type: 'text', required: true }),
            field({ key: 'email', type: 'email', required: true }),
            field({ key: 'age', type: 'number' }),
            field({ key: 'agreed', type: 'checkbox' }),
            field({ key: 'dob', type: 'date' }),
            field({ key: 'site', type: 'link' }),
            field({ key: 'phone', type: 'phone' })
        )
        expect(() => schema.parse({
            name: 'John',
            email: 'john@example.com',
            age: 30,
            agreed: true,
            dob: '1990-01-01',
            site: 'https://example.com',
            phone: '5551234567',
            phone_country_code: 'US'
        })).not.toThrow()
    })
})

describe('Pre Chat Text Fields', () => {
    test('required text field rejects empty', () => {
        const schema = build(field({ key: 'name', type: 'text', required: true }))
        expect(() => schema.parse({ name: '' })).toThrow()
        expect(() => schema.parse({})).toThrow()
        expect(() => schema.parse({ name: 'J' })).not.toThrow()
    })

    test('optional text field may be omitted or empty', () => {
        const schema = build(field({ key: 'company', type: 'text' }))
        expect(() => schema.parse({})).not.toThrow()
        expect(() => schema.parse({ company: '' })).not.toThrow()
    })

    test('name field caps at 128 characters', () => {
        const schema = build(field({ key: 'name', type: 'text', required: true }))
        expect(() => schema.parse({ name: 'a'.repeat(128) })).not.toThrow()
        expect(() => schema.parse({ name: 'a'.repeat(129) })).toThrow()
    })

    test('other text fields cap at 1000 characters', () => {
        const schema = build(field({ key: 'notes', type: 'text', required: true }))
        expect(() => schema.parse({ notes: 'a'.repeat(1000) })).not.toThrow()
        expect(() => schema.parse({ notes: 'a'.repeat(1001) })).toThrow()
    })

    test('list fields behave like text fields', () => {
        const schema = build(field({ key: 'plan', type: 'list', required: true }))
        expect(() => schema.parse({ plan: '' })).toThrow()
        expect(() => schema.parse({ plan: 'pro' })).not.toThrow()
    })

    test('unknown types fall back to text', () => {
        const schema = build(field({ key: 'thing', type: 'weird', required: true }))
        expect(() => schema.parse({ thing: '' })).toThrow()
        expect(() => schema.parse({ thing: 'ok' })).not.toThrow()
    })
})

describe('Pre Chat Email Field', () => {
    test('rejects an invalid address', () => {
        const schema = build(field({ key: 'email', type: 'email', required: true }))
        expect(() => schema.parse({ email: 'john' })).toThrow()
        expect(() => schema.parse({ email: 'john@' })).toThrow()
        expect(() => schema.parse({ email: 'a@b.co' })).not.toThrow()
    })

    test('required rejects empty', () => {
        const schema = build(field({ key: 'email', type: 'email', required: true }))
        expect(() => schema.parse({ email: '' })).toThrow()
    })

    test('optional accepts empty or omitted', () => {
        const schema = build(field({ key: 'email', type: 'email' }))
        expect(() => schema.parse({ email: '' })).not.toThrow()
        expect(() => schema.parse({})).not.toThrow()
    })

    test('caps at 254 characters', () => {
        const schema = build(field({ key: 'email', type: 'email', required: true }))
        const local = 'a'.repeat(254 - '@example.com'.length)
        expect(() => schema.parse({ email: `${local}@example.com` })).not.toThrow()
        expect(() => schema.parse({ email: `${local}a@example.com` })).toThrow()
    })
})

describe('Pre Chat Number Field', () => {
    test('required rejects empty and non numbers', () => {
        const schema = build(field({ key: 'age', type: 'number', required: true }))
        expect(() => schema.parse({ age: '' })).toThrow()
        expect(() => schema.parse({ age: null })).toThrow()
        expect(() => schema.parse({})).toThrow()
        expect(() => schema.parse({ age: 'x' })).toThrow()
        expect(() => schema.parse({ age: 0 })).not.toThrow()
    })

    test('optional accepts empty and omitted', () => {
        const schema = build(field({ key: 'age', type: 'number' }))
        expect(() => schema.parse({ age: '' })).not.toThrow()
        expect(() => schema.parse({})).not.toThrow()
        expect(() => schema.parse({ age: 'x' })).toThrow()
    })
})

describe('Pre Chat Checkbox Field', () => {
    test('defaults to false and ignores the required flag', () => {
        const schema = build(field({ key: 'agreed', type: 'checkbox', required: true }))
        expect(schema.parse({}).agreed).toBe(false)
        expect(() => schema.parse({ agreed: true })).not.toThrow()
        expect(() => schema.parse({ agreed: 'yes' })).toThrow()
    })
})

describe('Pre Chat Date Field', () => {
    test('rejects a malformed date', () => {
        const schema = build(field({ key: 'dob', type: 'date', required: true }))
        expect(() => schema.parse({ dob: '01-01-1990' })).toThrow()
        expect(() => schema.parse({ dob: '1990-01-01' })).not.toThrow()
    })

    test('required rejects empty', () => {
        const schema = build(field({ key: 'dob', type: 'date', required: true }))
        expect(() => schema.parse({ dob: '' })).toThrow()
    })

    test('optional accepts empty and omitted', () => {
        const schema = build(field({ key: 'dob', type: 'date' }))
        expect(() => schema.parse({ dob: '' })).not.toThrow()
        expect(() => schema.parse({})).not.toThrow()
    })
})

describe('Pre Chat Link Field', () => {
    test('required rejects empty and non urls', () => {
        const schema = build(field({ key: 'site', type: 'link', required: true }))
        expect(() => schema.parse({ site: '' })).toThrow()
        expect(() => schema.parse({ site: 'example.com' })).toThrow()
        expect(() => schema.parse({ site: 'https://example.com' })).not.toThrow()
    })

    test('optional accepts empty and omitted but rejects a bad url', () => {
        const schema = build(field({ key: 'site', type: 'link' }))
        expect(() => schema.parse({ site: '' })).not.toThrow()
        expect(() => schema.parse({})).not.toThrow()
        expect(() => schema.parse({ site: 'example.com' })).toThrow()
    })

    test('caps at 1000 characters', () => {
        const schema = build(field({ key: 'site', type: 'link', required: true }))
        const base = 'https://example.com/'
        expect(() => schema.parse({ site: base + 'a'.repeat(1000 - base.length) })).not.toThrow()
        expect(() => schema.parse({ site: base + 'a'.repeat(1001 - base.length) })).toThrow()
    })
})

describe('Pre Chat Phone Field', () => {
    test('adds a country code companion field', () => {
        const schema = build(field({ key: 'phone', type: 'phone', required: true }))
        expect(Object.keys(schema.shape)).toEqual(['phone', 'phone_country_code'])
    })

    test('required rejects empty', () => {
        const schema = build(field({ key: 'phone', type: 'phone', required: true }))
        expect(() => schema.parse({ phone: '', phone_country_code: 'US' })).toThrow()
        expect(() => schema.parse({ phone: '5551234567', phone_country_code: '' })).toThrow()
        expect(() => schema.parse({ phone: '5551234567', phone_country_code: 'US' })).not.toThrow()
    })

    test('optional may be omitted', () => {
        const schema = build(field({ key: 'phone', type: 'phone' }))
        expect(() => schema.parse({})).not.toThrow()
        expect(() => schema.parse({ phone: '', phone_country_code: '' })).not.toThrow()
    })

    test('rejects a number without digits', () => {
        const schema = build(field({ key: 'phone', type: 'phone', required: true }))
        expect(() => schema.parse({ phone: '+()- ', phone_country_code: 'US' })).toThrow()
    })

    test('accepts a formatted number', () => {
        const schema = build(field({ key: 'phone', type: 'phone', required: true }))
        expect(() => schema.parse({ phone: '+1 (555) 123-4567', phone_country_code: 'US' })).not.toThrow()
    })

    test('caps the number at 20 characters', () => {
        const schema = build(field({ key: 'phone', type: 'phone', required: true }))
        expect(() => schema.parse({ phone: '1'.repeat(20), phone_country_code: 'US' })).not.toThrow()
        expect(() => schema.parse({ phone: '1'.repeat(21), phone_country_code: 'US' })).toThrow()
    })

    test('caps the country code at 10 characters', () => {
        const schema = build(field({ key: 'phone', type: 'phone', required: true }))
        expect(() => schema.parse({ phone: '5551234567', phone_country_code: 'a'.repeat(11) })).toThrow()
    })
})
