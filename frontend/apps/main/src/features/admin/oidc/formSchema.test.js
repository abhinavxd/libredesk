import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Google',
    provider_url: 'https://accounts.google.com',
    client_id: 'client-id',
    client_secret: 'client-secret'
}

describe('OIDC Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            disabled: false,
            provider: 'google',
            logo_url: 'https://cdn.example.com/logo.png',
            redirect_uri: 'https://desk.example.com/callback',
            enabled: false
        })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ ...validForm, name: '' })).toThrow()
    })

    test('name at minimum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'a' })).not.toThrow()
    })

    test('provider_url missing', () => {
        const { provider_url, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('provider_url not a url', () => {
        expect(() => schema.parse({ ...validForm, provider_url: 'accounts.google.com' })).toThrow()
    })

    test('provider_url empty string', () => {
        expect(() => schema.parse({ ...validForm, provider_url: '' })).toThrow()
    })

    test('client_id missing', () => {
        const { client_id, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('client_id empty string', () => {
        expect(() => schema.parse({ ...validForm, client_id: '' })).toThrow()
    })

    test('client_secret missing', () => {
        const { client_secret, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('client_secret empty string', () => {
        expect(() => schema.parse({ ...validForm, client_secret: '' })).toThrow()
    })

    test('logo_url empty string accepted', () => {
        expect(() => schema.parse({ ...validForm, logo_url: '' })).not.toThrow()
    })

    test('logo_url invalid url', () => {
        expect(() => schema.parse({ ...validForm, logo_url: 'logo.png' })).toThrow()
    })

    // Defaults
    test('enabled defaults to true', () => {
        expect(schema.parse(validForm).enabled).toBe(true)
    })

    test('disabled optional', () => {
        expect(() => schema.parse({ ...validForm, disabled: true })).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
