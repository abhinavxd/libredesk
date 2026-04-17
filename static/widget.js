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

    class Libredesk {
        constructor(config = {}) {
            if (!config.baseURL) {
                throw new Error('baseURL is required');
            }
            if (!config.inboxID) {
                throw new Error('inboxID is required');
            }

            this.IFRAME_BORDER_RADIUS = '16px';
            this.IFRAME_BOX_SHADOW = '0 12px 48px rgba(0,0,0,0.35), 0 4px 16px rgba(0,0,0,0.25)';
            this.IFRAME_WIDTH = '400px';
            this.IFRAME_HEIGHT = '700px';
            this.EXPANDED_WIDTH = '750px';
            this.MOBILE_BREAKPOINT = 600;
            this.LAUNCHER_SIZE = 60;
            this.MOBILE_LAUNCHER_SIZE = 50;

            this.config = config;
            this.iframe = null;
            this.toggleButton = null;
            this.widgetButtonWrapper = null;
            this.unreadBadge = null;
            this.isChatVisible = false;
            this.widgetSettings = null;
            this.unreadCount = 0;
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

        postToIframe (data) {
            if (this.iframe && this.iframe.contentWindow) {
                this.iframe.contentWindow.postMessage(data, '*');
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
                var r = parseInt(hex.substring(0, 2), 16) / 255;
                var g = parseInt(hex.substring(2, 4), 16) / 255;
                var b = parseInt(hex.substring(4, 6), 16) / 255;
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
                width: ${this.isMobile ? this.MOBILE_LAUNCHER_SIZE : this.LAUNCHER_SIZE}px;
                height: ${this.isMobile ? this.MOBILE_LAUNCHER_SIZE : this.LAUNCHER_SIZE}px;
                background-color: ${launcher.color || colors.primary};
                border-radius: 50%;
                display: flex;
                justify-content: center;
                align-items: center;
                box-shadow: 0 8px 24px rgba(0,0,0,0.35), 0 2px 8px rgba(0,0,0,0.25);
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

            if (launcher.logo_url) {
                this.defaultIcon = document.createElement('img');
                this.defaultIcon.src = launcher.logo_url;
                this.defaultIcon.style.cssText = `
                    width: 100%;
                    height: 100%;
                    border-radius: 50%;
                    object-fit: cover;
                `;
                this.iconContainer.appendChild(this.defaultIcon);
            }

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
                : 'width 0.3s ease, height 0.3s ease, bottom 0.3s ease, border-radius 0.3s ease, box-shadow 0.3s ease';

            this.iframe = document.createElement('iframe');
            this.iframe.src = `${this.config.baseURL}/widget?inbox_id=${this.config.inboxID}`;
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
            this.updateLauncherSize();
            this.postToIframe({
                type: 'SET_MOBILE_STATE',
                isMobile: this.isMobile
            });
        }

        updateLauncherSize () {
            if (!this.toggleButton) return;
            const size = this.isMobile ? this.MOBILE_LAUNCHER_SIZE : this.LAUNCHER_SIZE;
            this.toggleButton.style.width = size + 'px';
            this.toggleButton.style.height = size + 'px';
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
            if (event.source !== this.iframe.contentWindow) return;

            switch (event.data.type) {
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
                    visitorToken: visitorToken || ''
                });
                return;
            }

            var sessionToken = this.getCookie(this.getCookieName('session'));
            this.postToIframe({
                type: 'SESSION_DATA',
                sessionToken: sessionToken || '',
                visitorToken: visitorToken || ''
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

        setUser (jwt) {
            this.postToIframe({ type: 'SET_JWT_TOKEN', jwt: jwt });
        }

        logout () {
            this.deleteCookie(this.getCookieName('session'));
            this.deleteCookie(this.getCookieName('visitor'));
            this.postToIframe({ type: 'CLEAR_SESSION' });
        }

        destroy () {
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
