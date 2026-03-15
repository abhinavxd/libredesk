import md5 from 'blueimp-md5'

export function getGravatarUrl (email) {
  if (!email) return ''
  return `https://www.gravatar.com/avatar/${md5(email.trim().toLowerCase())}?d=404`
}
