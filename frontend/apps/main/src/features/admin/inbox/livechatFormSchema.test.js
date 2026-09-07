// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import { createFormSchema } from './livechatFormSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`
const schema = createFormSchema(mockT)

const validConfig = {
    brand_name: 'Acme',
    dark_mode: false,
    show_powered_by: true,
    language: 'en',
    launcher: {
        position: 'right',
        color: '#2563eb',
        spacing: { side: 20, bottom: 20 }
    },
    chat_introduction: 'Ask us anything',
    show_office_hours_in_chat: true,
    show_office_hours_after_assignment: false,
    notice_banner: { enabled: false },
    colors: { primary: '#2563eb' },
    home_screen: {
        header_text_color: 'white',
        background: { type: 'solid', color: '#2563eb' },
        fade_background: true
    },
    features: { file_upload: true, emoji: true },
    session_duration: '720h',
    home_apps: [],
    visitors: {
        start_conversation_button_text: 'Start',
        allow_start_conversation: true,
        prevent_multiple_conversations: false,
        prevent_reply_to_closed_conversation: false
    },
    users: {
        start_conversation_button_text: 'Start',
        allow_start_conversation: true,
        prevent_multiple_conversations: false,
        prevent_reply_to_closed_conversation: false
    },
    prechat_form: { enabled: false, fields: [] }
}

const validForm = {
    name: 'Website chat',
    enabled: true,
    csat_enabled: false,
    prompt_tags_on_reply: false,
    config: validConfig
}

const withConfig = (overrides) => ({ ...validForm, config: { ...validConfig, ...overrides } })

