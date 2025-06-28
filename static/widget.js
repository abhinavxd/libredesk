/**
 * Libredesk Chat Widget
 * Embeddable chat widget for websites
 */
(function () {
    'use strict';

    // Prevent multiple initializations
    if (window.LibreDeskWidget) {
        return;
    }

    class LibreDeskWidget {
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
            this.isChatVisible = false;
            this.widgetSettings = null;

            this.init();
        }

        async init () {
            try {
                await this.fetchWidgetSettings();
                this.createElements();
                this.setupEventListeners();
            } catch (error) {
                console.error('Failed to initialize LibreDesk Widget:', error);
            }
        }

        async fetchWidgetSettings () {
            try {
                const response = await fetch(`${this.config.baseUrl}/api/v1/widget/chat/settings?inbox_id=${this.config.inboxID}`);

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
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

            document.body.appendChild(this.toggleButton);
            document.body.appendChild(this.iframe);
            this.setLauncherPosition();
        }

        setLauncherPosition () {
            const launcher = this.widgetSettings.launcher;
            const spacing = launcher.spacing;
            const position = launcher.position;
            const side = position === 'right' ? 'right' : 'left';

            // Position toggle button
            this.toggleButton.style.bottom = `${spacing.bottom}px`;
            this.toggleButton.style[side] = `${spacing.side}px`;

            // Position iframe
            this.iframe.style.bottom = `${spacing.bottom + 80}px`;
            this.iframe.style[side] = `${spacing.side}px`;
        }

        setupEventListeners () {
            this.toggleButton.addEventListener('click', () => this.toggle());
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
                this.iframe.style.display = 'block';
                this.isChatVisible = true;
                this.toggleButton.style.transform = 'scale(0.9)';
            }
        }

        hideChat () {
            if (this.iframe) {
                this.iframe.style.display = 'none';
                this.isChatVisible = false;
                this.toggleButton.style.transform = 'scale(1)';
            }
        }

        destroy () {
            if (this.toggleButton) {
                document.body.removeChild(this.toggleButton);
                this.toggleButton = null;
            }
            if (this.iframe) {
                document.body.removeChild(this.iframe);
                this.iframe = null;
            }
            this.isChatVisible = false;
        }
    }

    // Global widget instance
    window.LibreDeskWidget = LibreDeskWidget;

    // Auto-initialize if configuration is provided
    if (window.libreDeskConfig) {
        window.libreDeskWidget = new LibreDeskWidget(window.libreDeskConfig);
    }

    window.initLibreDeskWidget = function (config = {}) {
        if (window.libreDeskWidget) {
            console.warn('LibreDesk Widget is already initialized');
            return window.libreDeskWidget;
        }
        window.libreDeskWidget = new LibreDeskWidget(config);
        return window.libreDeskWidget;
    };

})();