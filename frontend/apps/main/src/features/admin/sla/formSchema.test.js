// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Priority SLA',
    first_response_time: '30m'
}

const validNotification = {
    type: 'breach',
    time_delay_type: 'immediately',
    metric: 'first_response',
    recipients: ['assigned_user']
}

describe('SLA Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            name: 'Priority SLA',
            description: 'For paying customers',
            first_response_time: '30m',
            resolution_time: '4h',
            next_response_time: '1h',
            notifications: [validNotification]
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

    test('name too long', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(256) })).toThrow()
    })

    test('name at maximum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(255) })).not.toThrow()
    })

    test('description too long', () => {
        expect(() => schema.parse({ ...validForm, description: 'a'.repeat(256) })).toThrow()
    })

    test('description at maximum length', () => {
        expect(() => schema.parse({ ...validForm, description: 'a'.repeat(255) })).not.toThrow()
    })

    test('description null becomes empty string', () => {
        expect(schema.parse({ ...validForm, description: null }).description).toBe('')
    })

    test('at least one sla time required', () => {
        expect(() => schema.parse({ name: 'Priority SLA' })).toThrow()
    })

    test('resolution_time alone is enough', () => {
        expect(() => schema.parse({ name: 'Priority SLA', resolution_time: '4h' })).not.toThrow()
    })

    test('next_response_time alone is enough', () => {
        expect(() => schema.parse({ name: 'Priority SLA', next_response_time: '90m' })).not.toThrow()
    })

    test('invalid duration format', () => {
        expect(() => schema.parse({ ...validForm, first_response_time: '30s' })).toThrow()
        expect(() => schema.parse({ ...validForm, first_response_time: '1h30m' })).toThrow()
        expect(() => schema.parse({ ...validForm, first_response_time: '30' })).toThrow()
    })

    test('notifications defaults to empty array', () => {
        expect(schema.parse(validForm).notifications).toEqual([])
    })

    test('notification type invalid', () => {
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, type: 'reminder' }]
        })).toThrow()
    })

    test('notification metric invalid', () => {
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, metric: 'anything' }]
        })).toThrow()
    })

    test('all notification metrics accepted', () => {
        for (const metric of ['first_response', 'resolution', 'next_response', 'all']) {
            expect(() => schema.parse({ ...validForm, notifications: [{ ...validNotification, metric }] })).not.toThrow()
        }
    })

    test('notification recipients empty', () => {
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, recipients: [] }]
        })).toThrow()
    })

    test('notification recipients missing', () => {
        const { recipients, ...notification } = validNotification
        expect(() => schema.parse({ ...validForm, notifications: [notification] })).toThrow()
    })

    test('time_delay required when not immediate', () => {
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, time_delay_type: 'after' }]
        })).toThrow()
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, time_delay_type: 'before', time_delay: '' }]
        })).toThrow()
    })

    test('time_delay must be a valid duration when not immediate', () => {
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, time_delay_type: 'after', time_delay: '30s' }]
        })).toThrow()
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, time_delay_type: 'after', time_delay: '30m' }]
        })).not.toThrow()
    })

    test('time_delay ignored when immediate', () => {
        expect(() => schema.parse({
            ...validForm,
            notifications: [{ ...validNotification, time_delay_type: 'immediately', time_delay: '' }]
        })).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