describe('Livechat Inbox Form Schema', () => {
    test('valid minimal form', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('valid complete form', () => {
        expect(() => schema.parse({
            ...validForm,
            secret: 'shh',
            linked_email_inbox_id: 3,
            config: {
                ...validConfig,
                website_url: 'https://acme.example.com',
                fallback_language: 'fr',
                logo_url: 'https://cdn.example.com/logo.png',
                greeting_message: 'Hi there',
                introduction_message: 'We reply fast',
                chat_reply_expectation_message: 'Usually within an hour',
                notice_banner: { enabled: true, text: 'We are on holiday' },
                continuity: {
                    offline_threshold: '5m',
                    max_messages_per_email: 10,
                    min_email_interval: '30m'
                },
                direct_to_conversation: true,
                trusted_domains: 'acme.example.com',
                blocked_ips: '1.2.3.4',
                home_apps: [
                    { type: 'announcement', title: 'News', description: 'Read', text: 'Hi' },
                    { type: 'external_link', title: 'Docs', url: 'https://docs.example.com', image_url: '' }
                ],
                prechat_form: {
                    enabled: true,
                    title: 'Before we start',
                    fields: [{
                        key: 'email',
                        type: 'email',
                        label: 'Email',
                        placeholder: 'you@example.com',
                        required: true,
                        enabled: true,
                        order: 1,
                        is_default: true,
                        custom_attribute_id: 2
                    }]
                }
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

    test('enabled missing', () => {
        const { enabled, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('csat_enabled missing', () => {
        const { csat_enabled, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('config missing', () => {
        const { config, ...form } = validForm
        expect(() => schema.parse(form)).toThrow()
    })

    test('secret and linked_email_inbox_id accept null', () => {
        expect(() => schema.parse({ ...validForm, secret: null, linked_email_inbox_id: null })).not.toThrow()
    })

    test('brand_name empty', () => {
        expect(() => schema.parse(withConfig({ brand_name: '' }))).toThrow()
    })

    test('language empty', () => {
        expect(() => schema.parse(withConfig({ language: '' }))).toThrow()
    })

    test('website_url invalid', () => {
        expect(() => schema.parse(withConfig({ website_url: 'acme.example.com' }))).toThrow()
    })

    test('website_url empty string accepted', () => {
        expect(() => schema.parse(withConfig({ website_url: '' }))).not.toThrow()
    })

    test('primary color invalid hex', () => {
        expect(() => schema.parse(withConfig({ colors: { primary: 'blue' } }))).toThrow()
    })

    test('primary color three digit hex accepted', () => {
        expect(() => schema.parse(withConfig({ colors: { primary: '#fff' } }))).not.toThrow()
    })

    test('primary color eight digit hex rejected', () => {
        expect(() => schema.parse(withConfig({ colors: { primary: '#ffffff00' } }))).toThrow()
    })

    test('launcher position invalid', () => {
        expect(() => schema.parse(withConfig({
            launcher: { ...validConfig.launcher, position: 'center' }
        }))).toThrow()
    })

    test('launcher spacing out of range', () => {
        expect(() => schema.parse(withConfig({
            launcher: { ...validConfig.launcher, spacing: { side: -1, bottom: 20 } }
        }))).toThrow()
        expect(() => schema.parse(withConfig({
            launcher: { ...validConfig.launcher, spacing: { side: 20, bottom: 201 } }
        }))).toThrow()
    })

    test('launcher spacing at boundaries', () => {
        expect(() => schema.parse(withConfig({
            launcher: { ...validConfig.launcher, spacing: { side: 0, bottom: 200 } }
        }))).not.toThrow()
    })

    test('launcher spacing coerced from string', () => {
        const parsed = schema.parse(withConfig({
            launcher: { ...validConfig.launcher, spacing: { side: '30', bottom: '40' } }
        }))
        expect(parsed.config.launcher.spacing.side).toBe(30)
    })

    test('notice banner enabled without text', () => {
        expect(() => schema.parse(withConfig({ notice_banner: { enabled: true } }))).toThrow()
        expect(() => schema.parse(withConfig({ notice_banner: { enabled: true, text: '   ' } }))).toThrow()
    })

    test('notice banner disabled without text accepted', () => {
        expect(() => schema.parse(withConfig({ notice_banner: { enabled: false, text: '' } }))).not.toThrow()
    })

    test('home screen header_text_color invalid', () => {
        expect(() => schema.parse(withConfig({
            home_screen: { ...validConfig.home_screen, header_text_color: 'grey' }
        }))).toThrow()
    })

    test('solid background without a color accepted', () => {
        expect(() => schema.parse(withConfig({
            home_screen: { ...validConfig.home_screen, background: { type: 'solid' } }
        }))).not.toThrow()
    })

    test('gradient background requires both stops', () => {
        expect(() => schema.parse(withConfig({
            home_screen: { ...validConfig.home_screen, background: { type: 'gradient', gradient_start: '#000000' } }
        }))).toThrow()
        expect(() => schema.parse(withConfig({
            home_screen: {
                ...validConfig.home_screen,
                background: { type: 'gradient', gradient_start: '#000000', gradient_end: '#ffffff' }
            }
        }))).not.toThrow()
    })

    test('image background requires an image url', () => {
        expect(() => schema.parse(withConfig({
            home_screen: { ...validConfig.home_screen, background: { type: 'image' } }
        }))).toThrow()
        expect(() => schema.parse(withConfig({
            home_screen: {
                ...validConfig.home_screen,
                background: { type: 'image', image_url: 'https://cdn.example.com/bg.png' }
            }
        }))).not.toThrow()
    })

    test('background type invalid', () => {
        expect(() => schema.parse(withConfig({
            home_screen: { ...validConfig.home_screen, background: { type: 'video' } }
        }))).toThrow()
    })

    test('session_duration invalid duration', () => {
        expect(() => schema.parse(withConfig({ session_duration: '30 days' }))).toThrow()
    })

    test('session_duration empty', () => {
        expect(() => schema.parse(withConfig({ session_duration: '' }))).toThrow()
    })

    test('continuity optional', () => {
        expect(() => schema.parse(validForm)).not.toThrow()
    })

    test('continuity offline_threshold invalid duration', () => {
        expect(() => schema.parse(withConfig({
            continuity: { offline_threshold: '5 min', max_messages_per_email: 10, min_email_interval: '30m' }
        }))).toThrow()
    })

    test('continuity max_messages_per_email out of range', () => {
        expect(() => schema.parse(withConfig({
            continuity: { offline_threshold: '5m', max_messages_per_email: 0, min_email_interval: '30m' }
        }))).toThrow()
        expect(() => schema.parse(withConfig({
            continuity: { offline_threshold: '5m', max_messages_per_email: 101, min_email_interval: '30m' }
        }))).toThrow()
    })

    test('continuity max_messages_per_email at boundaries', () => {
        expect(() => schema.parse(withConfig({
            continuity: { offline_threshold: '5m', max_messages_per_email: 1, min_email_interval: '30m' }
        }))).not.toThrow()
        expect(() => schema.parse(withConfig({
            continuity: { offline_threshold: '5m', max_messages_per_email: 100, min_email_interval: '30m' }
        }))).not.toThrow()
    })

    test('home app type invalid', () => {
        expect(() => schema.parse(withConfig({ home_apps: [{ type: 'banner' }] }))).toThrow()
    })

    test('home app url invalid', () => {
        expect(() => schema.parse(withConfig({ home_apps: [{ type: 'external_link', url: 'docs' }] }))).toThrow()
    })

    test('home app with only a type accepted', () => {
        expect(() => schema.parse(withConfig({ home_apps: [{ type: 'announcement' }] }))).not.toThrow()
    })

    test('prechat field label empty', () => {
        expect(() => schema.parse(withConfig({
            prechat_form: {
                enabled: true,
                fields: [{ key: 'email', type: 'email', label: '', required: true, enabled: true, order: 1, is_default: true }]
            }
        }))).toThrow()
    })

    test('prechat field key empty', () => {
        expect(() => schema.parse(withConfig({
            prechat_form: {
                enabled: true,
                fields: [{ key: '', type: 'email', label: 'Email', required: true, enabled: true, order: 1, is_default: true }]
            }
        }))).toThrow()
    })

    test('prechat field type invalid', () => {
        expect(() => schema.parse(withConfig({
            prechat_form: {
                enabled: true,
                fields: [{ key: 'x', type: 'textarea', label: 'X', required: false, enabled: true, order: 1, is_default: false }]
            }
        }))).toThrow()
    })

    test('prechat field order below minimum', () => {
        expect(() => schema.parse(withConfig({
            prechat_form: {
                enabled: true,
                fields: [{ key: 'x', type: 'text', label: 'X', required: false, enabled: true, order: 0, is_default: false }]
            }
        }))).toThrow()
    })

    test('all prechat field types accepted', () => {
        for (const type of ['text', 'email', 'number', 'checkbox', 'date', 'link', 'list', 'phone']) {
            expect(() => schema.parse(withConfig({
                prechat_form: {
                    enabled: true,
                    fields: [{ key: 'x', type, label: 'X', required: false, enabled: true, order: 1, is_default: false }]
                }
            }))).not.toThrow()
        }
    })

    test('direct_to_conversation defaults to false', () => {
        expect(schema.parse(validForm).config.direct_to_conversation).toBe(false)
    })

    test('empty object', () => {
        expect(() => schema.parse({})).toThrow()
    })
})
