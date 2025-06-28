import * as z from 'zod';

export const createFormSchema = (t) => z
  .object({
    name: z.string({
      required_error: t('globals.messages.required'),
    }),
    body: z.string({
      required_error: t('globals.messages.required'),
    }),
    type: z.string().optional(),
    subject: z.string().optional(),
    is_default: z.boolean().optional().default(false),
  })
  .superRefine((data, ctx) => {
    if (data.type !== 'email_outgoing' && !data.subject) {
      ctx.addIssue({
        path: ['subject'],
        message: t('globals.messages.required'),
        code: z.ZodIssueCode.custom,
      });
    }
  });
