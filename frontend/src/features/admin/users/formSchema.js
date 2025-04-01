import * as z from 'zod'

export const createFormSchema = (t) => z.object({
  first_name: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .min(2, {
      message: t('form.error.minmax', {
        min: 2,
        max: 50,
      })
    })
    .max(50, {
      message: t('form.error.minmax', {
        min: 2,
        max: 50,
      })
    }),

  last_name: z.string().optional(),

  email: z
    .string({
      required_error: t('globals.messages.required'),
    })
    .email({
      message: t('globals.messages.invalidEmailAddress'),
    }),

  send_welcome_email: z.boolean().optional(),

  teams: z.array(z.string()).default([]),

  roles: z.array(z.string()).min(1, t('globals.messages.pleaseSelectAtLeastOne', {
    name: t('globals.terms.role')
  })),

  new_password: z
    .string()
    .regex(/^(?=.*[A-Z])(?=.*[a-z])(?=.*\d)(?=.*[\W_]).{10,72}$/, {
      message: t('globals.messages.strongPassword', {
        min: 10,
        max: 72,
      })
    })
    .optional(),
  enabled: z.boolean().optional().default(true)
})
