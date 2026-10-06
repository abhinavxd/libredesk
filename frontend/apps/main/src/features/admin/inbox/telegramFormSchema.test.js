import { describe, expect, it } from 'vitest'
import { createFormSchema } from './telegramFormSchema'

const schema = createFormSchema((key) => key)
const valid = () => ({
  name: 'Support',
  enabled: true,
  csat_enabled: false,
  prompt_tags_on_reply: false,
  reopen_window_hours: 48,
  config: {
    bot_token: '123:token',
    greeting_message: '',
    away_message: '',
    csat_message: '',
    business_hours_id: 0,
    timezone: 'UTC'
  }
})

describe('Telegram inbox validation', () => {
  it('accepts an inbox and preserves disabled options', () => {
    expect(schema.parse(valid())).toEqual(valid())
    const input = { ...valid(), enabled: false, reopen_window_hours: 0 }
    expect(schema.parse(input)).toEqual(input)
  })
  it('trims names and tokens and coerces whole hours', () => {
    expect(
      schema.parse({
        ...valid(),
        name: ' Support ',
        reopen_window_hours: '24',
        config: { bot_token: ' 123:token ' }
      })
    ).toEqual({ ...valid(), reopen_window_hours: 24 })
  })
  it.each([-1, -24, 0.5, 'abc', '', Infinity, NaN, 2147483648])(
    'rejects invalid reopen hours %s',
    (hours) => {
      expect(schema.safeParse({ ...valid(), reopen_window_hours: hours }).success).toBe(false)
    }
  )
  it.each(['', '   '])('rejects empty names and tokens', (value) => {
    expect(schema.safeParse({ ...valid(), name: value }).success).toBe(false)
    expect(schema.safeParse({ ...valid(), config: { bot_token: value } }).success).toBe(false)
  })
  it('accepts the masked token on edit', () => {
    expect(schema.safeParse({ ...valid(), config: { bot_token: '••••••••••' } }).success).toBe(true)
  })
})

it('requires saved business hours for away messages and preserves message settings', () => {
  const config = {
    ...valid().config,
    away_message: 'Back tomorrow',
    greeting_message: 'Hello',
    csat_message: 'How did we do?',
    timezone: 'Asia/Kolkata'
  }
  expect(schema.safeParse({ ...valid(), config }).success).toBe(false)
  config.business_hours_id = 2
  expect(schema.parse({ ...valid(), config }).config).toEqual(config)
  expect(
    schema.safeParse({ ...valid(), config: { ...config, greeting_message: 'x'.repeat(4097) } })
      .success
  ).toBe(false)
  expect(
    schema.safeParse({ ...valid(), config: { ...config, business_hours_id: -1 } }).success
  ).toBe(false)
})
