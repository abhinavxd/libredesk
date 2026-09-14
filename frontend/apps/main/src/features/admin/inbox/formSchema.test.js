// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validImap = {
    host: 'imap.example.com',
    port: 993,
    mailbox: 'INBOX',
    username: 'desk@example.com',
    password: 'secret',
    tls_type: 'tls',
    scan_inbox_since: '48h',
    read_interval: '5m'
}

const validSmtp = {
    host: 'smtp.example.com',
    port: 587,
    username: 'desk@example.com',
    password: 'secret',
    max_conns: 5,
    max_msg_retries: 2,
    idle_timeout: '15s',
    pool_wait_timeout: '5s',
    tls_type: 'starttls',
    auth_protocol: 'plain'
}

const validForm = {
    name: 'Support',
    from: 'Support <desk@example.com>',
    auth_type: 'password',
    imap: validImap,
    smtp: validSmtp
}

const withImap = (overrides) => ({ ...validForm, imap: { ...validImap, ...overrides } })
const withSmtp = (overrides) => ({ ...validForm, smtp: { ...validSmtp, ...overrides } })

describe('Email Inbox Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            from_name_template: '{{ .Agent.FirstName }}',
            reply_to: 'reply@example.com',
            enabled: true,
            csat_enabled: true,
            prompt_tags_on_reply: false,
            enable_plus_addressing: true,
            auth_type: 'oauth2',
            oauth: {
                access_token: 'a',
                client_id: 'b',
                client_secret: 'c',
                expires_at: '2030-01-01',
                provider: 'google',
                refresh_token: 'd'
            }
        })).not.toThrow()
    })

    test('name missing', () => {
        const { name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('name empty string', () => {
        expect(() => schema.parse({ ...validForm, name: '' })).toThrow()
    })

    test('from missing', () => {
        const { from, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('from empty string', () => {
        expect(() => schema.parse({ ...validForm, from: '' })).toThrow()
    })

    test('from_name_template defaults to empty string', () => {
        expect(schema.parse(validForm).from_name_template).toBe('')
    })

    test('from_name_template with an unknown variable', () => {
        expect(() => schema.parse({ ...validForm, from_name_template: '{{ .Agent.Nickname }}' })).toThrow()
    })

    test('from_name_template with unbalanced braces', () => {
        expect(() => schema.parse({ ...validForm, from_name_template: '{{ .Agent.FirstName' })).toThrow()
    })

    test('from_name_template with allowed variables', () => {
        for (const v of ['.Agent.FirstName', '.Agent.LastName', '.Agent.FullName', '.Inbox.Name']) {
            expect(() => schema.parse({ ...validForm, from_name_template: `{{ ${v} }}` })).not.toThrow()
        }
    })

    test('from_name_template plain text accepted', () => {
        expect(() => schema.parse({ ...validForm, from_name_template: 'Support team' })).not.toThrow()
    })

    test('reply_to optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('reply_to empty string accepted', () => {
        expect(() => schema.parse({ ...validForm, reply_to: '' })).not.toThrow()
    })

    test('reply_to invalid email', () => {
        expect(() => schema.parse({ ...validForm, reply_to: 'reply@' })).toThrow()
    })

    test('auth_type missing', () => {
        const { auth_type, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('auth_type invalid value', () => {
        expect(() => schema.parse({ ...validForm, auth_type: 'basic' })).toThrow()
    })

    test('oauth block optional', () => {
        expect(() => schema.parse({ ...validForm, auth_type: 'oauth2' })).not.toThrow()
    })

    test('imap block missing', () => {
        const { imap, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('imap host empty', () => {
        expect(() => schema.parse(withImap({ host: '' }))).toThrow()
    })

    test('imap mailbox empty', () => {
        expect(() => schema.parse(withImap({ mailbox: '' }))).toThrow()
    })

    test('imap username empty', () => {
        expect(() => schema.parse(withImap({ username: '' }))).toThrow()
    })

    test('imap password empty', () => {
        expect(() => schema.parse(withImap({ password: '' }))).toThrow()
    })

    test('imap port out of range', () => {
        expect(() => schema.parse(withImap({ port: 0 }))).toThrow()
        expect(() => schema.parse(withImap({ port: 65536 }))).toThrow()
    })

    test('imap port at boundaries', () => {
        expect(() => schema.parse(withImap({ port: 1 }))).not.toThrow()
        expect(() => schema.parse(withImap({ port: 65535 }))).not.toThrow()
    })

    test('imap tls_type invalid value', () => {
        expect(() => schema.parse(withImap({ tls_type: 'ssl' }))).toThrow()
    })

    test('all imap tls_type values accepted', () => {
        for (const tls_type of ['none', 'starttls', 'tls']) {
            expect(() => schema.parse(withImap({ tls_type }))).not.toThrow()
        }
    })

    test('imap scan_inbox_since invalid duration', () => {
        expect(() => schema.parse(withImap({ scan_inbox_since: '48 hours' }))).toThrow()
    })

    test('imap read_interval invalid duration', () => {
        expect(() => schema.parse(withImap({ read_interval: '5 minutes' }))).toThrow()
    })

    test('imap read_interval compound duration accepted', () => {
        expect(() => schema.parse(withImap({ read_interval: '1h5m30s' }))).not.toThrow()
    })

    test('imap tls_skip_verify optional', () => {
        expect(() => schema.parse(withImap({ tls_skip_verify: true }))).not.toThrow()
    })

    test('smtp block missing', () => {
        const { smtp, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('smtp host empty', () => {
        expect(() => schema.parse(withSmtp({ host: '' }))).toThrow()
    })

    test('smtp port out of range', () => {
        expect(() => schema.parse(withSmtp({ port: 0 }))).toThrow()
        expect(() => schema.parse(withSmtp({ port: 65536 }))).toThrow()
    })

    test('smtp max_conns below minimum', () => {
        expect(() => schema.parse(withSmtp({ max_conns: 0 }))).toThrow()
    })

    test('smtp max_msg_retries out of range', () => {
        expect(() => schema.parse(withSmtp({ max_msg_retries: -1 }))).toThrow()
        expect(() => schema.parse(withSmtp({ max_msg_retries: 101 }))).toThrow()
    })

    test('smtp max_msg_retries at boundaries', () => {
        expect(() => schema.parse(withSmtp({ max_msg_retries: 0 }))).not.toThrow()
        expect(() => schema.parse(withSmtp({ max_msg_retries: 100 }))).not.toThrow()
    })

    test('smtp idle_timeout invalid duration', () => {
        expect(() => schema.parse(withSmtp({ idle_timeout: '15 sec' }))).toThrow()
    })

    test('smtp pool_wait_timeout empty', () => {
        expect(() => schema.parse(withSmtp({ pool_wait_timeout: '' }))).toThrow()
    })

    test('smtp auth_protocol invalid value', () => {
        expect(() => schema.parse(withSmtp({ auth_protocol: 'oauth2' }))).toThrow()
    })

    test('all smtp auth_protocol values accepted', () => {
        for (const auth_protocol of ['login', 'cram', 'plain', 'none']) {
            expect(() => schema.parse(withSmtp({ auth_protocol }))).not.toThrow()
        }
    })

    test('smtp hello_hostname optional', () => {
        expect(() => schema.parse(withSmtp({ hello_hostname: 'desk.example.com' }))).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
