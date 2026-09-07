import { describe, test, expect } from 'vitest'
import { createTeamFormSchema } from './teamFormSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createTeamFormSchema(mockT)

const validForm = {
    name: 'Billing',
    emoji: '💰',
    conversation_assignment_type: 'round_robin',
    timezone: 'Asia/Kolkata'
}

describe('Team Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            max_auto_assigned_conversations: 10,
            business_hours_id: 2,
            sla_policy_id: 3
        })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name too short', () => {
        expect(() => schema.parse({ ...validForm, name: 'a' })).toThrow()
    })

    test('name at minimum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'ab' })).not.toThrow()
    })

    test('emoji missing', () => {
        const { emoji, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('emoji not a string', () => {
        expect(() => schema.parse({ ...validForm, emoji: 1 })).toThrow()
    })

    test('conversation_assignment_type missing', () => {
        const { conversation_assignment_type, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('timezone missing', () => {
        const { timezone, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    // Max auto assigned conversations
    test('max_auto_assigned_conversations defaults to 0', () => {
        expect(schema.parse(validForm).max_auto_assigned_conversations).toBe(0)
    })

    test('max_auto_assigned_conversations coerced from string', () => {
        expect(schema.parse({ ...validForm, max_auto_assigned_conversations: '5' }).max_auto_assigned_conversations).toBe(5)
    })

    test('max_auto_assigned_conversations rejects non numeric string', () => {
        expect(() => schema.parse({ ...validForm, max_auto_assigned_conversations: 'abc' })).toThrow()
    })

    test('business_hours_id accepts null', () => {
        expect(() => schema.parse({ ...validForm, business_hours_id: null })).not.toThrow()
    })

    test('sla_policy_id accepts null', () => {
        expect(() => schema.parse({ ...validForm, sla_policy_id: null })).not.toThrow()
    })

    test('sla_policy_id must be a number', () => {
        expect(() => schema.parse({ ...validForm, sla_policy_id: '3' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
