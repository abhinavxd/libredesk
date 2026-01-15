// Adds titleCase property to string.
String.prototype.titleCase = function () {
  return this.toLowerCase()
    .split(' ')
    .map(function (word) {
      return word.charAt(0).toUpperCase() + word.slice(1)
    })
    .join(' ')
}

/**
 * Replaces the `src` attribute of all <img> tags with the class `inline-image`
 * to use the value of the `title` attribute as a Content-ID (cid).
 * The resulting `src` will be in the format `cid:content_id`
 *
 * @param {string} htmlString - The input HTML string.
 * @returns {string} - The updated HTML string with `src` replaced by `cid:title`.
 */
export function transformImageSrcToCID (htmlString) {
  return htmlString.replace(/(<img\s+class="inline-image"[^>]*?src=")[^"]*(".*?title=")([^"]*)("[^>]*?>)/g, '$1cid:$3$2$3$4');
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