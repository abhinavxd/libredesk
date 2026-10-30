import * as z from 'zod'
import { getTextFromHTML } from '@shared-ui/utils/string'

const actionSchema = () => z.array(
  z.object({
    type: z.string().optional(),
    value: z.array(z.string()).optional(),
  })
)

export const createFormSchema = (i18n) => z.object({
  name: z.string().min(1, i18n._('{name} Required')),
  message_content: z.string().optional(),
  actions: actionSchema(i18n).optional().default([]),
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
      message: i18n._('Either message content or actions are required'),
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
      message: i18n._('{name} Required'),
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
      message: i18n._('{name} Required'),
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
      message: i18n._('Each action must have a type and a value'),
      // Field path to highlight
      path: ['actions'],
    }
  )
