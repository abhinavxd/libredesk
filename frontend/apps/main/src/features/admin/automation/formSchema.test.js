import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Auto assign',
    type: 'new_conversation'
}

describe('Automation Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            name: 'Auto assign',
            description: 'Assigns new conversations',
            enabled: false,
            type: 'conversation_update',
            events: ['status_change']
        })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name null', () => {
        expect(() => schema.parse({ ...validForm, name: null })).toThrow()
    })

    test('name not a string', () => {
        expect(() => schema.parse({ ...validForm, name: 123 })).toThrow()
    })

    test('type missing', () => {
        const { type, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('type not a string', () => {
        expect(() => schema.parse({ ...validForm, type: 1 })).toThrow()
    })

    test('conversation_update without events', () => {
        expect(() => schema.parse({ ...validForm, type: 'conversation_update' })).toThrow()
    })

    test('conversation_update with empty events', () => {
        expect(() => schema.parse({ ...validForm, type: 'conversation_update', events: [] })).toThrow()
    })

    test('conversation_update with one event', () => {
        expect(() => schema.parse({ ...validForm, type: 'conversation_update', events: ['status_change'] })).not.toThrow()
    })

    test('events optional for other types', () => {
        expect(() => schema.parse({ ...validForm, type: 'new_conversation' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, type: 'new_conversation', events: [] })).not.toThrow()
    })

    test('events not an array', () => {
        expect(() => schema.parse({ ...validForm, events: 'status_change' })).toThrow()
    })

    test('description defaults to empty string', () => {
        expect(schema.parse(validForm).description).toBe('')
    })

    test('enabled defaults to true', () => {
        expect(schema.parse(validForm).enabled).toBe(true)
    })

    test('enabled must be boolean', () => {
        expect(() => schema.parse({ ...validForm, enabled: 'true' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
