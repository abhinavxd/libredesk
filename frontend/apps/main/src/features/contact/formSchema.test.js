import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    first_name: 'John',
    email: 'john@example.com'
}

describe('Contact Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            enabled: true,
            last_name: 'Doe',
            phone_number: '+1 (555) 123-4567',
            phone_number_country_code: 'US',
            country: 'United States',
            avatar_url: 'https://cdn.example.com/a.png'
        })).not.toThrow()
    })

    test('first_name missing', () => {
        const { first_name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('first_name too short', () => {
        expect(() => schema.parse({ ...validForm, first_name: 'J' })).toThrow()
    })

    test('first_name at minimum length', () => {
        expect(() => schema.parse({ ...validForm, first_name: 'Jo' })).not.toThrow()
    })

    test('first_name too long', () => {
        expect(() => schema.parse({ ...validForm, first_name: 'a'.repeat(51) })).toThrow()
    })

    test('first_name at maximum length', () => {
        expect(() => schema.parse({ ...validForm, first_name: 'a'.repeat(50) })).not.toThrow()
    })

    test('email missing', () => {
        const { email, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('email invalid format', () => {
        expect(() => schema.parse({ ...validForm, email: 'john' })).toThrow()
        expect(() => schema.parse({ ...validForm, email: 'john@' })).toThrow()
        expect(() => schema.parse({ ...validForm, email: '' })).toThrow()
    })

    test('email short valid address', () => {
        expect(() => schema.parse({ ...validForm, email: 'a@b.co' })).not.toThrow()
    })

    test('phone_number optional and nullable', () => {
        expect(() => schema.parse({ ...validForm, phone_number: null })).not.toThrow()
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('phone_number empty string accepted', () => {
        expect(() => schema.parse({ ...validForm, phone_number: '' })).not.toThrow()
    })

    test('phone_number without digits rejected', () => {
        expect(() => schema.parse({ ...validForm, phone_number: 'abcd' })).toThrow()
    })

    test('phone_number too long', () => {
        expect(() => schema.parse({ ...validForm, phone_number: '1'.repeat(21) })).toThrow()
    })

    test('phone_number at maximum length', () => {
        expect(() => schema.parse({ ...validForm, phone_number: '1'.repeat(20) })).not.toThrow()
    })

    test('last_name optional', () => {
        expect(() => schema.parse({ ...validForm, last_name: '' })).not.toThrow()
    })

    test('country and avatar_url accept null', () => {
        expect(() => schema.parse({ ...validForm, country: null, avatar_url: null })).not.toThrow()
    })

    test('enabled must be boolean', () => {
        expect(() => schema.parse({ ...validForm, enabled: 'yes' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
