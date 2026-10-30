import * as z from 'zod'

export const createFormSchema = (i18n) => z.object({
  first_name: z
    .string({
      required_error: i18n._('{name} Required', {
        name: i18n.ngettext('First name', 'First names', 1),
      }),
    })
    .min(2, {
      message: i18n._('Must be between {min} and {max} characters', {
        min: 2,
        max: 50,
      })
    })
    .max(50, {
      message: i18n._('Must be between {min} and {max} characters', {
        min: 2,
        max: 50,
      })
    }),

  last_name: z.string().optional(),

  email: z
    .string({
      required_error: i18n._('{name} Required', {
        name: i18n.ngettext('Email', 'Emails', 1),
      }),
    })
    .email({
      message: i18n._('Invalid email address'),
    }),

  send_welcome_email: z.boolean().optional(),

  teams: z.array(z.string()).default([]),

  roles: z.array(z.string(), {
    required_error: i18n._('Please select at least one role'),
  }).min(1, i18n._('Please select at least one role')),

  new_password: z
    .string()
    .min(10, {
      message: i18n._('Password must be between {min} and {max} characters long, should contain at least one uppercase letter, one lowercase letter, one number, and one special character.', { min: 10, max: 72 })
    })
    .max(72, {
      message: i18n._('Password must be between {min} and {max} characters long, should contain at least one uppercase letter, one lowercase letter, one number, and one special character.', { min: 10, max: 72 })
    })
    .refine(val => /[a-z]/.test(val), i18n._('Password must be between {min} and {max} characters long, should contain at least one uppercase letter, one lowercase letter, one number, and one special character.', { min: 10, max: 72 }))
    .refine(val => /[A-Z]/.test(val), i18n._('Password must be between {min} and {max} characters long, should contain at least one uppercase letter, one lowercase letter, one number, and one special character.', { min: 10, max: 72 }))
    .refine(val => /\d/.test(val), i18n._('Password must be between {min} and {max} characters long, should contain at least one uppercase letter, one lowercase letter, one number, and one special character.', { min: 10, max: 72 }))
    .refine(val => /[\W_]/.test(val), i18n._('Password must be between {min} and {max} characters long, should contain at least one uppercase letter, one lowercase letter, one number, and one special character.', { min: 10, max: 72 }))
    .optional(),
  enabled: z.boolean().optional().default(true),
  availability_status: z.string().optional().default('offline'),
})
