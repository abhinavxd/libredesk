import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'CRM',
    url_template: 'https://crm.example.com/{{.Contact.Email}}'
}

describe('Context Links Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            secret: 's3cret',
            token_expiry_seconds: 60,
            is_active: false
        })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ ...validForm, name: '' })).toThrow()
    })

    test('name single character accepted', () => {
        expect(() => schema.parse({ ...validForm, name: 'a' })).not.toThrow()
    })

    test('url_template missing', () => {
        const { url_template, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('url_template empty string', () => {
        expect(() => schema.parse({ ...validForm, url_template: '' })).toThrow()
    })

    test('token_expiry_seconds defaults to 1200', () => {
        expect(schema.parse(validForm).token_expiry_seconds).toBe(1200)
    })

    test('token_expiry_seconds below minimum', () => {
        expect(() => schema.parse({ ...validForm, token_expiry_seconds: 0 })).toThrow()
    })

    test('token_expiry_seconds at minimum', () => {
        expect(() => schema.parse({ ...validForm, token_expiry_seconds: 1 })).not.toThrow()
    })

    test('token_expiry_seconds not an integer', () => {
        expect(() => schema.parse({ ...validForm, token_expiry_seconds: 1.5 })).toThrow()
    })

    test('token_expiry_seconds coerced from string', () => {
        expect(schema.parse({ ...validForm, token_expiry_seconds: '300' }).token_expiry_seconds).toBe(300)
    })

    test('secret optional', () => {
        expect(() => schema.parse({ ...validForm, secret: '' })).not.toThrow()
    })

    test('is_active defaults to true', () => {
        expect(schema.parse(validForm).is_active).toBe(true)
    })

    test('is_active must be boolean', () => {
        expect(() => schema.parse({ ...validForm, is_active: 'yes' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
