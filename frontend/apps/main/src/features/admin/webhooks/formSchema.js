import * as z from 'zod'

const discordWebhookUrl =
  /^https:\/\/((ptb|canary)\.)?discord(app)?\.com\/api(\/v\d+)?\/webhooks\/\d+\/[\w-]+(\?.*)?$/i

export const isDiscordWebhookUrl = (url) => discordWebhookUrl.test(url || '')

export const createFormSchema = (t) =>
  z
    .object({
      name: z
        .string({
          required_error: t('globals.messages.required')
        })
        .min(1, {
          message: t('globals.messages.required')
        }),
      delivery: z.enum(['http', 'discord']).default('http'),
      url: z
        .string({
          required_error: t('globals.messages.required')
        })
        .url({
          message: t('validation.invalidUrl')
        }),
      events: z.array(z.string()).min(1, {
        message: t('globals.messages.required')
      }),
      inbox_ids: z.array(z.coerce.number().int().positive()).optional().default([]),
      team_ids: z.array(z.coerce.number().int().positive()).optional().default([]),
      user_ids: z.array(z.coerce.number().int().positive()).optional().default([]),
      secret: z.string().optional(),
      is_active: z.boolean().default(true).optional(),
      headers: z.string().optional()
    })
    .superRefine((data, ctx) => {
      if (data.delivery === 'discord' && !isDiscordWebhookUrl(data.url)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['url'],
          message: t('admin.webhook.invalidDiscordURL')
        })
      }
    })
