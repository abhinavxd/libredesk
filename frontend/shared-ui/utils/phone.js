import { z } from 'zod'
import { countryCallingOptions } from '../constants/countries.js'

export const PHONE_COUNTRY_CODE_SUFFIX = '_country_code'
export const PHONE_NUMBER_MAX = 20
export const PHONE_COUNTRY_CODE_MAX = 10
export const DEFAULT_COUNTRY_CODE = 'US'

// Accepts a national number or a full E.164/formatted number (e.g. "+1 (555) 123-4567"); requires at least one digit.
const PHONE_NUMBER_REGEX = /^\+?[\d\s()-]*\d[\d\s()-]*$/

const VALID_COUNTRY_CODES = new Set(countryCallingOptions.map((c) => c.value))

export const countryCodeKey = (fieldKey) => `${fieldKey}${PHONE_COUNTRY_CODE_SUFFIX}`

export const defaultCountryCode = () => {
  const locales =
    typeof navigator !== 'undefined' ? [navigator.language, ...(navigator.languages || [])] : []
  for (const locale of locales) {
    if (!locale) continue
    let region
    try {
      region = new Intl.Locale(locale).region
    } catch {
      region = locale.split('-')[1]
    }
    region = region?.toUpperCase()
    if (region && VALID_COUNTRY_CODES.has(region)) return region
  }
  return DEFAULT_COUNTRY_CODE
}

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
