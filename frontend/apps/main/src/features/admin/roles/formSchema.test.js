import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Agent',
    description: 'Handles conversations'
}

describe('Roles Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({ ...validForm, permissions: ['conversations:read'] })).not.toThrow()
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

    test('name too long', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(51) })).toThrow()
    })

    test('name at maximum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(50) })).not.toThrow()
    })

    test('description missing', () => {
        const { description, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('description too short', () => {
        expect(() => schema.parse({ ...validForm, description: 'a' })).toThrow()
    })

    test('description at minimum length', () => {
        expect(() => schema.parse({ ...validForm, description: 'ab' })).not.toThrow()
    })

    test('description too long', () => {
        expect(() => schema.parse({ ...validForm, description: 'a'.repeat(301) })).toThrow()
    })

    test('description at maximum length', () => {
        expect(() => schema.parse({ ...validForm, description: 'a'.repeat(300) })).not.toThrow()
    })

    test('permissions optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('permissions empty array accepted', () => {
        expect(() => schema.parse({ ...validForm, permissions: [] })).not.toThrow()
    })

    test('permissions must be an array of strings', () => {
        expect(() => schema.parse({ ...validForm, permissions: 'conversations:read' })).toThrow()
        expect(() => schema.parse({ ...validForm, permissions: [1] })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
