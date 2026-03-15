/**
 * Libredesk Chat Widget
 * Embeddable chat widget for websites
 */
(function () {
    'use strict';

    // Prevent multiple initializations
    if (window.__libredeskWidgetLoaded) {
        return;
    }
    window.__libredeskWidgetLoaded = true;

    class LibredeskWidget {
        constructor(config = {}) {
            // Validate required config
            if (!config.baseURL) {
                throw new Error('baseURL is required');
            }
            if (!config.inboxID) {
                throw new Error('inboxID is required');
            }

            this.config = config;
            this.iframe = null;
            this.toggleButton = null;
            this.widgetButtonWrapper = null;
            this.unreadBadge = null;
            this.isChatVisible = false;
            this.widgetSettings = null;
            this.unreadCount = 0;
            this.isMobile = window.innerWidth <= 600;
            this.isExpanded = false;
            this.isVueAppReady = false;
            this.hideDefaultLauncher = config.hideDefaultLauncher || false;
            this.init();
        }

        async init () {
            try {
                await this.fetchWidgetSettings();
                this.createElements();
                this.setLauncherPosition();
                // Hide widget initially until Vue app is ready
                this.widgetButtonWrapper.style.display = 'none';
                this.iframe.addEventListener('load', () => {
                    setTimeout(() => {
                        this.sendMobileState();
                    }, 2000);
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

        // Create launcher and iframe elements.
        createElements () {
            const launcher = this.widgetSettings.launcher;
            const colors = this.widgetSettings.colors;

            // Create toggle button
            this.toggleButton = document.createElement('div');
            this.toggleButton.style.cssText = `
                position: fixed;
                cursor: pointer;
                z-index: 9999;
                width: 60px;
                height: 60px;
                background-color: ${colors.primary};
                border-radius: 50%;
                display: flex;
                justify-content: center;
                align-items: center;
                box-shadow: 0 5px 20px rgba(0,0,0,0.3);
                transition: transform 0.3s ease;
            `;

            // Create icon element or arrow based on state
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

            // Create downward arrow SVG
            this.arrowIcon = document.createElement('div');
            this.arrowIcon.innerHTML = `
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <path d="M7 10L12 15L17 10" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
            `;
            this.arrowIcon.style.cssText = `
                width: 100%;
                height: 100%;
                display: none;
                justify-content: center;
                align-items: center;
            `;
            this.iconContainer.appendChild(this.arrowIcon);

            this.toggleButton.appendChild(this.iconContainer);

            // Create unread badge
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

            // Create iframe
            this.iframe = document.createElement('iframe');
            this.iframe.src = `${this.config.baseURL}/widget/?inbox_id=${this.config.inboxID}`;
            this.iframe.style.cssText = `
                position: fixed;
                border: none;
                border-radius: 12px;
                box-shadow: 0 5px 80px rgba(0,0,0,0.3);
                z-index: 9999;
                width: 400px;
                height: 700px;
                transition: all 0.3s ease;
                display: none;
            `;

            document.body.appendChild(this.widgetButtonWrapper);
            document.body.appendChild(this.iframe);
        }

        sendMobileState () {
            this.isMobile = window.innerWidth <= 600;
            // Send message to iframe to update mobile state there.
            if (this.iframe && this.iframe.contentWindow) {
                this.iframe.contentWindow.postMessage({
                    type: 'SET_MOBILE_STATE',
                    isMobile: this.isMobile
                }, '*');
            }
        }

        setLauncherPosition () {
            const launcher = this.widgetSettings.launcher;
            const spacing = launcher.spacing;
            const position = launcher.position;
            const side = position === 'right' ? 'right' : 'left';

            // Position button wrapper (which contains the toggle button and badge)
            this.widgetButtonWrapper.style.bottom = `${spacing.bottom}px`;
            this.widgetButtonWrapper.style[side] = `${spacing.side}px`;

            // Position iframe
            this.iframe.style.bottom = `${spacing.bottom + 80}px`;
            this.iframe.style[side] = `${spacing.side}px`;
        }

        setupEventListeners () {
            this.toggleButton.addEventListener('click', () => this.toggle());

            // Listen for messages from the iframe (Vue widget app)
            window.addEventListener('message', (event) => {
                // Verify the message is from our iframe.
                if (event.source === this.iframe.contentWindow) {
                    if (event.data.type === 'VUE_APP_READY') {
                        this.handleVueAppReady();
                    } else if (event.data.type === 'CLOSE_WIDGET') {
                        this.hideChat();
                    } else if (event.data.type === 'UPDATE_UNREAD_COUNT') {
                        this.updateUnreadCount(event.data.count);
                    } else if (event.data.type === 'EXPAND_WIDGET') {
                        this.expandWidget();
                    } else if (event.data.type === 'COLLAPSE_WIDGET') {
                        this.collapseWidget();
                    } else if (event.data.type === 'REQUEST_PAGE_INFO') {
                        this.iframe.contentWindow.postMessage({
                            type: 'PAGE_VISIT',
                            url: window.location.href,
                            title: document.title || ''
                        }, '*');
                    }
                }
            });
        }

        setupMobileDetection () {
            window.addEventListener('resize', () => {
                this.sendMobileState();
                if (this.isChatVisible) {
                    this.showChat();
                }
            });
            window.addEventListener('orientationchange', () => {
                this.sendMobileState();
                if (this.isChatVisible) {
                    this.showChat();
                }
            });
        }

        handleVueAppReady () {
            this.isVueAppReady = true;
            if (!this.hideDefaultLauncher) {
                this.widgetButtonWrapper.style.display = '';
            }

            // Send JWT token if provided in config
            if (this.config.userJWT) {
                this.iframe.contentWindow.postMessage({
                    type: 'SET_JWT_TOKEN',
                    jwt: this.config.userJWT
                }, '*');
            }
        }

        toggle () {
            if (this.isChatVisible) {
                this.hideChat();
                // Send WIDGET_CLOSED event to iframe
                if (this.iframe && this.iframe.contentWindow) {
                    this.iframe.contentWindow.postMessage({ type: 'WIDGET_CLOSED' }, '*');
                }
            } else {
                this.showChat();
                // Send WIDGET_OPENED event to iframe
                if (this.iframe && this.iframe.contentWindow) {
                    this.iframe.contentWindow.postMessage({ type: 'WIDGET_OPENED' }, '*');
                }
            }
        }

        showChat () {
            if (this.iframe) {
                this.isMobile = window.innerWidth <= 600;
                if (this.isMobile) {
                    this.iframe.style.display = 'block';
                    this.iframe.style.position = 'fixed';
                    this.iframe.style.top = '0';
                    this.iframe.style.left = '0';
                    this.iframe.style.width = '100vw';
                    this.iframe.style.height = '100vh';
                    this.iframe.style.borderRadius = '0';
                    this.iframe.style.boxShadow = 'none';
                    this.iframe.style.bottom = '';
                    this.iframe.style.right = '';
                    this.iframe.style.left = '';
                    this.iframe.style.top = '0';
                    this.widgetButtonWrapper.style.display = 'none';
                } else {
                    this.iframe.style.display = 'block';
                    this.iframe.style.position = 'fixed';
                    this.iframe.style.width = '400px';
                    this.iframe.style.borderRadius = '12px';
                    this.iframe.style.boxShadow = '0 5px 40px rgba(0,0,0,0.2)';
                    this.iframe.style.top = '';
                    this.iframe.style.left = '';
                    this.widgetButtonWrapper.style.display = '';

                    // Apply expanded or normal height based on current state
                    if (this.isExpanded) {
                        this.iframe.style.width = '650px';
                        this.iframe.style.height = 'calc(100vh - 110px)';
                        this.iframe.style.bottom = '90px';
                    } else {
                        this.iframe.style.height = '700px';
                        this.setLauncherPosition();
                    }
                }
                this.isChatVisible = true;
                this.toggleButton.style.transform = 'scale(0.9)';
                this.unreadBadge.style.display = 'none';

                // Switch to arrow icon
                if (this.defaultIcon) this.defaultIcon.style.display = 'none';
                this.arrowIcon.style.display = 'flex';
            }
        }

        hideChat () {
            if (this.iframe) {
                this.iframe.style.display = 'none';
                this.isChatVisible = false;
                this.toggleButton.style.transform = 'scale(1)';
                this.widgetButtonWrapper.style.display = '';

                // Switch back to default icon
                if (this.defaultIcon) this.defaultIcon.style.display = 'block';
                this.arrowIcon.style.display = 'none';
            }
        }

        updateUnreadCount (count) {
            this.unreadCount = count;

            if (count > 0 && !this.isChatVisible) {
                this.unreadBadge.textContent = count > 99 ? '99+' : count.toString();
                this.unreadBadge.style.display = 'flex';
            } else {
                this.unreadBadge.style.display = 'none';
            }
        }

        expandWidget () {
            if (this.iframe && this.isChatVisible && !this.isMobile) {
                this.isExpanded = true;

                // Expand to nearly full viewport height with gaps and wider
                this.iframe.style.width = '650px';
                this.iframe.style.height = 'calc(100vh - 110px)';
                this.iframe.style.bottom = '90px';
                this.iframe.style.maxHeight = '';

                // Send expanded state to iframe
                this.iframe.contentWindow.postMessage({
                    type: 'WIDGET_EXPANDED',
                    isExpanded: true
                }, '*');
            }
        }

        collapseWidget () {
            if (this.iframe && this.isChatVisible && !this.isMobile) {
                this.isExpanded = false;

                // Reset to original size and position
                this.iframe.style.width = '400px';
                this.iframe.style.height = '700px';
                this.iframe.style.maxHeight = '';
                this.iframe.style.top = '';

                // Restore launcher position
                this.setLauncherPosition();

                // Send collapsed state to iframe
                this.iframe.contentWindow.postMessage({
                    type: 'WIDGET_EXPANDED',
                    isExpanded: false
                }, '*');
            }
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
                setTimeout(() => {
                    if (self.iframe && self.iframe.contentWindow) {
                        self.iframe.contentWindow.postMessage({
                            type: 'PAGE_VISIT',
                            url: url,
                            title: document.title || ''
                        }, '*');
                    }
                }, 100);
            };

            // Monkey-patch history methods.
            history.pushState = function () {
                self._origPushState.apply(this, arguments);
                onPageChange();
            };
            history.replaceState = function () {
                self._origReplaceState.apply(this, arguments);
                onPageChange();
            };

            // Hash routing and browser back/forward.
            this._onPopState = onPageChange;
            this._onHashChange = onPageChange;
            window.addEventListener('popstate', this._onPopState);
            window.addEventListener('hashchange', this._onHashChange);

            // Fallback polling for edge cases.
            this._pageTrackInterval = setInterval(onPageChange, 7000);

            // Send initial page.
            onPageChange();
        }

        stopPageTracking () {
            if (this._origPushState) {
                history.pushState = this._origPushState;
            }
            if (this._origReplaceState) {
                history.replaceState = this._origReplaceState;
            }
            if (this._onPopState) {
                window.removeEventListener('popstate', this._onPopState);
            }
            if (this._onHashChange) {
                window.removeEventListener('hashchange', this._onHashChange);
            }
            if (this._pageTrackInterval) {
                clearInterval(this._pageTrackInterval);
            }
        }

        setUser (jwt) {
            if (this.iframe && this.iframe.contentWindow) {
                this.iframe.contentWindow.postMessage({
                    type: 'SET_JWT_TOKEN',
                    jwt: jwt
                }, '*');
            }
        }

        logout () {
            if (this.iframe && this.iframe.contentWindow) {
                this.iframe.contentWindow.postMessage({ type: 'CLEAR_SESSION' }, '*');
            }
        }

        destroy () {
            this.stopPageTracking();
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
        }
    }

    LibredeskWidget.prototype.show = LibredeskWidget.prototype.showChat;
    LibredeskWidget.prototype.hide = LibredeskWidget.prototype.hideChat;
    LibredeskWidget.prototype.isVisible = function () { return this.isChatVisible; };

    // Global widget instance
    window.LibredeskWidget = LibredeskWidget;

    // Auto-initialize if configuration is provided
    if (window.LibredeskConfig) {
        window.LibredeskWidget = new LibredeskWidget(window.LibredeskConfig);
    }

    window.initLibredeskWidget = function (config = {}) {
        if (window.LibredeskWidget && window.LibredeskWidget instanceof LibredeskWidget) {
            console.warn('Libredesk Widget is already initialized');
            return window.LibredeskWidget;
        }
        window.LibredeskWidget = new LibredeskWidget(config);
        return window.LibredeskWidget;
    };

})();