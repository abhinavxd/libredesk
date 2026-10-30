import * as z from 'zod'

export const createTeamFormSchema = (i18n) => z.object({
  name: z
    .string({
      required_error: i18n._('{name} Required')
    })
    .min(2, {
      message: i18n._('{name} Required')
    }),
  emoji: z.string({ required_error: i18n._('{name} Required') })
    .trim().min(1, { message: i18n._('{name} Required') }),
  conversation_assignment_type: z.string({ required_error: i18n._('{name} Required') }),
  max_auto_assigned_conversations: z.coerce.number().default(0),
  timezone: z.string({ required_error: i18n._('{name} Required') }),
  business_hours_id: z.number().optional().nullable(),
  sla_policy_id: z.number().optional().nullable(),
})
