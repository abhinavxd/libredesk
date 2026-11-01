import { z } from 'zod'
import { countryCallingOptions } from '../constants/countries.js'
import { timezoneCountries } from '../constants/timezoneCountries.js'

export const PHONE_COUNTRY_CODE_SUFFIX = '_country_code'
export const PHONE_NUMBER_MAX = 20
export const PHONE_COUNTRY_CODE_MAX = 10
export const DEFAULT_COUNTRY_CODE = 'US'

// Accepts a national number or a full E.164/formatted number (e.g. "+1 (555) 123-4567"); requires at least one digit.
const PHONE_NUMBER_REGEX = /^\+?[\d\s()-]*\d[\d\s()-]*$/

const VALID_COUNTRY_CODES = new Set(countryCallingOptions.map((c) => c.value))

export const countryCodeKey = (fieldKey) => `${fieldKey}${PHONE_COUNTRY_CODE_SUFFIX}`

export const defaultCountryCode = () => {
  return countryFromTimezone() || countryFromLanguage() || DEFAULT_COUNTRY_CODE
}

export const phoneNumberSchema = (i18n, required = false) => {
  let schema = z.string().max(PHONE_NUMBER_MAX, {
    message: i18n._('Must be at most {max} characters', { max: PHONE_NUMBER_MAX })
  })
  if (required) {
    schema = schema.min(1, { message: i18n._('{name} Required') })
  }
  return schema.refine((val) => !val || PHONE_NUMBER_REGEX.test(val), {
    message: i18n._('Invalid phone number')
  })
}

export const countryCodeSchema = (i18n, required = false) => {
  let schema = z.string().max(PHONE_COUNTRY_CODE_MAX, {
    message: i18n._('Must be at most {max} characters', { max: PHONE_COUNTRY_CODE_MAX })
  })
  if (required) {
    schema = schema.min(1, { message: i18n._('{name} Required') })
  }
  return schema
}

const countryFromTimezone = () => {
  let timezone
  try {
    timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return ''
  }
  const country = timezoneCountries[timezone]
  return country && VALID_COUNTRY_CODES.has(country) ? country : ''
}

const countryFromLanguage = () => {
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
  return ''
}
