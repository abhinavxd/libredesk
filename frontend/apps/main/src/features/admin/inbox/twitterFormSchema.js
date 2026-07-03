import * as z from 'zod'
import { isGoDuration } from '@shared-ui/utils/string'
import { AUTH_TYPE_OAUTH2 } from '@/constants/auth.js'

export const parseTwitterStreamRules = (value = '') => {
  const trimmed = value.trim()
  if (!trimmed) return []
  return JSON.parse(trimmed)
}

export const isValidTwitterStreamRules = (value = '') => {
  try {
    const parsed = parseTwitterStreamRules(value)
    return Array.isArray(parsed) && parsed.every((rule) => (
      rule &&
      typeof rule.value === 'string' &&
      rule.value.trim().length > 0 &&
      (rule.id === undefined || typeof rule.id === 'string') &&
      (rule.tag === undefined || typeof rule.tag === 'string') &&
      (rule.enabled === undefined || typeof rule.enabled === 'boolean')
    ))
  } catch {
    return false
  }
}

export const parseTwitterActivityEventTypes = (value = '') => value
  .split(',')
  .map((item) => item.trim())
  .filter(Boolean)

export const createTwitterFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  enabled: z.boolean().optional(),
  csat_enabled: z.boolean().optional(),
  prompt_tags_on_reply: z.boolean().optional(),
  account_user_id: z.string().min(1, t('globals.messages.required')),
  screen_name: z.string().min(1, t('globals.messages.required')),
  auth_type: z.literal(AUTH_TYPE_OAUTH2),
  provider: z.enum(['official']),
  base_url: z.string().optional(),
  delivery_mode: z.enum(['polling', 'webhook', 'filtered_stream', 'activity_stream']),
  filtered_stream_rules_json: z.string().optional().refine(isValidTwitterStreamRules, {
    message: 'Enter a JSON array of stream rules.'
  }),
  webhook_id: z.string().optional(),
  webhook_url: z.string().optional(),
  webhook_consumer_secret: z.string().optional(),
  activity_subscription_id: z.string().optional(),
  activity_event_types: z.string().optional(),
  ingest_dms: z.boolean().optional(),
  ingest_mentions: z.boolean().optional(),
  poll_interval: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
    message: t('validation.invalidDuration')
  }),
  scan_since: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
    message: t('validation.invalidDuration')
  }),
  dm_cursor: z.string().optional(),
  mentions_cursor: z.string().optional(),
  oauth: z.object({
    provider: z.literal('twitter'),
    access_token: z.string().min(1, t('globals.messages.required')),
    refresh_token: z.string().optional(),
    client_id: z.string().optional(),
    client_secret: z.string().optional(),
    expires_at: z.string().optional()
  })
}).refine((data) => data.delivery_mode !== 'polling' || data.ingest_dms || data.ingest_mentions, {
  path: ['ingest_dms'],
  message: 'Enable DMs, mentions, or both when polling is selected.'
})
