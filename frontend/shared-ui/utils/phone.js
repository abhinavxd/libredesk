import { z } from 'zod'

export const PHONE_COUNTRY_CODE_SUFFIX = '_country_code'
export const PHONE_NUMBER_MAX = 15
export const PHONE_COUNTRY_CODE_MAX = 10

export const countryCodeKey = (fieldKey) => `${fieldKey}${PHONE_COUNTRY_CODE_SUFFIX}`

export const phoneNumberSchema = (t) =>
  z
    .string()
    .refine((val) => !val || /^\d{1,15}$/.test(val), {
      message: t('validation.minmax', { min: 1, max: PHONE_NUMBER_MAX })
    })

export const countryCodeSchema = (t) =>
  z.string().max(PHONE_COUNTRY_CODE_MAX, {
    message: t('globals.messages.maxLength', { max: PHONE_COUNTRY_CODE_MAX })
  })
