// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    username: 'smtp-user',
    host: 'smtp.example.com',
    password: 'smtp-pass',
    max_conns: 10,
    auth_protocol: 'plain',
    email_address: 'desk@example.com',
    tls_type: 'starttls'
}

describe('Notification Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            enabled: true,
            port: 465,
            idle_timeout: '30s',
            wait_timeout: '10s',
            max_msg_retries: 5,
            hello_hostname: 'desk.example.com',
            tls_skip_verify: true
        })).not.toThrow()
    })

    test('username missing', () => {
        const { username, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('username empty string', () => {
        expect(() => schema.parse({ ...validForm, username: '' })).toThrow()
    })

    test('host missing', () => {
        const { host, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('host empty string', () => {
        expect(() => schema.parse({ ...validForm, host: '' })).toThrow()
    })

    test('password missing', () => {
        const { password, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('email_address missing', () => {
        const { email_address, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('email_address empty string', () => {
        expect(() => schema.parse({ ...validForm, email_address: '' })).toThrow()
    })

    test('port defaults to 587', () => {
        expect(schema.parse(validForm).port).toBe(587)
    })

    test('port below minimum', () => {
        expect(() => schema.parse({ ...validForm, port: 0 })).toThrow()
    })

    test('port at minimum', () => {
        expect(() => schema.parse({ ...validForm, port: 1 })).not.toThrow()
    })

    test('port above maximum', () => {
        expect(() => schema.parse({ ...validForm, port: 65536 })).toThrow()
    })

    test('port at maximum', () => {
        expect(() => schema.parse({ ...validForm, port: 65535 })).not.toThrow()
    })

    test('port as string', () => {
        expect(() => schema.parse({ ...validForm, port: '587' })).toThrow()
    })

    test('max_conns missing', () => {
        const { max_conns, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('max_conns below minimum', () => {
        expect(() => schema.parse({ ...validForm, max_conns: 0 })).toThrow()
    })

    test('max_conns at minimum', () => {
        expect(() => schema.parse({ ...validForm, max_conns: 1 })).not.toThrow()
    })

    test('max_conns above maximum', () => {
        expect(() => schema.parse({ ...validForm, max_conns: 1001 })).toThrow()
    })

    test('max_conns at maximum', () => {
        expect(() => schema.parse({ ...validForm, max_conns: 1000 })).not.toThrow()
    })

    test('max_msg_retries defaults to 2', () => {
        expect(schema.parse(validForm).max_msg_retries).toBe(2)
    })

    test('max_msg_retries at minimum', () => {
        expect(() => schema.parse({ ...validForm, max_msg_retries: 0 })).not.toThrow()
    })

    test('max_msg_retries below minimum', () => {
        expect(() => schema.parse({ ...validForm, max_msg_retries: -1 })).toThrow()
    })

    test('max_msg_retries above maximum', () => {
        expect(() => schema.parse({ ...validForm, max_msg_retries: 1001 })).toThrow()
    })

    test('idle_timeout defaults to 15s', () => {
        expect(schema.parse(validForm).idle_timeout).toBe('15s')
    })

    test('wait_timeout defaults to 5s', () => {
        expect(schema.parse(validForm).wait_timeout).toBe('5s')
    })

    test('idle_timeout invalid duration', () => {
        expect(() => schema.parse({ ...validForm, idle_timeout: '15 seconds' })).toThrow()
    })

    test('idle_timeout empty string', () => {
        expect(() => schema.parse({ ...validForm, idle_timeout: '' })).toThrow()
    })

    test('wait_timeout compound duration accepted', () => {
        expect(() => schema.parse({ ...validForm, wait_timeout: '1h30m5s' })).not.toThrow()
    })

    test('auth_protocol missing', () => {
        const { auth_protocol, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('auth_protocol invalid value', () => {
        expect(() => schema.parse({ ...validForm, auth_protocol: 'oauth2' })).toThrow()
    })

    test('all auth_protocol values accepted', () => {
        for (const auth_protocol of ['plain', 'login', 'cram', 'none']) {
            expect(() => schema.parse({ ...validForm, auth_protocol })).not.toThrow()
        }
    })

    test('tls_type missing', () => {
        const { tls_type, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('tls_type invalid value', () => {
        expect(() => schema.parse({ ...validForm, tls_type: 'ssl' })).toThrow()
    })

    test('all tls_type values accepted', () => {
        for (const tls_type of ['none', 'starttls', 'tls']) {
            expect(() => schema.parse({ ...validForm, tls_type })).not.toThrow()
        }
    })

    test('enabled defaults to false', () => {
        expect(schema.parse(validForm).enabled).toBe(false)
    })

    test('hello_hostname and tls_skip_verify optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
