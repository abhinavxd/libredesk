import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Snoozed',
    category: 'waiting'
}

describe('Status Form Schema', () => {
    test('valid form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
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
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(26) })).toThrow()
    })

    test('name at maximum length', () => {
        expect(() => schema.parse({ ...validForm, name: 'a'.repeat(25) })).not.toThrow()
    })

    test('name not a string', () => {
        expect(() => schema.parse({ ...validForm, name: 42 })).toThrow()
    })

    test('category missing', () => {
        const { category, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('category invalid value', () => {
        expect(() => schema.parse({ ...validForm, category: 'closed' })).toThrow()
    })

    test('all category values accepted', () => {
        for (const category of ['open', 'waiting', 'resolved']) {
            expect(() => schema.parse({ ...validForm, category })).not.toThrow()
        }
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
