// Adds titleCase property to string.
String.prototype.titleCase = function () {
  return this.toLowerCase()
    .split(' ')
    .map(function (word) {
      return word.charAt(0).toUpperCase() + word.slice(1)
    })
    .join(' ')
}

export function convertTextToHtml (text) {
    const div = document.createElement('div')
    div.innerText = text
    return div.innerHTML.replace(/\n/g, '<br>')
}

export function parseJWT (token) {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    return JSON.parse(atob(base64))
}

// Mirrors what net/mail accepts for a bare address, so a send cannot fail at SMTP instead.
const emailRegex =
  /^[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]{2,}$/

export function validateEmail (email) {
  if (typeof email !== 'string') return false
  if (email.length > 254) return false
  return emailRegex.test(email)
}

export const isGoDuration = (value) => {
  if (value === '') return false
  const regex = /^(\d+h)?(\d+m)?(\d+s)?$/
  return regex.test(value)
}

export const isGoHourMinuteDuration = (value) => {
  const regex = /^([0-9]+h|[0-9]+m)$/
  return regex.test(value)
}

const template = document.createElement('template')
export function getTextFromHTML (htmlString) {
  try {
    template.innerHTML = htmlString
    const text = template.content.textContent || template.content.innerText || ''
    template.innerHTML = ''
    return text.trim()
  } catch (error) {
    console.error('Error converting HTML to text:', error)
    return ''
  }
}

export function getInitials (firstName = '', lastName = '') {
  const firstInitial = firstName.charAt(0).toUpperCase() || ''
  const lastInitial = lastName.charAt(0).toUpperCase() || ''
  return `${firstInitial}${lastInitial}`
}

/**
 * Replaces {{.Key}} or {{.Key | fallback}} placeholders with values from data.
 * Keys are case-insensitive. e.g. Hi {{.FirstName | there}}
 */
export function renderTemplate(text, data) {
  if (!text || !data) return text

  return text.replace(/\{\{\s*\.(\w+)(?:\s*\|\s*([^}]*))?\s*\}\}/gi, (_, key, fallback) =>
    data[Object.keys(data).find(k => k.toLowerCase() === key.toLowerCase())] || fallback?.trim() || ''
  )
}

export function isValidTemplate (text, allowedVars = []) {
  if (!text) return true
  // Strip valid variable tokens; any leftover brace means bad syntax or an unknown variable.
  const leftover = text.replace(/\{\{\s*(.*?)\s*\}\}/g, (m, v) =>
    allowedVars.includes(v.trim()) ? '' : m
  )
  return !leftover.includes('{') && !leftover.includes('}')
}
