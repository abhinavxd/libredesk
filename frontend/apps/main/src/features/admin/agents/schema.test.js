import { describe, test, expect } from 'vitest'
import { createFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validForm = {
    first_name: 'John',
    email: 'test@test.com',
    roles: ['admin'],
    new_password: 'Password123!'
}

describe('Form Schema', () => {
    // Valid cases
    test('valid complete form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid minimal form', () => {
        expect(() => schema.parse({
            first_name: 'Jo',
            email: 'a@b.co',
            roles: ['user']
        })).not.toThrow()
    })

    // First name tests
    test('first_name too short', () => {
        expect(() => schema.parse({ ...validForm, first_name: 'J' })).toThrow()
    })

    test('first_name too long', () => {
        expect(() => schema.parse({ ...validForm, first_name: 'a'.repeat(51) })).toThrow()
    })

    test('first_name missing', () => {
        const { first_name, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('first_name empty string', () => {
        expect(() => schema.parse({ ...validForm, first_name: '' })).toThrow()
    })

    test('first_name null', () => {
        expect(() => schema.parse({ ...validForm, first_name: null })).toThrow()
    })

    // Email tests
    test('invalid email format', () => {
        expect(() => schema.parse({ ...validForm, email: 'invalid' })).toThrow()
    })

    test('email missing @', () => {
        expect(() => schema.parse({ ...validForm, email: 'test.com' })).toThrow()
    })

    test('email missing domain', () => {
        expect(() => schema.parse({ ...validForm, email: 'test@' })).toThrow()
    })

    test('email empty', () => {
        expect(() => schema.parse({ ...validForm, email: '' })).toThrow()
    })

    test('email missing', () => {
        const { email, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    // Roles tests
    test('roles empty array', () => {
        expect(() => schema.parse({ ...validForm, roles: [] })).toThrow()
    })

    test('roles missing', () => {
        const { roles, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('roles multiple values', () => {
        expect(() => schema.parse({ ...validForm, roles: ['admin', 'user', 'moderator'] })).not.toThrow()
    })

    // Password tests
    test('password too short', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Pass1!' })).toThrow()
    })

    test('password too long', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'P'.repeat(73) + 'a1!' })).toThrow()
    })

    test('password missing uppercase', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'password123!' })).toThrow()
    })

    test('password missing lowercase', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORD123!' })).toThrow()
    })

    test('password missing number', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password!@#$' })).toThrow()
    })

    test('password missing special char', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123' })).toThrow()
    })

    test('password only special chars', () => {
        expect(() => schema.parse({ ...validForm, new_password: '!@#$%^&*()' })).toThrow()
    })

    test('password unicode special chars', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123ñ' })).not.toThrow()
    })

    test('password underscore as special char', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123_' })).not.toThrow()
    })

    test('password exactly 10 chars', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password1!' })).not.toThrow()
    })

    test('password exactly 72 chars', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'P'.repeat(69) + 'a1!' })).not.toThrow()
    })

    // Optional fields
    test('last_name optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
        expect(() => schema.parse({ ...validForm, last_name: 'Doe' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, last_name: '' })).not.toThrow()
    })

    test('send_welcome_email optional', () => {
        expect(() => schema.parse({ ...validForm, send_welcome_email: true })).not.toThrow()
        expect(() => schema.parse({ ...validForm, send_welcome_email: false })).not.toThrow()
    })

    test('enabled defaults to true', () => {
        const result = schema.parse(validForm)
        expect(result.enabled).toBe(true)
    })

    test('availability_status defaults to offline', () => {
        const result = schema.parse(validForm)
        expect(result.availability_status).toBe('offline')
    })

    test('teams defaults to empty array', () => {
        const result = schema.parse(validForm)
        expect(result.teams).toEqual([])
    })

    test('teams with values', () => {
        expect(() => schema.parse({ ...validForm, teams: ['team1', 'team2'] })).not.toThrow()
    })

    // Edge cases
    test('undefined values', () => {
        expect(() => schema.parse({
            first_name: undefined,
            email: 'test@test.com',
            roles: ['admin']
        })).toThrow()
    })

    test('null values', () => {
        expect(() => schema.parse({
            first_name: null,
            email: 'test@test.com',
            roles: ['admin']
        })).toThrow()
    })

    test('number as string field', () => {
        expect(() => schema.parse({ ...validForm, first_name: 123 })).toThrow()
    })

    test('string as boolean field', () => {
        expect(() => schema.parse({ ...validForm, enabled: 'true' })).toThrow()
    })

    test('string as array field', () => {
        expect(() => schema.parse({ ...validForm, roles: 'admin' })).toThrow()
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })

    test('extra unknown fields ignored', () => {
        expect(() => schema.parse({
            ...validForm,
            unknown_field: 'value',
            another_field: 123
        })).not.toThrow()
    })
})

