import * as z from 'zod';

export const createFormSchema = (i18n) => z
  .object({
    name: z.string({
      required_error: i18n._('{name} Required'),
    }),
    body: z.string({
      required_error: i18n._('{name} Required'),
    }),
    type: z.string().optional(),
    subject: z.string().optional(),
    is_default: z.boolean().optional().default(false),
  })
  .superRefine((data, ctx) => {
    if (data.type !== 'email_outgoing' && data.name !== 'CSAT request' && !data.subject) {
      ctx.addIssue({
        path: ['subject'],
        message: i18n._('{name} Required'),
        code: z.ZodIssueCode.custom,
      });
    }
  });
