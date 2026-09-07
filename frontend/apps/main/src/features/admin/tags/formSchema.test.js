import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'billing'
}

describe('Tags Form Schema', () => {
    test('valid form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('name missing', () => {
        expect(() => schema.parse({})).toThrow()
    })

    test('name null', () => {
        expect(() => schema.parse({ name: null })).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ name: '' })).toThrow()
    })

    test('name too short', () => {
        expect(() => schema.parse({ name: 'ab' })).toThrow()
    })

    test('name at minimum length', () => {
        expect(() => schema.parse({ name: 'abc' })).not.toThrow()
    })

    test('name not a string', () => {
        expect(() => schema.parse({ name: 123 })).toThrow()
    })

    test('extra unknown fields ignored', () => {
        expect(() => schema.parse({ ...validForm, id: 1 })).not.toThrow()
    })
})
