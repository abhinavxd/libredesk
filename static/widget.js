/**
 * Libredesk Chat Widget
 * Embeddable chat widget for websites
 */
(function () {
    'use strict';

    // Prevent multiple initializations
    if (window.LibredeskWidget) {
        return;
    }

    class LibredeskWidget {
        constructor(config = {}) {
            // Validate required config
            if (!config.baseUrl) {
                throw new Error('baseUrl is required');
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
            this.init();
        }

        async init () {
            try {
                await this.fetchWidgetSettings();
                this.createElements();
                this.setLauncherPosition();
                this.iframe.addEventListener('load', () => {
                    setTimeout(() => {
                        this.sendMobileState();
                    }, 2000);
                });
                this.setupMobileDetection();
                this.setupEventListeners();
            } catch (error) {
                console.error('Failed to initialize Libredesk Widget:', error);
            }
        }

        async fetchWidgetSettings () {
            try {
                const response = await fetch(`${this.config.baseUrl}/api/v1/widget/chat/settings/launcher?inbox_id=${this.config.inboxID}`);

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
                box-shadow: 0 4px 12px rgba(0,0,0,0.15);
                transition: transform 0.3s ease;
            `;

            // Create icon element
            if (launcher.logo_url) {
                const icon = document.createElement('img');
                icon.src = launcher.logo_url;
                icon.style.cssText = `
                    width: 60%;
                    height: 60%;
                    filter: brightness(0) invert(1);
                `;
                this.toggleButton.appendChild(icon);
            }

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
            this.iframe.src = `${this.config.baseUrl}/widget/?inbox_id=${this.config.inboxID}`;
            this.iframe.style.cssText = `
                position: fixed;
                border: none;
                border-radius: 10px;
                box-shadow: 0 4px 20px rgba(0,0,0,0.25);
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
                    if (event.data.type === 'CLOSE_WIDGET') {
                        this.hideChat();
                    } else if (event.data.type === 'UPDATE_UNREAD_COUNT') {
                        this.updateUnreadCount(event.data.count);
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

        toggle () {
            if (this.isChatVisible) {
                this.hideChat();
            } else {
                this.showChat();
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
                    this.iframe.style.height = '700px';
                    this.iframe.style.borderRadius = '10px';
                    this.iframe.style.boxShadow = '0 4px 20px rgba(0,0,0,0.25)';
                    this.iframe.style.top = '';
                    this.iframe.style.left = '';
                    this.setLauncherPosition();
                    this.widgetButtonWrapper.style.display = '';
                }
                this.isChatVisible = true;
                this.toggleButton.style.transform = 'scale(0.9)';
                this.unreadBadge.style.display = 'none';
            }
        }

        hideChat () {
            if (this.iframe) {
                this.iframe.style.display = 'none';
                this.isChatVisible = false;
                this.toggleButton.style.transform = 'scale(1)';
                this.widgetButtonWrapper.style.display = '';
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

        destroy () {
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

    // Global widget instance
    window.LibredeskWidget = LibredeskWidget;

    // Auto-initialize if configuration is provided
    if (window.LibredeskConfig) {
        window.libreDeskWidget = new LibredeskWidget(window.LibredeskConfig);
    }

    window.initLibreDeskWidget = function (config = {}) {
        if (window.libreDeskWidget) {
            console.warn('Libredesk Widget is already initialized');
            return window.libreDeskWidget;
        }
        window.libreDeskWidget = new LibredeskWidget(config);
        return window.libreDeskWidget;
    };

})();