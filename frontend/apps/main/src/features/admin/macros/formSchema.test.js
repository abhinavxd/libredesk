// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    name: 'Ask for order id',
    message_content: '<p>Could you share your order id?</p>',
    visibility: 'all',
    visible_when: ['replying']
}

describe('Macros Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid form with actions instead of message content', () => {
        expect(() => schema.parse({
            name: 'Close it',
            visibility: 'all',
            visible_when: ['replying'],
            actions: [{ type: 'set_status', value: ['resolved'] }]
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

    test('neither message content nor actions', () => {
        expect(() => schema.parse({ name: 'x', visibility: 'all', visible_when: [] })).toThrow()
    })

    test('message content with only empty html', () => {
        expect(() => schema.parse({ ...validForm, message_content: '<p></p>' })).toThrow()
    })

    test('message content with only whitespace', () => {
        expect(() => schema.parse({ ...validForm, message_content: '<p>   </p>' })).toThrow()
    })

    test('actions defaults to empty array', () => {
        expect(schema.parse(validForm).actions).toEqual([])
    })

    test('action without a type', () => {
        expect(() => schema.parse({ ...validForm, actions: [{ value: ['resolved'] }] })).toThrow()
    })

    test('action without a value', () => {
        expect(() => schema.parse({ ...validForm, actions: [{ type: 'set_status' }] })).toThrow()
    })

    test('action with empty value array', () => {
        expect(() => schema.parse({ ...validForm, actions: [{ type: 'set_status', value: [] }] })).toThrow()
    })

    test('visibility missing', () => {
        const { visibility, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('visibility invalid value', () => {
        expect(() => schema.parse({ ...validForm, visibility: 'everyone' })).toThrow()
    })

    test('team visibility requires team_id', () => {
        expect(() => schema.parse({ ...validForm, visibility: 'team' })).toThrow()
        expect(() => schema.parse({ ...validForm, visibility: 'team', team_id: '' })).toThrow()
        expect(() => schema.parse({ ...validForm, visibility: 'team', team_id: '2' })).not.toThrow()
    })

    test('user visibility requires user_id', () => {
        expect(() => schema.parse({ ...validForm, visibility: 'user' })).toThrow()
        expect(() => schema.parse({ ...validForm, visibility: 'user', user_id: null })).toThrow()
        expect(() => schema.parse({ ...validForm, visibility: 'user', user_id: '7' })).not.toThrow()
    })

    test('visible_when missing', () => {
        const { visible_when, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('visible_when invalid value', () => {
        expect(() => schema.parse({ ...validForm, visible_when: ['forwarding'] })).toThrow()
    })

    test('visible_when empty array accepted', () => {
        expect(() => schema.parse({ ...validForm, visible_when: [] })).not.toThrow()
    })

    test('all visible_when values accepted', () => {
        expect(() => schema.parse({
            ...validForm,
            visible_when: ['replying', 'starting_conversation', 'adding_private_note']
        })).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