// Password regex validation tests
describe('Password Regex Validation', () => {
    // Lowercase tests
    test('lowercase - single letter', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORD123!a' })).not.toThrow()
    })

    test('lowercase - multiple letters', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORDabc123!' })).not.toThrow()
    })

    test('lowercase - accented characters', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORD123!ñ' })).toThrow()
    })

    test('lowercase - none', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORD123!' })).toThrow()
    })

    // Uppercase tests
    test('uppercase - single letter', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'passwordA123!' })).not.toThrow()
    })

    test('uppercase - multiple letters', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'passwordABC123!' })).not.toThrow()
    })

    test('uppercase - accented characters', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'passwordÑ123!' })).toThrow()
    })

    test('uppercase - none', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'password123!' })).toThrow()
    })

    // Digit tests
    test('digit - single number', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password1!' })).not.toThrow()
    })

    test('digit - multiple numbers', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123!' })).not.toThrow()
    })

    test('digit - zero', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password0!' })).not.toThrow()
    })

    test('digit - none', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password!' })).toThrow()
    })

    // Special character tests
    test('special - common symbols', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123!' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123@' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123#' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123$' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123%' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123^' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123&' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123*' })).not.toThrow()
    })

    test('special - brackets and parentheses', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123(' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123)' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123[' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123]' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123{' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123}' })).not.toThrow()
    })

    test('special - punctuation', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123.' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123,' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123;' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123:' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123?' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123/' })).not.toThrow()
    })

    test('special - quotes and backslash', () => {
        expect(() => schema.parse({ ...validForm, new_password: "Password123'" })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123"' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123\\' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123|' })).not.toThrow()
    })

    test('special - math symbols', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123+' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123-' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123=' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123<' })).not.toThrow()
        expect(() => schema.parse({ ...validForm, new_password: 'Password123>' })).not.toThrow()
    })

    test('special - underscore', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123_' })).not.toThrow()
    })

    test('special - space', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123 ' })).not.toThrow()
    })

    test('special - none', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123' })).toThrow()
    })

    // Combination edge cases
    test('only uppercase and special', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORD!@#$%^&*()' })).toThrow()
    })

    test('only lowercase and digits', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'password123456' })).toThrow()
    })

    test('whitespace only special char', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123   ' })).not.toThrow()
    })

    test('tab as special char', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123\t' })).not.toThrow()
    })

    test('newline as special char', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password123\n' })).not.toThrow()
    })
})


// Password validation - passing cases
describe('Password Valid Cases', () => {
    test('exact minimum length with all requirements', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Password1!' })).not.toThrow()
    })

    test('exact maximum length with all requirements', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'P'.repeat(67) + 'ass1!' })).not.toThrow()
    })

    test('multiple of each requirement', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PASSWORDpassword123456!@#$%^&*()' })).not.toThrow()
    })

    test('mixed case throughout', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'PaSSwoRD123!@#' })).not.toThrow()
    })

    test('numbers at start', () => {
        expect(() => schema.parse({ ...validForm, new_password: '123Password!' })).not.toThrow()
    })

    test('special chars at start', () => {
        expect(() => schema.parse({ ...validForm, new_password: '!@#Password123' })).not.toThrow()
    })

    test('all character types mixed', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'P@ssw0rd123!' })).not.toThrow()
    })

    test('unicode characters', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Påssw0rd123!' })).not.toThrow()
    })

    test('long valid password', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'ThisIsAVeryLongPasswordWith123!SpecialChars' })).not.toThrow()
    })

    test('password with spaces', () => {
        expect(() => schema.parse({ ...validForm, new_password: 'Pass Word 123!' })).not.toThrow()
    })
})