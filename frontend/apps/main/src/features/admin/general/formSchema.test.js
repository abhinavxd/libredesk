import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    site_name: 'Support',
    root_url: 'https://support.example.com',
    favicon_url: 'https://support.example.com/favicon.ico',
    max_file_upload_size: 20
}

describe('General Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            lang: 'en',
            timezone: 'Asia/Kolkata',
            business_hours_id: '1',
            logo_url: 'https://support.example.com/logo.png',
            allowed_file_upload_extensions: ['png', 'pdf'],
            show_conversation_subject: true
        })).not.toThrow()
    })

    test('site_name missing', () => {
        const { site_name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('site_name empty string', () => {
        expect(() => schema.parse({ ...validForm, site_name: '' })).toThrow()
    })

    test('site_name single character accepted', () => {
        expect(() => schema.parse({ ...validForm, site_name: 'a' })).not.toThrow()
    })

    test('root_url missing', () => {
        const { root_url, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('root_url not a url', () => {
        expect(() => schema.parse({ ...validForm, root_url: 'support.example.com' })).toThrow()
    })

    test('root_url empty string', () => {
        expect(() => schema.parse({ ...validForm, root_url: '' })).toThrow()
    })

    test('root_url with port and path accepted', () => {
        expect(() => schema.parse({ ...validForm, root_url: 'http://localhost:9000/desk' })).not.toThrow()
    })

    test('favicon_url missing', () => {
        const { favicon_url, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('favicon_url not a url', () => {
        expect(() => schema.parse({ ...validForm, favicon_url: '/favicon.ico' })).toThrow()
    })

    test('logo_url optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('logo_url empty string accepted', () => {
        expect(() => schema.parse({ ...validForm, logo_url: '' })).not.toThrow()
    })

    test('logo_url invalid url', () => {
        expect(() => schema.parse({ ...validForm, logo_url: 'not-a-url' })).toThrow()
    })

    test('max_file_upload_size missing', () => {
        const { max_file_upload_size, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('max_file_upload_size below minimum', () => {
        expect(() => schema.parse({ ...validForm, max_file_upload_size: 0 })).toThrow()
    })

    test('max_file_upload_size at minimum', () => {
        expect(() => schema.parse({ ...validForm, max_file_upload_size: 1 })).not.toThrow()
    })

    test('max_file_upload_size above maximum', () => {
        expect(() => schema.parse({ ...validForm, max_file_upload_size: 501 })).toThrow()
    })

    test('max_file_upload_size at maximum', () => {
        expect(() => schema.parse({ ...validForm, max_file_upload_size: 500 })).not.toThrow()
    })

    test('max_file_upload_size as string', () => {
        expect(() => schema.parse({ ...validForm, max_file_upload_size: '20' })).toThrow()
    })

    // Optional fields
    test('allowed_file_upload_extensions defaults to empty array', () => {
        expect(schema.parse(validForm).allowed_file_upload_extensions).toEqual([])
    })

    test('allowed_file_upload_extensions accepts null', () => {
        expect(() => schema.parse({ ...validForm, allowed_file_upload_extensions: null })).not.toThrow()
    })

    test('show_conversation_subject must be boolean', () => {
        expect(() => schema.parse({ ...validForm, show_conversation_subject: 'true' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
