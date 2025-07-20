import { z } from 'zod'

export const createVisitorInfoSchema = (t, requireContactInfo) => {
  const baseSchema = {
    name: z.string().optional(),
    email: z.string().optional()
  }

  if (requireContactInfo === 'required') {
    return z.object({
      name: z
        .string({
          required_error: t('globals.messages.required', { name: t('globals.terms.name') }),
        })
        .min(1, {
          message: t('globals.messages.required', { name: t('globals.terms.name') }),
        }),
      email: z
        .string({
          required_error: t('globals.messages.required', { name: t('globals.terms.email') }),
        })
        .min(1, {
          message: t('globals.messages.required', { name: t('globals.terms.email') }),
        })
        .email({
          message: t('globals.messages.invalidEmail'),
        }),
    })
  } else if (requireContactInfo === 'optional') {
    return z.object({
      name: z.string().optional(),
      email: z
        .string()
        .optional()
        .refine(val => !val || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(val), {
          message: t('globals.messages.invalidEmail'),
        }),
    })
  }

  // Disabled mode - no validation
  return z.object(baseSchema)
}