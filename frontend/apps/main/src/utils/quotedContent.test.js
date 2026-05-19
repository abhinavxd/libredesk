import { describe, test, expect } from 'vitest'
import { containsQuoteMarkers } from '@shared-ui/utils/quotedContent.js'

describe('containsQuoteMarkers', () => {
  test('returns false for null', () => {
    expect(containsQuoteMarkers(null)).toBe(false)
  })

  test('returns false for undefined', () => {
    expect(containsQuoteMarkers(undefined)).toBe(false)
  })

  test('returns false for empty string', () => {
    expect(containsQuoteMarkers('')).toBe(false)
  })

  test('returns false for plain HTML with no quote', () => {
    expect(containsQuoteMarkers('<p>Hello world</p><div>thanks</div>')).toBe(false)
  })

  test('detects <blockquote> (Gmail/Apple/Thunderbird)', () => {
    expect(containsQuoteMarkers('<blockquote>previous</blockquote>')).toBe(true)
  })

  test('detects <blockquote> with attributes', () => {
    expect(containsQuoteMarkers('<blockquote type="cite" class="gmail_quote">x</blockquote>'))
      .toBe(true)
  })

  test('detects Outlook Web appendonsend marker', () => {
    expect(containsQuoteMarkers('<div id="appendonsend"></div>')).toBe(true)
  })

  test('detects Outlook divRplyFwdMsg marker', () => {
    expect(containsQuoteMarkers('<div id="divRplyFwdMsg" dir="ltr">From:</div>')).toBe(true)
  })

  test('detects Outlook for Mac OLK_SRC_BODY_SECTION marker', () => {
    expect(containsQuoteMarkers('<div id="OLK_SRC_BODY_SECTION">body</div>')).toBe(true)
  })

  test('detects legacy Outlook OutlookMessageHeader class', () => {
    expect(containsQuoteMarkers('<div class="OutlookMessageHeader">x</div>')).toBe(true)
  })

  test('detects Yahoo Mail yahoo_quoted class', () => {
    const html = `<div id="yahoo_quoted_9284519336" class="yahoo_quoted">
      <div>On Monday, 18 May 2026, Support wrote:</div>
      <div>Original message body</div>
    </div>`
    expect(containsQuoteMarkers(html)).toBe(true)
  })

  test('detects Gmail forward gmail_quote_container', () => {
    const html = `<div class="gmail_quote gmail_quote_container">
      <div dir="ltr" class="gmail_attr">---------- Forwarded message ---------</div>
      <div>Original message body</div>
    </div>`
    expect(containsQuoteMarkers(html)).toBe(true)
  })

  test('detects real Hotmail reply payload', () => {
    const html = `<div class="elementToProof">Quoted reply!</div>
      <div id="appendonsend"></div>
      <hr style="display:inline-block;width:98%" tabindex="-1">
      <div id="divRplyFwdMsg" dir="ltr"><font face="Calibri, sans-serif">From: Libredesk
      Sent: 17 May 2026 18:12</font></div>
      <div>Original message body</div>`
    expect(containsQuoteMarkers(html)).toBe(true)
  })

  test('does not match marker word in plain prose', () => {
    expect(containsQuoteMarkers('I love the appendonsend feature in Outlook')).toBe(false)
  })

  test('does not match if id uses single quotes (edge case - Outlook always emits double)', () => {
    expect(containsQuoteMarkers(`<div id='appendonsend'></div>`)).toBe(false)
  })
})
