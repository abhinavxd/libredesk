// @vitest-environment jsdom
import { describe, test, expect } from 'vitest'
import {
  createFormSchema,
  moveCampaign,
  normalizeAudienceConfig,
  normalizePrechatConfig
} from './livechatFormSchema'

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
const validCampaign = {
  id: '8a3660e6-e29b-461c-924f-314c7576f75a',
  name: 'Pricing invitation',
  enabled: true,
  message: 'Need help choosing a plan?',
  sender_id: 0,
  team_id: 0,
  audience: 'all',
  include_urls: ['/pricing'],
  exclude_urls: [],
  conditions: { logical_op: 'AND', rules: [] },
  event: '',
  delay_seconds: 10,
  business_hours_id: 0,
  business_hours: 'any',
  desktop: true,
  mobile: true,
  repeat: 'once',
  repeat_hours: 24
}

describe('Livechat Inbox Form Schema', () => {
  test('valid minimal form', () => {
    const parsed = schema.parse(validForm)
    expect(parsed.config.previews).toEqual({
      desktop: true,
      mobile: true,
      content: 'message',
      auto_hide_seconds: 0
    })
  })

  test('help tab preserves audience placement and featured article order', () => {
    const parsed = schema.parse(
      withConfig({
        help: {
          help_center_id: 4,
          visitors: { tab: true },
          users: { tab: false },
          featured_ids: [13, 8, 21]
        }
      })
    )

    expect(parsed.config.help).toEqual({
      help_center_id: 4,
      visitors: { tab: true },
      users: { tab: false },
      featured_ids: [13, 8, 21]
    })
  })

  test('help tab rejects more than ten featured articles', () => {
    expect(() =>
      schema.parse(
        withConfig({
          help: {
            help_center_id: 4,
            visitors: { tab: true },
            users: { tab: true },
            featured_ids: Array.from({ length: 11 }, (_, index) => index + 1)
          }
        })
      )
    ).toThrow()
  })

  test('reply preview settings enforce the auto-hide range', () => {
    expect(() =>
      schema.parse(
        withConfig({
          previews: { desktop: true, mobile: false, content: 'generic', auto_hide_seconds: 300 }
        })
      )
    ).not.toThrow()
    expect(() =>
      schema.parse(
        withConfig({
          previews: { desktop: true, mobile: true, content: 'message', auto_hide_seconds: 301 }
        })
      )
    ).toThrow()
  })

  test('valid complete form', () => {
    expect(() =>
      schema.parse({
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
          quick_replies: 'Billing question\nReset my password',
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
            fields: [
              {
                key: 'email',
                type: 'email',
                label: 'Email',
                placeholder: 'you@example.com',
                required: true,
                enabled: true,
                order: 1,
                is_default: true,
                custom_attribute_id: 2
              }
            ]
          }
        }
      })
    ).not.toThrow()
  })

  test('quick replies accept empty and repeated values', () => {
    expect(() => schema.parse(withConfig({ quick_replies: '' }))).not.toThrow()
    expect(() => schema.parse(withConfig({ quick_replies: 'Billing\nBilling' }))).not.toThrow()
  })

  test('campaigns trim and discard blank URL rows', () => {
    const parsed = schema.parse(
      withConfig({ campaigns: [{ ...validCampaign, include_urls: [' /pricing ', '', '   '] }] })
    )
    expect(parsed.config.campaigns[0].include_urls).toEqual(['/pricing'])
  })

  test('campaigns require a name and message', () => {
    const result = schema.safeParse(
      withConfig({ campaigns: [{ ...validCampaign, name: ' ', message: '' }] })
    )
    expect(result.success).toBe(false)
    expect(result.error.issues.map((issue) => issue.path.join('.'))).toEqual(
      expect.arrayContaining(['config.campaigns.0.name', 'config.campaigns.0.message'])
    )
  })

  test('campaigns require at least one target device', () => {
    const result = schema.safeParse(
      withConfig({ campaigns: [{ ...validCampaign, desktop: false, mobile: false }] })
    )
    expect(result.success).toBe(false)
    expect(result.error.issues[0].path.join('.')).toBe('config.campaigns.0.desktop')
  })

  test('campaign business-hour targeting requires a schedule', () => {
    const result = schema.safeParse(
      withConfig({ campaigns: [{ ...validCampaign, business_hours: 'inside' }] })
    )
    expect(result.success).toBe(false)
    expect(result.error.issues[0].path.join('.')).toBe('config.campaigns.0.business_hours_id')
  })

  test('incomplete campaign conditions use product validation copy', () => {
    const result = schema.safeParse(
      withConfig({
        campaigns: [
          {
            ...validCampaign,
            conditions: {
              logical_op: 'AND',
              rules: [
                {
                  field: '',
                  field_type: 'contact_custom_attribute',
                  operator: '',
                  value: '',
                  case_sensitive_match: false
                }
              ]
            }
          }
        ]
      })
    )
    expect(result.success).toBe(false)
    expect(result.error.issues.map((issue) => issue.message)).toEqual([
      'globals.terms.required {}',
      'globals.terms.required {}'
    ])
  })

  test('campaign limits use product validation copy', () => {
    const result = schema.safeParse(
      withConfig({
        campaigns: [
          {
            ...validCampaign,
            include_urls: Array.from({ length: 21 }, (_, index) => `/page-${index}`),
            delay_seconds: -1,
            repeat_hours: 0
          }
        ],
        campaign_cooldown_hours: 0
      })
    )
    expect(result.success).toBe(false)
    expect(result.error.issues.map((issue) => issue.message)).toEqual([
      'widget.campaignUrlLimit {}',
      'validation.minmaxNumber {"min":0,"max":86400}',
      'validation.minmaxNumber {"min":1,"max":8760}',
      'validation.minmaxNumber {"min":1,"max":8760}'
    ])
  })

  test('quick replies reject more than six non-empty lines', () => {
    expect(() => schema.parse(withConfig({ quick_replies: '1\n2\n3\n4\n5\n6\n7' }))).toThrow()
    expect(() => schema.parse(withConfig({ quick_replies: '1\n2\n\n3\n4\n5\n6' }))).not.toThrow()
  })

  test('quick replies reject entries over 120 characters', () => {
    expect(() => schema.parse(withConfig({ quick_replies: 'x'.repeat(121) }))).toThrow()
  })

  test('audience quick replies are validated independently', () => {
    expect(() =>
      schema.parse(
        withConfig({
          visitors: { ...validConfig.visitors, quick_replies: '1\n2\n3\n4\n5\n6\n7' },
          users: { ...validConfig.users, quick_replies: '' }
        })
      )
    ).toThrow()
  })

  test('legacy audience settings are copied without changing behavior', () => {
    const config = {
      ...validConfig,
      quick_replies: ['Billing', 'Support'],
      direct_to_conversation: true
    }
    expect(normalizeAudienceConfig(config, 'visitors')).toMatchObject({
      quick_replies: 'Billing\nSupport',
      direct_to_conversation: true
    })
    expect(normalizeAudienceConfig(config, 'users')).toMatchObject({
      quick_replies: 'Billing\nSupport',
      direct_to_conversation: true
    })
  })

  test('explicit empty audience quick replies do not fall back to legacy replies', () => {
    const config = {
      ...validConfig,
      quick_replies: ['Legacy'],
      visitors: { ...validConfig.visitors, quick_replies: [], direct_to_conversation: false }
    }
    expect(normalizeAudienceConfig(config, 'visitors')).toMatchObject({
      quick_replies: '',
      direct_to_conversation: false
    })
  })

  test('legacy prechat form is copied to both audiences', () => {
    const fields = [{ key: 'plan', label: 'Plan' }]
    const config = normalizePrechatConfig({
      enabled: true,
      title: 'Before we start',
      fields
    })

    expect(config.visitors).toEqual({ enabled: true, title: 'Before we start', fields })
    expect(config.users).toEqual({ enabled: true, title: 'Before we start', fields })
    expect(config.visitors.fields).not.toBe(config.users.fields)
  })

  test('audience prechat forms preserve separate fields', () => {
    const config = normalizePrechatConfig({
      enabled: true,
      title: 'Legacy',
      fields: [{ key: 'legacy' }],
      visitors: { enabled: true, title: 'Choose a plan', fields: [{ key: 'plan' }] },
      users: { enabled: false, title: 'What is the issue?', fields: [] }
    })

    expect(config.visitors).toEqual({
      enabled: true,
      title: 'Choose a plan',
      fields: [{ key: 'plan' }]
    })
    expect(config.users).toEqual({ enabled: false, title: 'What is the issue?', fields: [] })
  })

  test('campaign priority follows saved array order', () => {
    const campaigns = [{ id: 'first' }, { id: 'second' }, { id: 'third' }]
    expect(moveCampaign(campaigns, 2, -1).map(({ id }) => id)).toEqual(['first', 'third', 'second'])
    expect(campaigns.map(({ id }) => id)).toEqual(['first', 'second', 'third'])
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
    expect(() =>
      schema.parse({ ...validForm, secret: null, linked_email_inbox_id: null })
    ).not.toThrow()
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
    expect(() =>
      schema.parse(
        withConfig({
          launcher: { ...validConfig.launcher, position: 'center' }
        })
      )
    ).toThrow()
  })

  test('launcher spacing out of range', () => {
    expect(() =>
      schema.parse(
        withConfig({
          launcher: { ...validConfig.launcher, spacing: { side: -1, bottom: 20 } }
        })
      )
    ).toThrow()
    expect(() =>
      schema.parse(
        withConfig({
          launcher: { ...validConfig.launcher, spacing: { side: 20, bottom: 201 } }
        })
      )
    ).toThrow()
  })

  test('launcher spacing at boundaries', () => {
    expect(() =>
      schema.parse(
        withConfig({
          launcher: { ...validConfig.launcher, spacing: { side: 0, bottom: 200 } }
        })
      )
    ).not.toThrow()
  })

  test('launcher spacing coerced from string', () => {
    const parsed = schema.parse(
      withConfig({
        launcher: { ...validConfig.launcher, spacing: { side: '30', bottom: '40' } }
      })
    )
    expect(parsed.config.launcher.spacing.side).toBe(30)
  })

  test('notice banner enabled without text', () => {
    expect(() => schema.parse(withConfig({ notice_banner: { enabled: true } }))).toThrow()
    expect(() =>
      schema.parse(withConfig({ notice_banner: { enabled: true, text: '   ' } }))
    ).toThrow()
  })

  test('notice banner disabled without text accepted', () => {
    expect(() =>
      schema.parse(withConfig({ notice_banner: { enabled: false, text: '' } }))
    ).not.toThrow()
  })

  test('home screen header_text_color invalid', () => {
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: { ...validConfig.home_screen, header_text_color: 'grey' }
        })
      )
    ).toThrow()
  })

  test('solid background without a color accepted', () => {
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: { ...validConfig.home_screen, background: { type: 'solid' } }
        })
      )
    ).not.toThrow()
  })

  test('gradient background requires both stops', () => {
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: {
            ...validConfig.home_screen,
            background: { type: 'gradient', gradient_start: '#000000' }
          }
        })
      )
    ).toThrow()
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: {
            ...validConfig.home_screen,
            background: { type: 'gradient', gradient_start: '#000000', gradient_end: '#ffffff' }
          }
        })
      )
    ).not.toThrow()
  })

  test('image background requires an image url', () => {
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: { ...validConfig.home_screen, background: { type: 'image' } }
        })
      )
    ).toThrow()
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: {
            ...validConfig.home_screen,
            background: { type: 'image', image_url: 'https://cdn.example.com/bg.png' }
          }
        })
      )
    ).not.toThrow()
  })

  test('background type invalid', () => {
    expect(() =>
      schema.parse(
        withConfig({
          home_screen: { ...validConfig.home_screen, background: { type: 'video' } }
        })
      )
    ).toThrow()
  })

  test('session_duration invalid duration', () => {
    expect(() => schema.parse(withConfig({ session_duration: '30 days' }))).toThrow()
  })

  test.each(['0h', '59m59s', '-1h'])('session_duration rejects %s', (sessionDuration) => {
    expect(() => schema.parse(withConfig({ session_duration: sessionDuration }))).toThrow()
  })

  test.each(['1h', '60m', '3600s'])('session_duration accepts %s', (sessionDuration) => {
    expect(() => schema.parse(withConfig({ session_duration: sessionDuration }))).not.toThrow()
  })

  test('session_duration empty', () => {
    expect(() => schema.parse(withConfig({ session_duration: '' }))).toThrow()
  })

  test('continuity optional', () => {
    expect(() => schema.parse(validForm)).not.toThrow()
  })

  test('continuity offline_threshold invalid duration', () => {
    expect(() =>
      schema.parse(
        withConfig({
          continuity: {
            offline_threshold: '5 min',
            max_messages_per_email: 10,
            min_email_interval: '30m'
          }
        })
      )
    ).toThrow()
  })

  test('continuity max_messages_per_email out of range', () => {
    expect(() =>
      schema.parse(
        withConfig({
          continuity: {
            offline_threshold: '5m',
            max_messages_per_email: 0,
            min_email_interval: '30m'
          }
        })
      )
    ).toThrow()
    expect(() =>
      schema.parse(
        withConfig({
          continuity: {
            offline_threshold: '5m',
            max_messages_per_email: 101,
            min_email_interval: '30m'
          }
        })
      )
    ).toThrow()
  })

  test('continuity max_messages_per_email at boundaries', () => {
    expect(() =>
      schema.parse(
        withConfig({
          continuity: {
            offline_threshold: '5m',
            max_messages_per_email: 1,
            min_email_interval: '30m'
          }
        })
      )
    ).not.toThrow()
    expect(() =>
      schema.parse(
        withConfig({
          continuity: {
            offline_threshold: '5m',
            max_messages_per_email: 100,
            min_email_interval: '30m'
          }
        })
      )
    ).not.toThrow()
  })

  test('home app type invalid', () => {
    expect(() => schema.parse(withConfig({ home_apps: [{ type: 'banner' }] }))).toThrow()
  })

  test('home app url invalid', () => {
    expect(() =>
      schema.parse(withConfig({ home_apps: [{ type: 'external_link', url: 'docs' }] }))
    ).toThrow()
  })

  test('home app with only a type accepted', () => {
    expect(() => schema.parse(withConfig({ home_apps: [{ type: 'announcement' }] }))).not.toThrow()
  })

  test('prechat field label empty', () => {
    expect(() =>
      schema.parse(
        withConfig({
          prechat_form: {
            enabled: true,
            fields: [
              {
                key: 'email',
                type: 'email',
                label: '',
                required: true,
                enabled: true,
                order: 1,
                is_default: true
              }
            ]
          }
        })
      )
    ).toThrow()
  })

  test('prechat field key empty', () => {
    expect(() =>
      schema.parse(
        withConfig({
          prechat_form: {
            enabled: true,
            fields: [
              {
                key: '',
                type: 'email',
                label: 'Email',
                required: true,
                enabled: true,
                order: 1,
                is_default: true
              }
            ]
          }
        })
      )
    ).toThrow()
  })

  test('prechat field type invalid', () => {
    expect(() =>
      schema.parse(
        withConfig({
          prechat_form: {
            enabled: true,
            fields: [
              {
                key: 'x',
                type: 'textarea',
                label: 'X',
                required: false,
                enabled: true,
                order: 1,
                is_default: false
              }
            ]
          }
        })
      )
    ).toThrow()
  })

  test('prechat field order below minimum', () => {
    expect(() =>
      schema.parse(
        withConfig({
          prechat_form: {
            enabled: true,
            fields: [
              {
                key: 'x',
                type: 'text',
                label: 'X',
                required: false,
                enabled: true,
                order: 0,
                is_default: false
              }
            ]
          }
        })
      )
    ).toThrow()
  })

  test('all prechat field types accepted', () => {
    for (const type of ['text', 'email', 'number', 'checkbox', 'date', 'link', 'list', 'phone']) {
      expect(() =>
        schema.parse(
          withConfig({
            prechat_form: {
              enabled: true,
              fields: [
                {
                  key: 'x',
                  type,
                  label: 'X',
                  required: false,
                  enabled: true,
                  order: 1,
                  is_default: false
                }
              ]
            }
          })
        )
      ).not.toThrow()
    }
  })

  test('audience prechat fields are validated independently', () => {
    const result = schema.safeParse(
      withConfig({
        prechat_form: {
          enabled: true,
          title: 'Legacy',
          fields: [],
          visitors: {
            enabled: true,
            title: 'Choose a plan',
            fields: [
              {
                key: 'plan',
                type: 'text',
                label: '',
                required: true,
                enabled: true,
                order: 1,
                is_default: false
              }
            ]
          },
          users: { enabled: true, title: 'What is the issue?', fields: [] }
        }
      })
    )

    expect(result.success).toBe(false)
    expect(result.error.issues[0].path.join('.')).toBe(
      'config.prechat_form.visitors.fields.0.label'
    )
  })

  test('direct_to_conversation defaults to false', () => {
    expect(schema.parse(validForm).config.direct_to_conversation).toBe(false)
  })

  test('empty object', () => {
    expect(() => schema.parse({})).toThrow()
  })
})
