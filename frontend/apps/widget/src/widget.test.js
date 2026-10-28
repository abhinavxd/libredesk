// @vitest-environment jsdom
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

const source = readFileSync(resolve(import.meta.dirname, '../../../../static/widget.js'), 'utf8')
let widget

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'setTimeout', 'clearTimeout', 'performance'] })
  window.history.replaceState({}, '', '/pricing')
  delete window.__libredeskWidgetLoaded
  window.eval(source)
  vi.spyOn(window.Libredesk.prototype, 'init').mockImplementation(() => {})
  widget = new window.Libredesk({ baseURL: window.location.origin, inboxID: 'test' })
  vi.spyOn(widget, 'renderPreviews').mockImplementation(() => { widget.previewHost = {} })
  vi.spyOn(widget, 'postToIframe').mockImplementation(() => {})
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
})

afterEach(() => {
  widget.stopPageTracking()
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('campaign navigation', () => {
  const deliver = (url, pageGeneration = widget.campaignPageGeneration) => {
    const contentWindow = {}
    widget.iframe = { contentWindow }
    widget.handleMessage({
      source: contentWindow,
      origin: window.location.origin,
      data: { type: 'SHOW_CAMPAIGN', delivery: { id: 'pricing-message' }, url, pageGeneration }
    })
  }

  test('drops a delayed response after navigation before the next timer tick', () => {
    const requestedURL = window.location.href
    window.history.pushState({}, '', '/checkout')

    deliver(requestedURL)

    expect(widget.campaignData).toBeNull()
    expect(widget.renderPreviews).not.toHaveBeenCalled()
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'dropped', id: 'pricing-message' })
  })

  test('shows responses for the current page', () => {
    deliver(window.location.href)

    expect(widget.campaignData).toEqual({ id: 'pricing-message' })
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'displayed', id: 'pricing-message' })
  })

  test.each(['pushState', 'replaceState'])('rejects earlier visits after returning to the same URL with %s', method => {
    widget.startPageTracking()
    const requestedURL = window.location.href
    const requestedGeneration = widget.campaignPageGeneration
    for (let visit = 0; visit < 2; visit++) {
      window.history[method]({}, '', '/checkout')
      window.history[method]({}, '', '/pricing')
    }
    widget.renderPreviews.mockClear()

    deliver(requestedURL, requestedGeneration)

    expect(widget.campaignData).toBeNull()
    expect(widget.renderPreviews).not.toHaveBeenCalled()
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'dropped', id: 'pricing-message' })

    deliver(window.location.href)

    expect(widget.campaignData).toEqual({ id: 'pricing-message' })
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'displayed', id: 'pricing-message' })
  })

  test.each([false, true])('preserves an immediate event with page tracking enabled: %s', pageTracking => {
    if (pageTracking) widget.startPageTracking()
    widget.campaignDelay = 3
    widget.startCampaignTracking()
    widget.trackEvent('old-page')
    vi.advanceTimersByTime(60000)
    widget.postToIframe.mockClear()

    window.history.pushState({}, '', '/checkout')
    widget.trackEvent('checkout-viewed')
    vi.advanceTimersByTime(3000)

    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_CONTEXT', pageGeneration: widget.campaignPageGeneration, context: {
      url: window.location.href, mobile: widget.isMobile, active_seconds: 3, event: 'checkout-viewed'
    } })
  })

  test('clears old page events and visible messages on navigation', () => {
    widget.startPageTracking()
    widget.startCampaignTracking()
    widget.trackEvent('pricing-viewed')
    widget.campaignData = { id: 'pricing-message' }

    window.history.pushState({}, '', '/checkout')
    vi.advanceTimersByTime(2000)

    expect(widget.campaignData).toBeNull()
    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_CONTEXT', pageGeneration: widget.campaignPageGeneration, context: {
      url: window.location.href, mobile: widget.isMobile, active_seconds: 2, event: ''
    } })
  })
})

