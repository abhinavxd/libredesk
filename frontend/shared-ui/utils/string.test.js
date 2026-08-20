// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import {
    validateEmail,
    isGoDuration,
    isGoHourMinuteDuration,
    getTextFromHTML,
    getInitials
} from './string'

const valid = [
    'user@example.com',
    'USER@EXAMPLE.COM',
    'MiXeD.CaSe@Example.Co',
    'rbhise00@gmail.com',
    'first.last@example.com',
    'a.b.c.d.e@example.com',
    'user+tag@example.com',
    'user+tag+more@example.com',
    'user-name@example.com',
    'user_name@example.com',
    'user123@example.com',
    '123456@example.com',
    'a@b.co',
    "o'brien@example.com",
    'user!test@example.com',
    'user#test@example.com',
    'user$test@example.com',
    'user%test@example.com',
    'user&test@example.com',
    "user'test@example.com",
    'user*test@example.com',
    'user/test@example.com',
    'user=test@example.com',
    'user?test@example.com',
    'user^test@example.com',
    'user`test@example.com',
    'user{test}@example.com',
    'user|test@example.com',
    'user~test@example.com',
    'user@sub.example.com',
    'user@deep.sub.domain.example.com',
    'user@my-domain.com',
    'user@my-long-domain-name.co.uk',
    'user@example.museum',
    'user@example.travel',
    'user@x.io',
    'user@1domain.com',
    'user@domain1.com',
    'user@123.example.com',
    'support@zerodha.com',
    'agent.name+conv123@support.zerodha.com',

    // Display-name form an agent can type into the reply box
    'Name <user@example.com>',
    'Full Name <user@example.com>',
    'Name<user@example.com>',
    '<user@example.com>',
    '"Last, First" <user@example.com>',
    "O'Brien <user@example.com>",
    'Support Team <support@zerodha.com>',
    'Name  <user@example.com>'
]

const invalid = [
    // The prod failure: a trailing paren slipped through the old check and broke the send.
    'rbhise00@gmail.com)',
    'user@example.com(',
    'user@example.com>',
    'user@example.com<',
    'user@example.com]',
    'user@example.com[',
    'user@example.com;',
    'user@example.com:',
    'user@example.com,',
    'user@example.com"',
    "user@example.com'",
    'user@example.com\\',
    '(user@example.com',
    'Name user@example.com',
    'user@example.com Name',
    'Name <user@example.com',
    'Name user@example.com>',
    'Name <>',
    'Name <not-an-email>',
    'Name <user@example.com)>',
    'Name <a@b.com> extra',
    'Name <a@b.com><c@d.com>',
    'Na<me <a@b.com>',
    '"unclosed <a@b.com>',
    'Name <a@b.com, c@d.com>',

    // Missing or repeated @
    '',
    'user',
    'example.com',
    '@example.com',
    'user@',
    'user@@example.com',
    'user@example@com',
    'a@b@c.com',

    // Domain shape
    'user@example',
    'user@.com',
    'user@example.',
    'user@example..com',
    'user@-example.com',
    'user@example-.com',
    'user@exam ple.com',
    'user@example.c',
    'user@example.c0m',
    'user@example.123',
    'user@example.co-',
    'user@[192.168.0.1]',
    'user@192.168.0.1',

    // Local part shape
    '.user@example.com',
    'user.@example.com',
    'us..er@example.com',
    'user name@example.com',
    'user\tname@example.com',
    'user\nname@example.com',

    // Whitespace and control characters anywhere
    ' user@example.com',
    'user@example.com ',
    ' user@example.com ',
    '\tuser@example.com',
    'user@example.com\n',
    'user@example.com\r',
    'user@ example.com',
    'user @example.com',

    // Injection-shaped input a paste can carry
    'user@example.com\nBcc: evil@example.com',
    'user@example.com%0ABcc:evil@example.com',
    'user@example.com, other@example.com',
    'user@example.com;other@example.com',

    // Non-strings and other junk
    'just some text',
    'http://example.com',
    'mailto:user@example.com',
    'user@example.com/path',
    '@',
    '.',
    '..'
]

