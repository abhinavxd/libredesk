import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Ops hook',
    url: 'https://hooks.example.com/libredesk',
    events: ['conversation.created']
}

describe('Webhooks Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            secret: 'shh',
            is_active: false,
            headers: '{"X-Token":"abc"}'
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

    test('url missing', () => {
        const { url, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('url not a url', () => {
        expect(() => schema.parse({ ...validForm, url: 'hooks.example.com' })).toThrow()
    })

    test('url empty string', () => {
        expect(() => schema.parse({ ...validForm, url: '' })).toThrow()
    })

    test('url with path accepted', () => {
        expect(() => schema.parse({ ...validForm, url: 'http://localhost:9000/hook' })).not.toThrow()
    })

    test('events missing', () => {
        const { events, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('events empty array', () => {
        expect(() => schema.parse({ ...validForm, events: [] })).toThrow()
    })

    test('events multiple values', () => {
        expect(() => schema.parse({ ...validForm, events: ['a', 'b'] })).not.toThrow()
    })

    test('events not an array', () => {
        expect(() => schema.parse({ ...validForm, events: 'a' })).toThrow()
    })

    // Optional fields
    test('is_active defaults to true', () => {
        expect(schema.parse(validForm).is_active).toBe(true)
    })

    test('secret and headers optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