describe('opening proactive messages', () => {
  beforeEach(() => {
    widget.renderPreviews.mockRestore()
    widget.widgetSettings = { launcher: { position: 'right', spacing: { side: 20, bottom: 20 } } }
    widget.createElements()
    widget.setupEventListeners()
    widget.widgetLoaded = true
    widget.previewData = {
      identity: 'visitor',
      previews: [],
      theme: { background: 'Canvas', foreground: 'CanvasText' },
      labels: { open: 'Open', dismiss: 'Dismiss' }
    }
    widget.campaignData = {
      id: 'pricing-message',
      snapshot: { sender: 'Support', message: 'Need help choosing a plan?' }
    }
    widget.renderPreviews()
  })

  afterEach(() => {
    window.removeEventListener('message', widget._boundHandleMessage)
    widget.clearPreviews()
    widget.iframe.remove()
    widget.widgetButtonWrapper.remove()
  })

  test.each(['launcher', 'bubble'])('opens the visible message through the %s once', entry => {
    const button = entry === 'launcher' ? widget.toggleButton : widget.previewHost.shadowRoot.querySelector('.open')

    button.click()

    expect(widget.postToIframe.mock.calls.map(([message]) => message)).toEqual([
      { type: 'CAMPAIGN_EVENT', event: 'opened', id: 'pricing-message' },
      { type: 'WIDGET_OPENED' }
    ])
    expect(widget.campaignData).toBeNull()
    expect(widget.previewHost).toBeNull()
    expect(widget.iframe.style.display).toBe('block')
    expect(widget.toggleButton.getAttribute('aria-expanded')).toBe('true')

    widget.showChat()
    widget.hideChat()
    widget.toggleButton.click()

    expect(widget.postToIframe.mock.calls.filter(([message]) => message.event === 'opened')).toHaveLength(1)
  })

  test('keeps normal navigation after the message is dismissed', () => {
    widget.previewHost.shadowRoot.querySelector('.close').click()
    widget.postToIframe.mockClear()

    widget.toggleButton.click()

    expect(widget.postToIframe).toHaveBeenCalledExactlyOnceWith({ type: 'WIDGET_OPENED' })
  })

  test('keeps unread replies ahead of a proactive message', () => {
    widget.updateUnreadCount(1 /* count */)

    expect(widget.previewHost).toBeNull()

    widget.toggleButton.click()

    expect(widget.postToIframe).toHaveBeenCalledExactlyOnceWith({ type: 'WIDGET_OPENED' })
  })

  test.each(['before', 'after'])('opens a reply when its unread count arrives %s its preview', order => {
    if (order === 'before') widget.updateUnreadCount(1 /* count */)
    widget.handleMessage({
      source: widget.iframe.contentWindow,
      origin: window.location.origin,
      data: {
        type: 'REPLY_PREVIEWS',
        ...widget.previewData,
        previews: [{ key: 'support-reply', conversation: 'support-thread', name: 'Support', text: 'Your answer' }]
      }
    })
    if (order === 'after') widget.updateUnreadCount(1 /* count */)
    widget.postToIframe.mockClear()

    widget.previewHost.shadowRoot.querySelector('.open').click()

    expect(widget.postToIframe.mock.calls.map(([message]) => message)).toEqual([
      { type: 'OPEN_CONVERSATION', uuid: 'support-thread' },
      { type: 'WIDGET_OPENED' }
    ])
  })

  test('restores the proactive bubble when the unread count clears', () => {
    widget.updateUnreadCount(1 /* count */)
    expect(widget.previewHost).toBeNull()

    widget.updateUnreadCount(0 /* count */)

    expect(widget.previewHost.shadowRoot.querySelector('.text').textContent).toBe('Need help choosing a plan?')

    widget.previewHost.shadowRoot.querySelector('.open').click()

    expect(widget.postToIframe).toHaveBeenCalledWith({ type: 'CAMPAIGN_EVENT', event: 'opened', id: 'pricing-message' })
  })

  test('does not open a proactive message without a visible bubble', () => {
    widget.hideLauncher = true
    widget.renderPreviews()

    widget.showChat()

    expect(widget.postToIframe).toHaveBeenCalledExactlyOnceWith({ type: 'WIDGET_OPENED' })
  })
})
