import { describe, test, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { createFormSchema } from './formSchema'
import { BUILT_IN_TEMPLATE_BODIES } from './defaults'

const mockI18n = { _: (key, params) => `${key} ${JSON.stringify(params || {})}` }
const schema = createFormSchema(mockI18n)

const validForm = {
    name: 'New reply',
    body: '<p>Hello</p>',
    type: 'email_notification',
    subject: 'You have a new reply'
}

describe('Templates Form Schema', () => {
    test('valid complete form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name not a string', () => {
        expect(() => schema.parse({ ...validForm, name: 1 })).toThrow()
    })

    test('body missing', () => {
        const { body, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('body not a string', () => {
        expect(() => schema.parse({ ...validForm, body: null })).toThrow()
    })

    test('subject required for non outgoing email templates', () => {
        const { subject, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('subject empty string rejected for non outgoing email templates', () => {
        expect(() => schema.parse({ ...validForm, subject: '' })).toThrow()
    })

    test('subject optional for email_outgoing templates', () => {
        expect(() => schema.parse({ name: 'Outgoing', body: '<p>x</p>', type: 'email_outgoing' })).not.toThrow()
    })

    test('subject optional for the CSAT request template', () => {
        expect(() => schema.parse({ name: 'CSAT request', body: '<p>x</p>', type: 'email_notification' })).not.toThrow()
    })

    test('type optional', () => {
        expect(() => schema.parse({ name: 'x', body: 'y', subject: 'z' })).not.toThrow()
    })

    test('is_default defaults to false', () => {
        expect(schema.parse(validForm).is_default).toBe(false)
    })

    test('is_default must be boolean', () => {
        expect(() => schema.parse({ ...validForm, is_default: 'true' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})

describe('built-in template defaults', () => {
    const repositoryURL = new URL('../../../../../../../', import.meta.url)
    const readBodies = source => new Map(
        [...source.matchAll(/(?:VALUES\s*\(|SELECT)\s*'email_notification'::template_type,\s*'((?:''|[^'])*)'\s*,\s*false,\s*'([^']+)'/g)]
            .map(([, body, name]) => [name, body.replaceAll("''", "'").trim()])
    )
    const schemaBodies = readBodies(readFileSync(new URL('schema.sql', repositoryURL), 'utf8'))
    const upgradeBodies = readBodies(readFileSync(new URL('internal/migrations/v2.9.0-rc.8.go', repositoryURL), 'utf8'))
    const normalize = body => body.replace(/\s+/g, ' ').trim()

    test.each(Object.keys(BUILT_IN_TEMPLATE_BODIES))('%s matches the fresh install default', name => {
        expect(schemaBodies.has(name)).toBe(true)
        expect(normalize(BUILT_IN_TEMPLATE_BODIES[name])).toBe(normalize(schemaBodies.get(name)))
    })

    test.each([
        'New reply from contact',
        'New reply on participating conversation',
        'Conversation reopened'
    ])('%s matches the upgrade default', name => {
        expect(upgradeBodies.get(name)).toBe(BUILT_IN_TEMPLATE_BODIES[name])
    })

    test.each(Object.keys(BUILT_IN_TEMPLATE_BODIES).filter(name => name !== 'CSAT request'))(
        '%s greets the receiving agent exactly once', name => {
            const greeting = '<p>Hi {{ .Recipient.FirstName }},</p>'
            expect(BUILT_IN_TEMPLATE_BODIES[name].startsWith(greeting)).toBe(true)
            expect(BUILT_IN_TEMPLATE_BODIES[name].split(greeting)).toHaveLength(2)
        }
    )

    test('CSAT leaves the greeting to the outgoing template and keeps all five rating links', () => {
        const body = BUILT_IN_TEMPLATE_BODIES['CSAT request']
        expect(body).not.toContain('.Recipient.FirstName')
        for (const rating of [1, 2, 3, 4, 5]) {
            expect(body).toContain(`{{ .CSATLink }}?rating=${rating}`)
        }
    })
})
