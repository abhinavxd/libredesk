import * as z from 'zod';
import { isGoDuration } from '@shared-ui/utils/string';

export const createFormSchema = (t) => z.object({
    enabled: z.boolean().default(false),
    username: z.string().nonempty({
        message: t('globals.messages.required')
    }),
    host: z.string().nonempty({
        message: t('globals.messages.required')
    }),
    port: z
        .number({
            invalid_type_error: t('globals.messages.invalidValue', { name: t('globals.terms.port') }),
            required_error: t('globals.messages.required')
        })
        .min(1, { message: t('form.error.minmaxNumber', { min: 1, max: 65535 }) })
        .max(65535, { message: t('form.error.minmaxNumber', { min: 1, max: 65535 }) })
        .default(587),
    password: z.string().nonempty({
        message: t('globals.messages.required')
    }),
    max_conns: z
        .number({
            invalid_type_error: t('globals.messages.mustBeNumber'),
            required_error: t('globals.messages.required')
        })
        .min(1, { message: t('form.error.minmaxNumber', { min: 1, max: 1000 }) })
        .max(1000, { message: t('form.error.minmaxNumber', { min: 1, max: 1000 }) }),
    idle_timeout: z
        .string()
        .refine(isGoDuration, {
            message: t('globals.messages.goDuration')
        })
        .default('15s'),
    wait_timeout: z
        .string()
        .refine(isGoDuration, {
            message: t('globals.messages.goDuration')
        })
        .default('5s'),
    auth_protocol: z.enum(['plain', 'login', 'cram', 'none']),
    email_address: z.string().nonempty({
        message: t('globals.messages.required')
    }),
    max_msg_retries: z
        .number({
            invalid_type_error: t('globals.messages.mustBeNumber'),
            required_error: t('globals.messages.required')
        })
        .min(0, { message: t('form.error.minmaxNumber', { min: 0, max: 1000 }) })
        .max(1000, { message: t('form.error.minmaxNumber', { min: 0, max: 1000 }) })
        .default(2),
    hello_hostname: z.string().optional(),
    tls_type: z.enum(['none', 'starttls', 'tls']),
    tls_skip_verify: z.boolean().optional(),
});
