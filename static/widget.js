/**
 * Libredesk Chat Widget
 * Embeddable chat widget for websites
 */
(function () {
    'use strict';

    if (window.__libredeskWidgetLoaded) {
        return;
    }
    window.__libredeskWidgetLoaded = true;
    const DEFAULT_LAUNCHER_LOGO_PATH = '/static/public/launcher-logo.png';

    class Libredesk {
        constructor(config = {}) {
            if (!config.baseURL) {
                throw new Error('baseURL is required');
            }
            if (!config.inboxID) {
                throw new Error('inboxID is required');
            }

            this.IFRAME_BORDER_RADIUS = '20px';
            this.IFRAME_BOX_SHADOW = '0 1px 6px rgba(9, 14, 21, 0.5), 0 4px 32px rgba(9, 14, 21, 0.65)';
            this.IFRAME_WIDTH = '400px';
            this.IFRAME_HEIGHT = '700px';
            this.EXPANDED_WIDTH = '750px';
            this.MOBILE_BREAKPOINT = 600;
            this.LAUNCHER_SIZE = 60;
            this.MOBILE_LAUNCHER_SIZE = 68;
            this.CAMPAIGN_POLL_MIN = 5;
            this.CAMPAIGN_POLL_MAX = 60;

            this.config = config;
            this.iframe = null;
            this.toggleButton = null;
            this.widgetButtonWrapper = null;
            this.unreadBadge = null;
            this.isChatVisible = false;
            this.widgetSettings = null;
            this.unreadCount = 0;
            this.previewData = null;
            this.previewHost = null;
            this.previewTimers = new Map();
            this.campaignData = null;
            this.campaignActiveSeconds = 0;
            this.campaignURL = location.href;
            this.campaignEvent = "";
            this.campaignBrowserKey = this.getCookie(this.getCookieName('campaign')) || this.randomKey();
            this.setCookie(this.getCookieName('campaign'), this.campaignBrowserKey);
            this.campaignSessionKey = this.getCookie(this.getCookieName('campaign-session')) || this.randomKey();
            this.setCampaignSessionCookie();
            this.isMobile = window.innerWidth <= this.MOBILE_BREAKPOINT;
            this.isExpanded = false;
            this.hideLauncher = config.hideLauncher || false;
            this.widgetLoaded = false;
            this._onShowCallback = null;
            this._onHideCallback = null;
            this._onUnreadCountChangeCallback = null;
            this._boundHandleMessage = (e) => this.handleMessage(e);
            this._boundHandleResize = () => this.handleResize();
            this.init();
        }

        // crypto.randomUUID is unavailable on plain-http host pages; getRandomValues is not.
        randomKey () {
            if (crypto.randomUUID) return crypto.randomUUID();
            const bytes = crypto.getRandomValues(new Uint8Array(16));
            bytes[6] = (bytes[6] & 0x0f) | 0x40;
            bytes[8] = (bytes[8] & 0x3f) | 0x80;
            const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('');
            return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
        }

        dropCampaign (id) {
            this.campaignData = null;
            this.postToIframe({ type: 'CAMPAIGN_EVENT', event: 'dropped', id });
        }

        postToIframe (data) {
            if (this.iframe && this.iframe.contentWindow) {
                this.iframe.contentWindow.postMessage(data, new URL(this.config.baseURL).origin);
            }
        }

        formatBadgeCount (count) {
            return count > 99 ? '99+' : count.toString();
        }

        getCookieName (type) {
            return 'libredesk-' + type + '-' + this.config.inboxID;
        }

        getCookieDomain () {
            if (this.config.cookieDomain) return this.config.cookieDomain;
            if (this._cookieDomain !== undefined) return this._cookieDomain;
            var hostname = window.location.hostname;
            if (/^(\d{1,3}\.){3}\d{1,3}$/.test(hostname) || hostname === 'localhost') {
                this._cookieDomain = '';
                return '';
            }
            var parts = hostname.split('.');
            for (var i = parts.length - 1; i >= 0; i--) {
                var domain = '.' + parts.slice(i).join('.');
                document.cookie = '__ld_test__=1;domain=' + domain + ';path=/';
                if (document.cookie.indexOf('__ld_test__') !== -1) {
                    document.cookie = '__ld_test__=;domain=' + domain + ';path=/;max-age=0';
                    this._cookieDomain = domain;
                    return domain;
                }
            }
            this._cookieDomain = '';
            return '';
        }

        setCookie (name, value) {
            var domain = this.getCookieDomain();
            var maxAge = 365 * 24 * 60 * 60;
            var cookie = name + '=' + encodeURIComponent(value) + ';path=/;max-age=' + maxAge + ';SameSite=Lax';
            if (domain) {
                cookie += ';domain=' + domain;
            }
            if (window.location.protocol === 'https:') {
                cookie += ';Secure';
            }
            document.cookie = cookie;
        }

        setCampaignSessionCookie () {
            const domain = this.getCookieDomain();
            document.cookie = this.getCookieName('campaign-session') + '=' + this.campaignSessionKey + ';path=/;SameSite=Lax' + (domain ? ';domain=' + domain : '') + (location.protocol === 'https:' ? ';Secure' : '');
        }

        getCookie (name) {
            var match = document.cookie.match(new RegExp('(?:^|; )' + name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '=([^;]*)'));
            return match ? decodeURIComponent(match[1]) : null;
        }

        deleteCookie (name) {
            var domain = this.getCookieDomain();
            var cookie = name + '=;path=/;max-age=0;SameSite=Lax';
            if (domain) {
                cookie += ';domain=' + domain;
            }
            document.cookie = cookie;
        }

        async init () {
            try {
                await this.fetchWidgetSettings();
                if (!document.body) {
                    await new Promise((resolve) => {
                        document.addEventListener('DOMContentLoaded', resolve, { once: true });
                    });
                }
                this.createElements();
                this.setLauncherPosition();
                this.widgetButtonWrapper.style.display = 'none';
                this.iframe.addEventListener('load', () => {
                    this.sendMobileState();
                });
                this.setupMobileDetection();
                this.setupEventListeners();
                this.startPageTracking();
            } catch (error) {
                console.error('Failed to initialize Libredesk Widget:', error);
            }
        }

        async fetchWidgetSettings () {
            try {
                const response = await fetch(`${this.config.baseURL}/api/v1/widget/chat/settings/launcher?inbox_id=${this.config.inboxID}`);

                if (!response.ok) {
                    throw new Error(`Error fetching widget settings. Status: ${response.status}`);
                }

                const result = await response.json();

                if (result.status !== 'success') {
                    throw new Error('Failed to fetch widget settings');
                }

                this.widgetSettings = result.data;
            } catch (error) {
                console.error('Error fetching widget settings:', error);
                throw error;
            }
        }

        contrastColor (hex) {
            try {
                hex = hex.replace(/^#/, '');
                if (hex.length === 3) {
                    hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
                }
                var toLinear = function (channel) {
                    var c = parseInt(channel, 16) / 255;
                    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
                };
                var r = toLinear(hex.substring(0, 2));
                var g = toLinear(hex.substring(2, 4));
                var b = toLinear(hex.substring(4, 6));
                // Relative luminance per WCAG, gamma-corrected so the 0.179 threshold
                // reflects perceived brightness instead of raw sRGB values.
                var L = 0.2126 * r + 0.7152 * g + 0.0722 * b;
                return L > 0.179 ? '#000000' : '#ffffff';
            } catch (e) {
                return '#ffffff';
            }
        }

        createElements () {
            const launcher = this.widgetSettings.launcher;
            const colors = this.widgetSettings.colors;

            this.toggleButton = document.createElement('div');
            this.toggleButton.style.cssText = `
                position: fixed;
                cursor: pointer;
                z-index: 9999;
                width: ${this.launcherSize()}px;
                height: ${this.launcherSize()}px;
                background-color: ${launcher.color || colors.primary};
                border-radius: 50%;
                display: flex;
                justify-content: center;
                align-items: center;
                box-shadow: 0 3px 8px rgba(9, 14, 21, 0.45), 0 14px 40px rgba(9, 14, 21, 0.55);
                transition: transform 0.3s ease;
            `;

            this.iconContainer = document.createElement('div');
            this.iconContainer.style.cssText = `
                width: 100%;
                height: 100%;
                display: flex;
                justify-content: center;
                align-items: center;
                transition: transform 0.3s ease;
            `;

            this.defaultIcon = document.createElement('img');
            this.defaultIcon.src = launcher.logo_url || (this.config.baseURL + DEFAULT_LAUNCHER_LOGO_PATH);
            this.defaultIcon.style.cssText = `
                width: 100%;
                height: 100%;
                border-radius: 50%;
                object-fit: cover;
            `;
            this.iconContainer.appendChild(this.defaultIcon);

            this.arrowIcon = document.createElement('div');
            const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            svg.setAttribute('width', '24');
            svg.setAttribute('height', '24');
            svg.setAttribute('viewBox', '0 0 24 24');
            svg.setAttribute('fill', 'none');
            const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            path.setAttribute('d', 'M7 10L12 15L17 10');
            path.setAttribute('stroke', this.contrastColor(launcher.color || colors.primary));
            path.setAttribute('stroke-width', '2');
            path.setAttribute('stroke-linecap', 'round');
            path.setAttribute('stroke-linejoin', 'round');
            svg.appendChild(path);
            this.arrowIcon.appendChild(svg);
            this.arrowIcon.style.cssText = `
                width: 100%;
                height: 100%;
                display: none;
                justify-content: center;
                align-items: center;
            `;
            this.iconContainer.appendChild(this.arrowIcon);

            this.toggleButton.appendChild(this.iconContainer);

            this.unreadBadge = document.createElement('div');
            this.unreadBadge.style.cssText = `
                position: absolute;
                top: -5px;
                right: -5px;
                background-color: #ef4444;
                color: white;
                border-radius: 50%;
                width: 20px;
                height: 20px;
                display: none;
                justify-content: center;
                align-items: center;
                font-size: 12px;
                font-weight: bold;
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                border: 2px solid white;
                box-sizing: border-box;
                z-index: 10000;
            `;

            const widgetButtonWrapper = document.createElement('div');
            widgetButtonWrapper.style.cssText = `
                position: fixed;
                z-index: 9999;
            `;

            widgetButtonWrapper.appendChild(this.toggleButton);
            widgetButtonWrapper.appendChild(this.unreadBadge);
            this.toggleButton.style.position = 'relative';
            this.widgetButtonWrapper = widgetButtonWrapper;

            const reducedMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
            const iframeTransition = reducedMotion
                ? 'none'
                : 'width 0.18s ease, height 0.18s ease, bottom 0.18s ease, border-radius 0.18s ease, box-shadow 0.18s ease';

            this.iframe = document.createElement('iframe');
            this.iframe.src = `${this.config.baseURL}/widget?inbox_id=${encodeURIComponent(this.config.inboxID)}&parent_origin=${encodeURIComponent(window.location.origin)}`;
            this.iframe.title = 'libredesk';
            this.iframe.style.cssText = `
                position: fixed;
                border: none;
                border-radius: ${this.IFRAME_BORDER_RADIUS};
                box-shadow: ${this.IFRAME_BOX_SHADOW};
                z-index: 9999;
                width: ${this.IFRAME_WIDTH};
                height: ${this.IFRAME_HEIGHT};
                transition: ${iframeTransition};
                display: none;
            `;

            document.body.appendChild(this.widgetButtonWrapper);
            document.body.appendChild(this.iframe);
        }

        sendMobileState () {
            this.isMobile = window.innerWidth <= this.MOBILE_BREAKPOINT;
            this.postToIframe({
                type: 'SET_MOBILE_STATE',
                isMobile: this.isMobile
            });
            if (this.toggleButton) {
                this.toggleButton.style.width = this.launcherSize() + 'px';
                this.toggleButton.style.height = this.launcherSize() + 'px';
            }
        }

        launcherSize () {
            return this.isMobile ? this.MOBILE_LAUNCHER_SIZE : this.LAUNCHER_SIZE;
        }

        getNormalIframeHeight () {
            const bottom = this.widgetSettings.launcher.spacing.bottom;
            return `min(${this.IFRAME_HEIGHT}, calc(100vh - ${bottom + 100}px))`;
        }

        sendPageInfo () {
            this.postToIframe({
                type: 'PAGE_VISIT',
                url: window.location.href,
                title: document.title || ''
            });
        }

        setLauncherPosition () {
            const spacing = this.widgetSettings.launcher.spacing;
            const side = this.widgetSettings.launcher.position === 'right' ? 'right' : 'left';
            this.widgetButtonWrapper.style.bottom = `${spacing.bottom}px`;
            this.widgetButtonWrapper.style[side] = `${spacing.side}px`;
        }

        applyIframeLayout () {
            if (!this.iframe) return;
            const iframe = this.iframe;

            if (this.isMobile) {
                iframe.style.top = '0';
                iframe.style.left = '0';
                iframe.style.right = '0';
                iframe.style.bottom = '0';
                iframe.style.width = '100vw';
                iframe.style.height = '100dvh';
                iframe.style.borderRadius = '0';
                iframe.style.boxShadow = 'none';
                return;
            }

            const spacing = this.widgetSettings.launcher.spacing;
            const side = this.widgetSettings.launcher.position === 'right' ? 'right' : 'left';

            iframe.style.top = '';
            iframe.style.left = '';
            iframe.style.right = '';
            iframe.style.borderRadius = this.IFRAME_BORDER_RADIUS;
            iframe.style.boxShadow = this.IFRAME_BOX_SHADOW;
            iframe.style[side] = `${spacing.side}px`;

            if (this.isExpanded) {
                iframe.style.width = this.EXPANDED_WIDTH;
                iframe.style.height = 'calc(100vh - 40px)';
                iframe.style.bottom = '20px';
            } else {
                iframe.style.width = this.IFRAME_WIDTH;
                iframe.style.height = this.getNormalIframeHeight();
                iframe.style.bottom = `${spacing.bottom + 80}px`;
            }
        }

        updateLauncherVisibility () {
            if (!this.widgetButtonWrapper) return;
            const shouldShow = this.widgetLoaded
                && !this.hideLauncher
                && !(this.isChatVisible && this.isMobile);
            this.widgetButtonWrapper.style.display = shouldShow ? '' : 'none';
        }

        handleMessage (event) {
            if (event.source !== this.iframe?.contentWindow || event.origin !== new URL(this.config.baseURL).origin || !event.data) return;

            switch (event.data.type) {
                case 'SHOW_CAMPAIGN':
                    if (this.isChatVisible || this.unreadCount > 0 || this.hideLauncher) {
                        this.dropCampaign(event.data.delivery.id);
                        break;
                    }
                    this.campaignData = event.data.delivery;
                    this.renderPreviews();
                    if (this.previewHost) this.postToIframe({ type: 'CAMPAIGN_EVENT', event: 'displayed', id: this.campaignData.id });
                    else this.dropCampaign(event.data.delivery.id);
                    break;
                case 'CLEAR_CAMPAIGN':
                    this.campaignData = null;
                    this.renderPreviews();
                    break;
                case 'REPLY_PREVIEWS':
                    if (event.data.previews?.length && this.campaignData) this.dropCampaign(this.campaignData.id);
                    this.previewData = event.data;
                    this.renderPreviews();
                    break;
                case 'VUE_APP_READY':
                    this.handleVueAppReady();
                    break;
                case 'CLOSE_WIDGET':
                    this.hideChat();
                    break;
                case 'UPDATE_UNREAD_COUNT':
                    this.updateUnreadCount(event.data.count);
                    break;
                case 'WIDGET_LOADED':
                    this.handleWidgetLoaded();
                    if (event.data.campaigns) this.startCampaignTracking();
                    break;
                case 'EXPAND_WIDGET':
                    this.expandWidget();
                    break;
                case 'COLLAPSE_WIDGET':
                    this.collapseWidget();
                    break;
                case 'REQUEST_PAGE_INFO':
                    this.sendPageInfo();
                    break;
                case 'STORE_SESSION':
                    this.setCookie(this.getCookieName('session'), event.data.token);
                    break;
                case 'STORE_VISITOR_TOKEN':
                    this.setCookie(this.getCookieName('visitor'), event.data.token);
                    break;
                case 'CLEAR_VISITOR_TOKEN':
                    this.deleteCookie(this.getCookieName('visitor'));
                    break;
                case 'CLEAR_SESSION_TOKEN':
                    this.deleteCookie(this.getCookieName('session'));
                    break;
            }
        }

        setupEventListeners () {
            this.toggleButton.addEventListener('click', () => this.toggle());
            window.addEventListener('message', this._boundHandleMessage);
        }

        handleResize () {
            const wasMobile = this.isMobile;
            this.sendMobileState();
            this.renderPreviews();
            if (this.isChatVisible && wasMobile !== this.isMobile) {
                this.applyIframeLayout();
                this.updateLauncherVisibility();
            }
        }

        setupMobileDetection () {
            window.addEventListener('resize', this._boundHandleResize);
            window.addEventListener('orientationchange', this._boundHandleResize);
        }

        handleVueAppReady () {
            this.sendMobileState();

            var visitorToken = this.getCookie(this.getCookieName('visitor'));

            if (this.config.userJWT) {
                this.postToIframe({
                    type: 'SET_JWT_TOKEN',
                    jwt: this.config.userJWT,
                    visitorToken: visitorToken || '',
                    campaignBrowserKey: this.campaignBrowserKey,
                    campaignSessionKey: this.campaignSessionKey
                });
                return;
            }

            var sessionToken = this.getCookie(this.getCookieName('session'));
            this.postToIframe({
                type: 'SESSION_DATA',
                sessionToken: sessionToken || '',
                visitorToken: visitorToken || '',
                    campaignBrowserKey: this.campaignBrowserKey,
                    campaignSessionKey: this.campaignSessionKey
            });
        }

        handleWidgetLoaded () {
            this.widgetLoaded = true;
            this.updateLauncherVisibility();
        }

        toggle () {
            if (this.isChatVisible) {
                this.hideChat();
            } else {
                this.showChat();
            }
        }

        showChat () {
            if (!this.iframe) return;

            this.isMobile = window.innerWidth <= this.MOBILE_BREAKPOINT;
            this.isChatVisible = true;
            this.renderPreviews();

            this.iframe.style.display = 'block';
            this.applyIframeLayout();
            this.updateLauncherVisibility();

            this.toggleButton.style.transform = 'scale(0.9)';
            this.unreadBadge.style.display = 'none';

            if (this.defaultIcon) this.defaultIcon.style.display = 'none';
            this.arrowIcon.style.display = 'flex';

            this.postToIframe({ type: 'WIDGET_OPENED' });

            if (this._onShowCallback) this._onShowCallback();
        }

        hideChat () {
            if (!this.iframe) return;

            this.iframe.style.display = 'none';
            this.isChatVisible = false;
            this.renderPreviews();
            this.toggleButton.style.transform = 'scale(1)';
            this.updateLauncherVisibility();

            if (this.defaultIcon) this.defaultIcon.style.display = 'block';
            this.arrowIcon.style.display = 'none';

            if (this.unreadCount > 0) {
                this.unreadBadge.textContent = this.formatBadgeCount(this.unreadCount);
                this.unreadBadge.style.display = 'flex';
            }

            this.postToIframe({ type: 'WIDGET_CLOSED' });

            if (this._onHideCallback) this._onHideCallback();
        }

        updateUnreadCount (count) {
            this.unreadCount = count;
            if (this._onUnreadCountChangeCallback) this._onUnreadCountChangeCallback(count);

            if (count > 0 && !this.isChatVisible) {
                this.unreadBadge.textContent = this.formatBadgeCount(count);
                this.unreadBadge.style.display = 'flex';
            } else {
                this.unreadBadge.style.display = 'none';
            }
        }

        expandWidget () {
            if (!this.iframe || !this.isChatVisible || this.isMobile) return;
            this.isExpanded = true;
            this.applyIframeLayout();
            this.postToIframe({ type: 'WIDGET_EXPANDED', isExpanded: true });
        }

        collapseWidget () {
            if (!this.iframe || !this.isChatVisible || this.isMobile) return;
            this.isExpanded = false;
            this.applyIframeLayout();
            this.postToIframe({ type: 'WIDGET_EXPANDED', isExpanded: false });
        }

        startPageTracking () {
            this._lastPageURL = '';
            this._origPushState = history.pushState;
            this._origReplaceState = history.replaceState;

            const self = this;
            const onPageChange = () => {
                const url = window.location.href;
                if (url === self._lastPageURL) return;
                self._lastPageURL = url;
                // Defer to let SPA frameworks update document.title after route change.
                setTimeout(() => { self.sendPageInfo(); }, 100);
            };

            history.pushState = function () {
                self._origPushState.apply(this, arguments);
                onPageChange();
            };
            history.replaceState = function () {
                self._origReplaceState.apply(this, arguments);
                onPageChange();
            };

            this._onPopState = onPageChange;
            this._onHashChange = onPageChange;
            window.addEventListener('popstate', this._onPopState);
            window.addEventListener('hashchange', this._onHashChange);

            this._pageTrackInterval = setInterval(onPageChange, 7000);
            onPageChange();
        }

        stopPageTracking () {
            if (this._origPushState) history.pushState = this._origPushState;
            if (this._origReplaceState) history.replaceState = this._origReplaceState;
            if (this._onPopState) window.removeEventListener('popstate', this._onPopState);
            if (this._onHashChange) window.removeEventListener('hashchange', this._onHashChange);
            if (this._pageTrackInterval) clearInterval(this._pageTrackInterval);
        }

        clearPreviews () {
            this.previewData = null;
            this.campaignData = null;
            this.dismissedPreviews = [];
            this.previewHost?.remove();
            this.previewHost = null;
            this.previewSignature = '';
            for (const timer of this.previewTimers.values()) clearTimeout(timer);
            this.previewTimers.clear();
        }

        dismissPreview (key) {
            clearTimeout(this.previewTimers.get(key));
            this.previewTimers.delete(key);
            if (this.campaignData?.id === key) {
                this.postToIframe({ type: 'CAMPAIGN_EVENT', event: 'dismissed', id: key });
                this.campaignData = null;
                this.renderPreviews();
                return;
            }
            const storageKey = `libredesk-previews-${this.config.inboxID}-${this.previewData?.identity || 'visitor'}`;
            let dismissed = this.dismissedPreviews || [];
            try { dismissed = JSON.parse(localStorage.getItem(storageKey) || '[]'); } catch {}
            dismissed = [...new Set([...dismissed, key])].slice(-200);
            this.dismissedPreviews = dismissed;
            try { localStorage.setItem(storageKey, JSON.stringify(dismissed)); } catch {}
            this.renderPreviews();
        }

        renderPreviews () {
            let data = this.previewData;
            const invitation = this.campaignData;
            if (invitation && this.unreadCount === 0 && data) {
                const snapshot = invitation.snapshot;
                data = { ...data, identity: 'campaign', config: { desktop: true, mobile: true, auto_hide_seconds: 0 }, previews: [{
                    key: invitation.id, campaign: true, name: snapshot.sender, avatar: snapshot.avatar,
                    text: snapshot.message.slice(0, 240),
                }] };
            }
            const enabled = data?.config?.[this.isMobile ? 'mobile' : 'desktop'];
            if (!data || !enabled || this.isChatVisible || this.hideLauncher) {
                this.previewHost?.remove();
                this.previewHost = null;
                this.previewSignature = '';
                return;
            }
            const storageKey = `libredesk-previews-${this.config.inboxID}-${data.identity || 'visitor'}`;
            let dismissed = this.dismissedPreviews || [];
            try { dismissed = JSON.parse(localStorage.getItem(storageKey) || '[]'); } catch {}
            const previews = data.previews.filter(item => !dismissed.includes(item.key)).slice(0, 3);
            const signature = JSON.stringify([previews, data.theme, data.labels, this.isMobile]);
            if (signature === this.previewSignature) return;
            this.previewSignature = signature;
            this.previewHost?.remove();
            this.previewHost = null;
            if (!previews.length) return;
            const host = document.createElement('div');
            const side = this.widgetSettings.launcher.position === 'left' ? 'left' : 'right';
            const spacing = this.widgetSettings.launcher.spacing;
            Object.assign(host.style, { position: 'fixed', zIndex: '9998', bottom: `${spacing.bottom + this.launcherSize() + 12}px`, [side]: `${spacing.side}px`, width: `min(340px, calc(100vw - ${spacing.side * 2}px))` });
            host.style.setProperty('--align', side === 'left' ? 'flex-start' : 'flex-end');
            host.style.setProperty('--muted', data.theme.muted || data.theme.foreground);
            const root = host.attachShadow({ mode: 'open' });
            const style = document.createElement('style');
            style.textContent = ':host{font:14px/1.45 system-ui,-apple-system,sans-serif;-webkit-font-smoothing:antialiased}button{font:inherit;color:inherit;cursor:pointer;border:0;background:transparent;padding:0}button:focus-visible{outline:2px solid;outline-offset:2px}.stack{display:flex;flex-direction:column;gap:10px;align-items:var(--align)}.card{position:relative;display:flex;max-width:100%;border-radius:18px;box-shadow:0 1px 3px rgba(9,14,21,.12),0 8px 28px rgba(9,14,21,.16);animation:rise .22s ease-out}.open{display:flex;gap:10px;align-items:flex-start;min-width:0;padding:12px 14px;text-align:left;border-radius:inherit}.body{display:flex;flex-direction:column;min-width:0;gap:2px}.name{font-size:12px;line-height:1.3;color:var(--muted)}.text{display:-webkit-box;-webkit-line-clamp:4;-webkit-box-orient:vertical;overflow:hidden;overflow-wrap:anywhere;white-space:pre-line}.avatar{flex:none;width:32px;height:32px;border-radius:50%;object-fit:cover}.image{margin-top:6px;max-width:160px;max-height:96px;border-radius:10px;object-fit:cover}.close{position:absolute;top:-8px;inset-inline-end:-8px;width:24px;height:24px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:15px;line-height:1;color:var(--muted);box-shadow:0 1px 3px rgba(9,14,21,.18);opacity:0;transition:opacity .15s}.card:hover .close,.card:focus-within .close{opacity:1}@media(hover:none){.close{opacity:1}}.all{font-size:12px;color:var(--muted);padding:4px 6px;border-radius:6px}.all:hover{text-decoration:underline}@keyframes rise{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:none}}@media(prefers-reduced-motion:reduce){.card{animation:none}}';
            root.append(style);
            const stack = document.createElement('div');
            stack.className = 'stack';
            const notice = document.createElement('span');
            notice.setAttribute('role', 'status');
            notice.style.cssText = 'position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)';
            notice.textContent = previews[0].text;
            stack.append(notice);
            const safeImage = (src, className) => {
                if (!src) return null;
                let url;
                try { url = new URL(src, this.config.baseURL); } catch { return null; }
                if (!['https:', 'http:'].includes(url.protocol)) return null;
                const img = document.createElement('img');
                img.src = url.href;
                img.alt = '';
                img.className = className;
                return img;
            };
            for (const item of previews) {
                const card = document.createElement('div');
                card.className = 'card';
                Object.assign(card.style, { background: data.theme.background, color: data.theme.foreground });
                const open = document.createElement('button');
                open.type = 'button';
                open.className = 'open';
                open.setAttribute('aria-label', `${data.labels.open}: ${item.name}. ${item.text}`);
                open.addEventListener('click', () => {
                    if (item.campaign) {
                        this.postToIframe({ type: 'CAMPAIGN_EVENT', event: 'opened', id: item.key });
                        this.campaignData = null;
                    } else this.postToIframe({ type: 'OPEN_CONVERSATION', uuid: item.conversation });
                    this.showChat();
                });
                const avatar = safeImage(item.avatar, 'avatar');
                if (avatar) open.append(avatar);
                const body = document.createElement('span');
                body.className = 'body';
                const name = document.createElement('span');
                name.className = 'name';
                name.textContent = item.name;
                const content = document.createElement('span');
                content.className = 'text';
                content.textContent = item.text;
                body.append(name, content);
                const image = safeImage(item.image, 'image');
                if (image) body.append(image);
                open.append(body);
                const close = document.createElement('button');
                close.type = 'button';
                close.className = 'close';
                close.textContent = '×';
                close.setAttribute('aria-label', data.labels.dismiss);
                Object.assign(close.style, { background: data.theme.background, border: `1px solid ${data.theme.border}` });
                close.addEventListener('click', () => { this.dismissPreview(item.key); this.toggleButton.focus(); });
                card.append(open, close);
                stack.append(card);
                if (data.config.auto_hide_seconds > 0 && !this.previewTimers.has(item.key)) {
                    this.previewTimers.set(item.key, setTimeout(() => this.dismissPreview(item.key), data.config.auto_hide_seconds * 1000));
                }
            }
            if (previews.length > 1) {
                const all = document.createElement('button');
                all.type = 'button';
                all.className = 'all';
                all.textContent = data.labels.dismissAll;
                all.addEventListener('click', () => { for (const item of previews) this.dismissPreview(item.key); this.toggleButton.focus(); });
                stack.append(all);
            }
            root.append(stack);
            document.body.append(host);
            this.previewHost = host;
        }

        startCampaignTracking () {
            if (this.campaignInterval) return;
            let last = performance.now();
            let elapsed = 0;
            let gap = this.CAMPAIGN_POLL_MIN;
            this.campaignInterval = setInterval(() => {
                const now = performance.now();
                const delta = Math.min((now - last) / 1000, 2);
                last = now;
                if (location.href !== this.campaignURL) {
                    this.campaignURL = location.href;
                    this.campaignActiveSeconds = 0;
                    this.campaignEvent = '';
                    elapsed = 0;
                    gap = this.CAMPAIGN_POLL_MIN;
                }
                if (document.hidden || this.isChatVisible || this.hideLauncher) return;
                this.campaignActiveSeconds += delta;
                elapsed += delta;
                if (elapsed < gap || this.unreadCount > 0 || this.campaignData) return;
                elapsed = 0;
                gap = Math.min(gap * 2, this.CAMPAIGN_POLL_MAX);
                this.postToIframe({ type: 'CAMPAIGN_CONTEXT', context: { url: location.href, mobile: this.isMobile, active_seconds: Math.floor(this.campaignActiveSeconds), event: this.campaignEvent } });
            }, 1000);
        }

        trackEvent (name) {
            if (typeof name === 'string' && name.length <= 128) this.campaignEvent = name;
        }

        setUser (jwt) {
            this.clearPreviews();
            this.postToIframe({ type: 'SET_JWT_TOKEN', jwt: jwt });
        }

        logout () {
            this.clearPreviews();
            this.deleteCookie(this.getCookieName('session'));
            this.deleteCookie(this.getCookieName('visitor'));
            this.campaignBrowserKey = this.randomKey();
            this.setCookie(this.getCookieName('campaign'), this.campaignBrowserKey);
            this.postToIframe({ type: 'CLEAR_SESSION', campaignBrowserKey: this.campaignBrowserKey });
        }

        destroy () {
            this.clearPreviews();
            clearInterval(this.campaignInterval);
            this.stopPageTracking();
            window.removeEventListener('message', this._boundHandleMessage);
            window.removeEventListener('resize', this._boundHandleResize);
            window.removeEventListener('orientationchange', this._boundHandleResize);
            if (this.widgetButtonWrapper) {
                document.body.removeChild(this.widgetButtonWrapper);
                this.widgetButtonWrapper = null;
                this.toggleButton = null;
                this.unreadBadge = null;
            }
            if (this.iframe) {
                document.body.removeChild(this.iframe);
                this.iframe = null;
            }
            this.isChatVisible = false;
            this._onShowCallback = null;
            this._onHideCallback = null;
            this._onUnreadCountChangeCallback = null;
        }
    }

    Libredesk.prototype.show = Libredesk.prototype.showChat;
    Libredesk.prototype.hide = Libredesk.prototype.hideChat;
    Libredesk.prototype.isVisible = function () { return this.isChatVisible; };
    Libredesk.prototype.onShow = function (fn) { this._onShowCallback = fn; };
    Libredesk.prototype.onHide = function (fn) { this._onHideCallback = fn; };
    Libredesk.prototype.onUnreadCountChange = function (fn) { this._onUnreadCountChangeCallback = fn; fn(this.unreadCount); };

    window.Libredesk = Libredesk;

    window.initLibredesk = function (config = {}) {
        if (window.Libredesk && window.Libredesk instanceof Libredesk) {
            console.warn('Libredesk Widget is already initialized');
            return window.Libredesk;
        }
        window.Libredesk = new Libredesk(config);
        return window.Libredesk;
    };

    function autoInit () {
        if (window.LibredeskSettings) {
            window.initLibredesk(window.LibredeskSettings);
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', autoInit, { once: true });
    } else {
        autoInit();
    }

})();
