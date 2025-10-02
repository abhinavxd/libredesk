/**
 * Libredesk Chat Widget
 * Embeddable chat widget for websites
 */
(function () {
    'use strict';

    // Prevent multiple initializations
    if (window.LibredeskWidget && window.LibredeskWidget instanceof Function) {
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
            this.isExpanded = false;
            this.isVueAppReady = false;
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
                box-shadow: 0 5px 20px rgba(0,0,0,0.3);
                transition: transform 0.3s ease;
            `;

            // Create icon element or arrow based on state
            this.iconContainer = document.createElement('div');
            this.iconContainer.style.cssText = `
                width: 60%;
                height: 60%;
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
            this.iframe.src = `${this.config.baseUrl}/widget/?inbox_id=${this.config.inboxID}`;
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
            // Show the widget button now that Vue app is ready
            this.widgetButtonWrapper.style.display = '';

            // Send JWT token if provided in config
            if (this.config.libredesk_user_jwt) {
                this.iframe.contentWindow.postMessage({
                    type: 'SET_JWT_TOKEN',
                    jwt: this.config.libredesk_user_jwt
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
        window.LibredeskWidget = new LibredeskWidget(window.LibredeskConfig);
    }

    window.initLibreDeskWidget = function (config = {}) {
        if (window.LibredeskWidget && window.LibredeskWidget instanceof LibredeskWidget) {
            console.warn('Libredesk Widget is already initialized');
            return window.LibredeskWidget;
        }
        window.LibredeskWidget = new LibredeskWidget(config);
        return window.LibredeskWidget;
    };

    LibredeskWidget.show = function () {
        if (window.LibredeskWidget && window.LibredeskWidget instanceof LibredeskWidget) {
            window.LibredeskWidget.showChat();
        } else {
            console.warn('Libredesk Widget is not initialized. Call initLibreDeskWidget() first.');
        }
    };

    LibredeskWidget.hide = function () {
        if (window.LibredeskWidget && window.LibredeskWidget instanceof LibredeskWidget) {
            window.LibredeskWidget.hideChat();
        } else {
            console.warn('Libredesk Widget is not initialized. Call initLibreDeskWidget() first.');
        }
    };

    LibredeskWidget.toggle = function () {
        if (window.LibredeskWidget && window.LibredeskWidget instanceof LibredeskWidget) {
            window.LibredeskWidget.toggle();
        } else {
            console.warn('Libredesk Widget is not initialized. Call initLibreDeskWidget() first.');
        }
    };

    LibredeskWidget.isVisible = function () {
        if (window.LibredeskWidget && window.LibredeskWidget instanceof LibredeskWidget) {
            return window.LibredeskWidget.isChatVisible;
        } else {
            console.warn('Libredesk Widget is not initialized. Call initLibreDeskWidget() first.');
            return false;
        }
    };

})();