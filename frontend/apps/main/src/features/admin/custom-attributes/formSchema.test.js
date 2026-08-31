import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    applies_to: 'contact',
    name: 'Account tier',
    key: 'account_tier',
    description: 'Tier of the customer account',
    data_type: 'text'
}

describe('Custom Attributes Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid list form', () => {
        expect(() => schema.parse({ ...validForm, data_type: 'list', values: ['gold'] })).not.toThrow()
    })

    // applies_to
    test('applies_to missing', () => {
        const { applies_to, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('applies_to invalid value', () => {
        expect(() => schema.parse({ ...validForm, applies_to: 'team' })).toThrow()
    })

    test('applies_to conversation accepted', () => {
        expect(() => schema.parse({ ...validForm, applies_to: 'conversation' })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name too short', () => {
        expect(() => schema.parse({ ...validForm, name: 'ab' })).toThrow()
    })

    test('name at minimum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'abc' })).not.toThrow()
    })

    test('name too long', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(141) })).toThrow()
    })

    test('name at maximum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(140) })).not.toThrow()
    })

    test('key missing', () => {
        const { key, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('key too short', () => {
        expect(() => schema.parse({ ...validForm, key: 'ab' })).toThrow()
    })

    test('key at minimum length', () => {
        expect(() => schema.parse({ ...validForm, key: 'abc' })).not.toThrow()
    })

    test('key too long', () => {
        expect(() => schema.parse({ ...validForm, key: 'a'.repeat(141) })).toThrow()
    })

    test('key with uppercase rejected', () => {
        expect(() => schema.parse({ ...validForm, key: 'Account_tier' })).toThrow()
    })

    test('key with dash rejected', () => {
        expect(() => schema.parse({ ...validForm, key: 'account-tier' })).toThrow()
    })

    test('key with space rejected', () => {
        expect(() => schema.parse({ ...validForm, key: 'account tier' })).toThrow()
    })

    test('key with digits and underscores accepted', () => {
        expect(() => schema.parse({ ...validForm, key: 'account_tier_2' })).not.toThrow()
    })

    test('description missing', () => {
        const { description, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('description too short', () => {
        expect(() => schema.parse({ ...validForm, description: 'ab' })).toThrow()
    })

    test('description at minimum length', () => {
        expect(() => schema.parse({ ...validForm, description: 'abc' })).not.toThrow()
    })

    test('description too long', () => {
        expect(() => schema.parse({ ...validForm, description: 'a'.repeat(301) })).toThrow()
    })

    test('description at maximum length', () => {
        expect(() => schema.parse({ ...validForm, description: 'a'.repeat(300) })).not.toThrow()
    })

    // data_type
    test('data_type missing', () => {
        const { data_type, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('data_type invalid value', () => {
        expect(() => schema.parse({ ...validForm, data_type: 'json' })).toThrow()
    })

    test('all data_type values accepted', () => {
        for (const dt of ['text', 'number', 'checkbox', 'date', 'link']) {
            expect(() => schema.parse({ ...validForm, data_type: dt })).not.toThrow()
        }
    })

    test('values required for list type', () => {
        expect(() => schema.parse({ ...validForm, data_type: 'list' })).toThrow()
    })

    test('values empty for list type', () => {
        expect(() => schema.parse({ ...validForm, data_type: 'list', values: [] })).toThrow()
    })

    test('values defaults to empty array', () => {
        expect(schema.parse(validForm).values).toEqual([])
    })

    test('id optional', () => {
        expect(() => schema.parse({ ...validForm, id: 3 })).not.toThrow()
    })

    test('id must be a number', () => {
        expect(() => schema.parse({ ...validForm, id: '3' })).toThrow()
    })

    test('regex and regex_hint optional', () => {
        expect(() => schema.parse({ ...validForm, regex: '^a+$', regex_hint: 'letters' })).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
