import { createI18n } from 'vue-i18n'

let i18n = null

export function initI18n(config) {
  i18n = createI18n(config)
  return i18n
}

export function getI18n() {
  return i18n
}
