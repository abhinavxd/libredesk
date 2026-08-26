import * as z from 'zod'

export const createFormSchema = () =>
  z.object({
    enabled: z.boolean().default(false),
    tickets_from_article_only: z.boolean().default(false),
    inbox_id: z.string().default('0'),
    help_center_id: z.string().default('0'),
    livechat_inbox_id: z.string().default('0'),
    form_id: z.string().default('0')
  })