describe('validateEmail', () => {
    test.each(valid)('accepts %j', (email) => {
        expect(validateEmail(email)).toBe(true)
    })

    test.each(invalid)('rejects %j', (email) => {
        expect(validateEmail(email)).toBe(false)
    })

    test('rejects non-string input', () => {
        expect(validateEmail(null)).toBe(false)
        expect(validateEmail(undefined)).toBe(false)
        expect(validateEmail(0)).toBe(false)
        expect(validateEmail(123)).toBe(false)
        expect(validateEmail(true)).toBe(false)
        expect(validateEmail({})).toBe(false)
        expect(validateEmail([])).toBe(false)
        expect(validateEmail(['user@example.com'])).toBe(false)
    })

    test('rejects an address longer than the 254 char limit', () => {
        const domain = '@example.com'
        const atLimit = `${'a'.repeat(254 - domain.length)}${domain}`
        expect(atLimit).toHaveLength(254)
        expect(validateEmail(atLimit)).toBe(true)
        expect(validateEmail(`a${atLimit}`)).toBe(false)
        expect(validateEmail(`${'a'.repeat(1000)}@example.com`)).toBe(false)
    })

    test('is not vulnerable to a multiline bypass', () => {
        expect(validateEmail('user@example.com\nrbhise00@gmail.com)')).toBe(false)
        expect(validateEmail('bad)\nuser@example.com')).toBe(false)
    })
})

describe('isGoDuration', () => {
    test.each(['1h', '30m', '45s', '1h30m', '1h30m15s', '2h15s', '10m30s', '0s', '999h'])(
        'accepts %j',
        (v) => expect(isGoDuration(v)).toBe(true)
    )

    test.each(['', '1d', '1w', 'h', 'm30', '30', '1.5h', '1 h', '30m ', 'abc', '-1h', '1h30x'])(
        'rejects %j',
        (v) => expect(isGoDuration(v)).toBe(false)
    )
})

describe('isGoHourMinuteDuration', () => {
    test.each(['1h', '24h', '30m', '0h', '0m'])('accepts %j', (v) =>
        expect(isGoHourMinuteDuration(v)).toBe(true)
    )

    test.each(['', '45s', '1h30m', '1d', '30', 'h', '1.5h', ' 1h', '1h '])('rejects %j', (v) =>
        expect(isGoHourMinuteDuration(v)).toBe(false)
    )
})

describe('getTextFromHTML', () => {
    test('strips tags and trims', () => {
        expect(getTextFromHTML('<p>Hello <b>world</b></p>')).toBe('Hello world')
        expect(getTextFromHTML('  <div> spaced </div>  ')).toBe('spaced')
    })

    test('returns empty string for empty or tag-only html', () => {
        expect(getTextFromHTML('')).toBe('')
        expect(getTextFromHTML('<br>')).toBe('')
        expect(getTextFromHTML('<div><span></span></div>')).toBe('')
    })

    test('does not leak content between calls', () => {
        expect(getTextFromHTML('<p>first</p>')).toBe('first')
        expect(getTextFromHTML('<p>second</p>')).toBe('second')
    })

    test('keeps script text as text rather than executing it', () => {
        expect(getTextFromHTML('<div>keep</div><script>alert(1)</script>')).toBe('keepalert(1)')
    })
})

describe('getInitials', () => {
    test('builds initials from both names', () => {
        expect(getInitials('John', 'Doe')).toBe('JD')
        expect(getInitials('john', 'doe')).toBe('JD')
    })

    test('handles a missing name', () => {
        expect(getInitials('John', '')).toBe('J')
        expect(getInitials('', 'Doe')).toBe('D')
        expect(getInitials()).toBe('')
        expect(getInitials('', '')).toBe('')
    })

    test('uses the first character even when it is not a letter', () => {
        expect(getInitials('123', '456')).toBe('14')
        expect(getInitials(' john', 'doe')).toBe(' D')
    })
})
