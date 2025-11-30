import * as z from 'zod'
import { isGoDuration } from '@/utils/strings'
import { AUTH_TYPE_PASSWORD, AUTH_TYPE_OAUTH2 } from '@/constants/auth.js'

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  from: z.string().min(1, t('globals.messages.required')),
  enabled: z.boolean().optional(),
  csat_enabled: z.boolean().optional(),
  auth_type: z.enum([AUTH_TYPE_PASSWORD, AUTH_TYPE_OAUTH2]),
  oauth: z.object({
    access_token: z.string().optional(),
    client_id: z.string().optional(),
    client_secret: z.string().optional(),
    expires_at: z.string().optional(),
    provider: z.string().optional(),
    refresh_token: z.string().optional()
  }).optional(),
  imap: z.object({
    host: z.string().min(1, t('globals.messages.required')),
    port: z.number().min(1).max(65535),
    mailbox: z.string().min(1, t('globals.messages.required')),
    username: z.string().min(1, t('globals.messages.required')),
    password: z.string().min(1, t('globals.messages.required')),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
    scan_inbox_since: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('globals.messages.goDuration')
    }),
    read_interval: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('globals.messages.goDuration')
    })
  }),
  smtp: z.object({
    host: z.string().min(1, t('globals.messages.required')),
    port: z.number().min(1).max(65535),
    username: z.string().min(1, t('globals.messages.required')),
    password: z.string().min(1, t('globals.messages.required')),
    max_conns: z.number().min(1),
    max_msg_retries: z.number().min(0).max(100),
    idle_timeout: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('globals.messages.goDuration')
    }),
    wait_timeout: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('globals.messages.goDuration')
    }),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
    hello_hostname: z.string().optional(),
    auth_protocol: z.enum(['login', 'cram', 'plain', 'none'])
  })
})
