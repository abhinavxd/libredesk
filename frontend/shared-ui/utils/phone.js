import { z } from 'zod'

export const PHONE_COUNTRY_CODE_SUFFIX = '_country_code'
export const PHONE_NUMBER_MAX = 20
export const PHONE_COUNTRY_CODE_MAX = 10

// Accepts a national number or a full E.164/formatted number (e.g. "+1 (555) 123-4567"); requires at least one digit.
const PHONE_NUMBER_REGEX = /^\+?[\d\s()-]*\d[\d\s()-]*$/

export const countryCodeKey = (fieldKey) => `${fieldKey}${PHONE_COUNTRY_CODE_SUFFIX}`

export const phoneNumberSchema = (t, required = false) => {
  let schema = z.string().max(PHONE_NUMBER_MAX, {
    message: t('globals.messages.maxLength', { max: PHONE_NUMBER_MAX })
  })
  if (required) {
    schema = schema.min(1, { message: t('globals.messages.required') })
  }
  return schema.refine((val) => !val || PHONE_NUMBER_REGEX.test(val), {
    message: t('validation.invalidPhone')
  })
}

export const countryCodeSchema = (t, required = false) => {
  let schema = z.string().max(PHONE_COUNTRY_CODE_MAX, {
    message: t('globals.messages.maxLength', { max: PHONE_COUNTRY_CODE_MAX })
  })
  if (required) {
    schema = schema.min(1, { message: t('globals.messages.required') })
  }
  return schema
}
