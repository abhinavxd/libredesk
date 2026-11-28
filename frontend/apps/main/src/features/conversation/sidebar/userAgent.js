/**
 * The conversation meta `user_agent` is either an app identifying itself in the
 * standard product-token form "Name/version (platform)" or a browser UA string.
 * Apps get a readable label, browser strings are kept as-is (the caller truncates
 * them and shows the full text in a tooltip), and machine agents show nothing.
 */

// One name/version token plus a single parenthesised platform, and nothing else.
const PRODUCT_UA = /^([A-Za-z][A-Za-z0-9 ._-]*?)\/(\d[A-Za-z0-9.+-]*)\s*\(([A-Za-z][A-Za-z0-9 ._-]*)\)$/

const MACHINE_UA = /^(curl|wget|node|node-fetch|axios|go-http-client|python-requests)(\/|$)|bot\b|crawler|spider/i

const MOBILE = /\b(android|ios|ipados|iphone|ipad|mobile)\b/i

/**
 * @returns {{label: string, isMobile: boolean}|null} null when there is nothing worth showing.
 */
export const formatUserAgent = (userAgent) => {
  const ua = String(userAgent ?? '').trim()
  if (!ua || MACHINE_UA.test(ua)) return null

  const match = ua.match(PRODUCT_UA)
  // Browser strings start with "Mozilla/5.0 (...)", which would otherwise pass as an app.
  if (match && match[1].trim().toLowerCase() !== 'mozilla') {
    const [, name, version, platform] = match
    return { label: `${name.trim()} ${version} · ${platform}`, isMobile: MOBILE.test(platform) }
  }
  return { label: ua, isMobile: MOBILE.test(ua) }
}
