import * as z from 'zod'

export const createTeamFormSchema = (t) => z.object({
  name: z
    .string({
      required_error: t('globals.messages.required')
    })
    .min(2, {
      message: t('globals.messages.required')
    }),
  emoji: z.string({ required_error: t('globals.messages.required') }),
  conversation_assignment_type: z.string({ required_error: t('globals.messages.required') }),
  max_auto_assigned_conversations: z.coerce.number().optional().default(0),
  timezone: z.string({ required_error: t('globals.messages.required') }),
  business_hours_id: z.number().optional().nullable(),
  sla_policy_id: z.number().optional().nullable(),
})
