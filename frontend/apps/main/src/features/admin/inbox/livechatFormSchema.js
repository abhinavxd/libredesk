import { z } from 'zod'

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, { message: t('globals.messages.required') }),
  enabled: z.boolean(),
  csat_enabled: z.boolean(),
  secret: z.string(),
  linked_email_inbox_id: z.number().nullable().optional(),
  config: z.object({
    brand_name: z.string().min(1, { message: t('globals.messages.required') }),
    dark_mode: z.boolean(),
    show_powered_by: z.boolean(),
    language: z.string().min(1, { message: t('globals.messages.required') }),
    logo_url: z.string().url({
      message: t('globals.messages.invalid', {
        name: t('globals.terms.url').toLowerCase()
      })
    }).optional().or(z.literal('')),
    launcher: z.object({
      position: z.enum(['left', 'right']),
      logo_url: z.string().url({
        message: t('globals.messages.invalid', {
          name: t('globals.terms.url').toLowerCase()
        })
      }).optional().or(z.literal('')),
      spacing: z.object({
        side: z.number().min(0),
        bottom: z.number().min(0),
      })
    }),
    greeting_message: z.string().optional(),
    introduction_message: z.string().optional(),
    chat_introduction: z.string(),
    show_office_hours_in_chat: z.boolean(),
    show_office_hours_after_assignment: z.boolean(),
    notice_banner: z.object({
      enabled: z.boolean(),
      text: z.string().optional()
    }),
    colors: z.object({
      primary: z.string().regex(/^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/, {
        message: t('globals.messages.invalid', {
          name: t('globals.terms.colors').toLowerCase()
        })
      }),
    }),
    features: z.object({
      file_upload: z.boolean(),
      emoji: z.boolean(),
    }),
    trusted_domains: z.string().optional(),
    external_links: z.array(z.object({
      text: z.string().min(1),
      url: z.string().url({
        message: t('globals.messages.invalid', {
          name: t('globals.terms.url').toLowerCase()
        })
      })
    })),
    visitors: z.object({
      start_conversation_button_text: z.string(),
      allow_start_conversation: z.boolean(),
      prevent_multiple_conversations: z.boolean(),
    }),
    users: z.object({
      start_conversation_button_text: z.string(),
      allow_start_conversation: z.boolean(),
      prevent_multiple_conversations: z.boolean(),
    }),
    prechat_form: z.object({
      enabled: z.boolean(),
      title: z.string().optional(),
      fields: z.array(z.object({
        key: z.string().min(1),
        type: z.enum(['text', 'email', 'number', 'checkbox', 'date', 'link', 'list']),
        label: z.string().min(1, { message: t('globals.messages.required') }),
        placeholder: z.string().optional(),
        required: z.boolean(),
        enabled: z.boolean(),
        order: z.number().min(1),
        is_default: z.boolean(),
        custom_attribute_id: z.number().optional()
      }))
    })
  })
})
