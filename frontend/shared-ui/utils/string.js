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

/**
 * Reverts the `src` attribute of all <img> tags with the class `inline-image`
 * from the `cid:filename` format to `/uploads/filename`, where the filename is stored in the `title` attribute.
 *
 * @param {string} htmlString - The input HTML string.
 * @returns {string} - The updated HTML string with `cid:title` replaced by `/uploads/title`.
 */
export function revertCIDToImageSrc (htmlString) {
  return htmlString.replace(/(<img\s+class="inline-image"[^>]*?src=")cid:([^"]*)(".*?title=")\2("[^>]*?>)/g, '$1/uploads/$2$3$2$4');
}

export function validateEmail (email) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
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
 * Parses template variables in text and replaces them with user data.
 * Mimics Go's text/template whitespace handling - flexible with spaces/tabs inside delimiters.
 * Supports {{.FirstName}} and {{.LastName}} variables.
 *
 * @param {string} text - The text containing template variables
 * @param {Object} userData - Object containing firstName and lastName
 * @returns {string} - Text with variables replaced
 */
export function parseTemplateVariables(text, userData) {
  if (!text) return text

  return text
    .replace(/\{\{\s*\.\s*FirstName\s*\}\}/gi, userData?.firstName || '')
    .replace(/\{\{\s*\.\s*LastName\s*\}\}/gi, userData?.lastName || '')
}
