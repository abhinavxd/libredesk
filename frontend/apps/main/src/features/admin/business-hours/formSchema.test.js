import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Weekdays',
    is_always_open: false,
    hours: {
        monday: { open: '09:00', close: '17:30' }
    }
}

describe('Business Hours Form Schema', () => {
    test('valid complete form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid always open form without hours', () => {
        expect(() => schema.parse({ name: 'Always', is_always_open: true })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ ...validForm, name: '' })).toThrow()
    })

    test('name single character accepted', () => {
        expect(() => schema.parse({ ...validForm, name: 'a' })).not.toThrow()
    })

    // is_always_open
    test('is_always_open missing', () => {
        const { is_always_open, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('is_always_open not a boolean', () => {
        expect(() => schema.parse({ ...validForm, is_always_open: 'false' })).toThrow()
    })

    test('hours required when not always open', () => {
        expect(() => schema.parse({ name: 'Weekdays', is_always_open: false })).toThrow()
    })

    test('hours empty object when not always open', () => {
        expect(() => schema.parse({ ...validForm, hours: {} })).toThrow()
    })

    test('hours with multiple days', () => {
        expect(() => schema.parse({
            ...validForm,
            hours: {
                monday: { open: '09:00', close: '17:00' },
                tuesday: { open: '10:15', close: '18:45' }
            }
        })).not.toThrow()
    })

    test('open time invalid format', () => {
        expect(() => schema.parse({ ...validForm, hours: { monday: { open: '9:00', close: '17:00' } } })).toThrow()
    })

    test('close time invalid hour', () => {
        expect(() => schema.parse({ ...validForm, hours: { monday: { open: '09:00', close: '24:00' } } })).toThrow()
    })

    test('close time invalid minute', () => {
        expect(() => schema.parse({ ...validForm, hours: { monday: { open: '09:00', close: '17:60' } } })).toThrow()
    })

    test('boundary times accepted', () => {
        expect(() => schema.parse({ ...validForm, hours: { monday: { open: '00:00', close: '23:59' } } })).not.toThrow()
    })

    test('close time missing', () => {
        expect(() => schema.parse({ ...validForm, hours: { monday: { open: '09:00' } } })).toThrow()
    })

    test('description optional', () => {
        expect(() => schema.parse({ ...validForm, description: 'Office hours' })).not.toThrow()
    })

    test('description null becomes empty string', () => {
        expect(schema.parse({ ...validForm, description: null }).description).toBe('')
    })

    test('description omitted becomes empty string', () => {
        expect(schema.parse(validForm).description).toBe('')
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
