import * as z from 'zod'
import { getTextFromHTML } from '@shared-ui/utils/string'

const actionSchema = () => z.array(
  z.object({
    type: z.string().optional(),
    value: z.array(z.string()).optional(),
  })
)

export const createFormSchema = (t) => z.object({
  name: z.string().min(1, t('globals.messages.required')),
  message_content: z.string().optional(),
  actions: actionSchema(t).optional().default([]),
  visibility: z.enum(['all', 'team', 'user']),
  visible_when: z.array(z.enum(['replying', 'starting_conversation', 'adding_private_note'])),
  team_id: z.string().nullable().optional(),
  user_id: z.string().nullable().optional(),
})
  .refine(
    (data) => {
      // Check if message_content has non-empty text after stripping HTML
      const hasMessageContent = getTextFromHTML(data.message_content || '').trim().length > 0
      // Check if actions has at least one valid action
      const hasValidActions = data.actions && data.actions.length > 0
      // Either message content or actions must be valid
      return hasMessageContent || hasValidActions
    },
    {
      message: t('admin.macro.messageOrActionRequired'),
      // Field path to highlight
      path: ['message_content'],
    }
  )
  .refine(
    (data) => {
      // If visibility is 'team', team_id is required
      if (data.visibility === 'team') {
        return !!data.team_id
      }
      return true
    },
    {
      message: t('globals.messages.required'),
      path: ['team_id'],
    }
  )
  .refine(
    (data) => {
      // If visibility is 'user', user_id is required
      if (data.visibility === 'user') {
        return !!data.user_id
      }
      return true
    },
    {
      message: t('globals.messages.required'),
      path: ['user_id'],
    }
  ).refine(
    (data) => {
      // if actions are present, all actions should have type and value defined.
      if (data.actions && data.actions.length > 0) {
        return data.actions.every(action => action.type?.length > 0 && action.value?.length > 0)
      }
      return true
    },
    {
      message: t('admin.macro.actionInvalid'),
      // Field path to highlight
      path: ['actions'],
    }
  )