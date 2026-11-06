import * as z from 'zod'
import { isGoDuration, validateEmail, isValidTemplate } from '@shared-ui/utils/string'
import { AUTH_TYPE_PASSWORD, AUTH_TYPE_OAUTH2, AUTH_TYPE_OAUTH2_APP } from '@main/constants/auth.js'
import { extractEmailAddress } from '@main/utils/email-sender'

export const FROM_NAME_TEMPLATE_VARS = ['.Agent.FirstName', '.Agent.LastName', '.Agent.FullName', '.Inbox.Name']
// Multi-tenant authorities cannot issue app-only tokens for a specific organisation.
const SHARED_TENANTS = ['common', 'organizations', 'consumers']

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  from: z.string().min(1, t('globals.messages.required')),
  aliases: z.array(z.object({
    email: z.string().refine(validateEmail, { message: t('validation.invalidEmail') }),
    verification_status: z.string().optional()
  })).optional().default([]),
  from_name_template: z
    .string()
    .optional()
    .default('')
    .refine((val) => isValidTemplate(val, FROM_NAME_TEMPLATE_VARS), {
      message: t('admin.inbox.fromNameTemplate.invalidTemplate')
    }),
  reply_to: z
    .string()
    .optional()
    .refine((v) => !v || validateEmail(v), {
      message: t('validation.invalidEmail')
    }),
  enabled: z.boolean().optional(),
  csat_enabled: z.boolean().optional(),
  prompt_tags_on_reply: z.boolean().optional(),
  enable_plus_addressing: z.boolean().optional(),
  auth_type: z.enum([AUTH_TYPE_PASSWORD, AUTH_TYPE_OAUTH2, AUTH_TYPE_OAUTH2_APP]),
  oauth: z.object({
    access_token: z.string().optional(),
    client_id: z.string().optional(),
    client_secret: z.string().optional(),
    expires_at: z.string().optional(),
    provider: z.string().optional(),
    refresh_token: z.string().optional(),
    tenant_id: z.string().optional()
  }).optional(),
  imap: z.object({
    host: z.string().min(1, t('globals.messages.required')),
    port: z.number().min(1).max(65535),
    mailbox: z.string().min(1, t('globals.messages.required')),
    username: z.string().min(1, t('globals.messages.required')),
    password: z.string().default(''),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
    scan_inbox_since: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('validation.invalidDuration')
    }),
    read_interval: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('validation.invalidDuration')
    })
  }),
  smtp: z.object({
    host: z.string().min(1, t('globals.messages.required')),
    port: z.number().min(1).max(65535),
    username: z.string().default(''),
    password: z.string().default(''),
    max_conns: z.number().min(1),
    max_msg_retries: z.number().min(0).max(100),
    idle_timeout: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('validation.invalidDuration')
    }),
    pool_wait_timeout: z.string().min(1, t('globals.messages.required')).refine(isGoDuration, {
      message: t('validation.invalidDuration')
    }),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
    hello_hostname: z.string().optional(),
    auth_protocol: z.enum(['login', 'cram', 'plain', 'none'])
  })
}).superRefine((values, ctx) => {
  const required = (path) => ctx.addIssue({ code: 'custom', path, message: t('globals.messages.required') })
  if (values.auth_type === AUTH_TYPE_PASSWORD) {
    if (!values.imap.password) required(['imap', 'password'])
    if (values.smtp.auth_protocol !== 'none') {
      for (const field of ['username', 'password']) {
        if (!values.smtp[field]) required(['smtp', field])
      }
    }
  } else if (values.auth_type === AUTH_TYPE_OAUTH2_APP) {
    if (!values.smtp.username) required(['smtp', 'username'])
    for (const field of ['tenant_id', 'client_id', 'client_secret']) {
      if (!values.oauth?.[field]?.trim()) required(['oauth', field])
    }
    if (SHARED_TENANTS.includes(values.oauth?.tenant_id?.trim().toLowerCase())) {
      ctx.addIssue({ code: 'custom', path: ['oauth', 'tenant_id'], message: t('admin.inbox.oauth.appOnly.tenantRequired') })
    }
  }

  const primary = extractEmailAddress(values.from).toLowerCase()
  const seen = new Set([primary])
  for (const [index, value] of (values.aliases || []).entries()) {
    const alias = value.email.trim().toLowerCase()
    if (seen.has(alias)) {
      ctx.addIssue({ code: 'custom', path: ['aliases', index, 'email'], message: t('globals.messages.errorAlreadyExists') })
    }
    seen.add(alias)
  }
})
