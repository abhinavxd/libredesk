import { describe, test, expect } from 'vitest'
import { createArticleFormSchema } from './articleFormSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createArticleFormSchema(mockT)

const validForm = {
    title: 'How to reset your password',
    content: '<p>Click the reset link.</p>',
    collection_id: 1
}

describe('Help Center Article Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            status: 'published',
            sort_order: 3,
            ai_enabled: true,
            locale: 'de',
            author_id: '7',
            excerpt: 'Reset flow',
            meta_title: 'Reset',
            meta_description: 'How to reset',
            meta_image_url: 'https://cdn.example.com/a.png'
        })).not.toThrow()
    })

    test('title missing', () => {
        const { title, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('title empty string', () => {
        expect(() => schema.parse({ ...validForm, title: '' })).toThrow()
    })

    test('title at minimum length', () => {
        expect(() => schema.parse({ ...validForm, title: 'a' })).not.toThrow()
    })

    test('content missing', () => {
        const { content, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('content empty string', () => {
        expect(() => schema.parse({ ...validForm, content: '' })).toThrow()
    })

    test('untouched editor markup rejected', () => {
        expect(() => schema.parse({ ...validForm, content: '<p></p>' })).toThrow()
    })

    test('whitespace only content rejected', () => {
        expect(() => schema.parse({ ...validForm, content: '<p>   </p>' })).toThrow()
        expect(() => schema.parse({ ...validForm, content: '<p>&nbsp;</p>' })).toThrow()
    })

    test('embed only content accepted', () => {
        expect(() => schema.parse({ ...validForm, content: '<p><img src="a.png"></p>' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, content: '<table><tr><td></td></tr></table>' })).not.toThrow()
    })

    test('collection_id missing', () => {
        const { collection_id, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('collection_id below minimum', () => {
        expect(() => schema.parse({ ...validForm, collection_id: 0 })).toThrow()
    })

    test('collection_id coerced from string', () => {
        expect(schema.parse({ ...validForm, collection_id: '2' }).collection_id).toBe(2)
    })

    test('collection_id rejects non numeric string', () => {
        expect(() => schema.parse({ ...validForm, collection_id: 'abc' })).toThrow()
    })

    test('status invalid value', () => {
        expect(() => schema.parse({ ...validForm, status: 'archived' })).toThrow()
    })

    test('status defaults to draft', () => {
        expect(schema.parse(validForm).status).toBe('draft')
    })

    test('sort_order defaults to 0', () => {
        expect(schema.parse(validForm).sort_order).toBe(0)
    })

    test('ai_enabled defaults to false', () => {
        expect(schema.parse(validForm).ai_enabled).toBe(false)
    })

    test('locale defaults to en', () => {
        expect(schema.parse(validForm).locale).toBe('en')
    })

    test('meta fields default to empty strings', () => {
        const parsed = schema.parse(validForm)
        expect(parsed.excerpt).toBe('')
        expect(parsed.meta_title).toBe('')
        expect(parsed.meta_description).toBe('')
        expect(parsed.meta_image_url).toBe('')
    })

    test('author_id optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
