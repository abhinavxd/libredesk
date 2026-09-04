import { describe, test, expect } from 'vitest'
import { createCollectionFormSchema } from './collectionFormSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createCollectionFormSchema(mockT)

const validForm = {
    name: 'Getting started'
}

describe('Help Center Collection Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            name: 'Getting started',
            description: 'Basics',
            icon: 'book',
            locale: 'fr',
            parent_id: 4,
            is_published: false,
            sort_order: 2
        })).not.toThrow()
    })

    test('name missing', () => {
        expect(() => schema.parse({})).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ name: '' })).toThrow()
    })

    test('name at minimum length', () => {
        expect(() => schema.parse({ name: 'a' })).not.toThrow()
    })

    test('name not a string', () => {
        expect(() => schema.parse({ name: 1 })).toThrow()
    })

    test('locale defaults to en', () => {
        expect(schema.parse(validForm).locale).toBe('en')
    })

    test('is_published defaults to true', () => {
        expect(schema.parse(validForm).is_published).toBe(true)
    })

    test('sort_order defaults to 0', () => {
        expect(schema.parse(validForm).sort_order).toBe(0)
    })

    test('sort_order must be a number', () => {
        expect(() => schema.parse({ ...validForm, sort_order: '2' })).toThrow()
    })

    test('parent_id accepts null', () => {
        expect(() => schema.parse({ ...validForm, parent_id: null })).not.toThrow()
    })

    test('parent_id coerced from string', () => {
        expect(schema.parse({ ...validForm, parent_id: '4' }).parent_id).toBe(4)
    })

    test('parent_id rejects non numeric string', () => {
        expect(() => schema.parse({ ...validForm, parent_id: 'abc' })).toThrow()
    })

    test('description and icon optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })
})
