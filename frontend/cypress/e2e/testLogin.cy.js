const existingInstallation = Cypress.env('FRESH_INSTALL') ? describe.skip : describe
const freshInstallation = Cypress.env('FRESH_INSTALL') ? describe : describe.skip

existingInstallation('Login Component', () => {
    beforeEach(() => {
        // Mock the API response for OIDC providers
        cy.intercept('GET', '**/api/v1/config', {
            statusCode: 200,
            body: {
                data: {
                    "app.favicon_url": "http://localhost:9000/favicon.ico",
                    "app.lang": "en",
                    "app.logo_url": "http://localhost:9000/logo.png",
                    "app.site_name": "libredesk",
                    "app.sso_providers": [
                        {
                            "client_id": "xx",
                            "enabled": true,
                            "id": 1,
                            "logo_url": "/images/google-logo.svg",
                            "name": "Google",
                            "provider": "Google",
                            "provider_url": "https://accounts.google.com",
                            "redirect_uri": "http://localhost:9000/api/v1/oidc/1/finish"
                        }
                    ]
                }
            }
        }).as('getOIDCProviders')

        // Visit the login page
        cy.visit('/')
        cy.get('#email').clear()
        cy.get('#password').clear()
    })

    it('should display login form', () => {
        cy.contains('h3', 'libredesk').should('be.visible')
        cy.contains('p', 'Sign in to your account').should('be.visible')
        cy.get('#email').should('be.visible')
        cy.get('#password').should('be.visible')
        cy.contains('a', 'Forgot password?').should('be.visible')
        cy.contains('button', 'Sign in').should('be.visible')
    })

    it('should display OIDC providers when loaded', () => {
        cy.wait('@getOIDCProviders')
        cy.contains('button', 'Google').should('be.visible')
        cy.contains('div', 'Or continue with').should('be.visible')
    })

    it('should show error for invalid login attempt', () => {
        // Mock failed login API call
        cy.intercept('POST', '**/api/v1/auth/login', {
            statusCode: 401,
            body: {
                message: 'Invalid credentials'
            }
        }).as('loginFailure')

        // Enter System username and wrong password
        cy.get('#email').type('System')
        cy.get('#password').type('WrongPassword')

        // Submit form
        cy.contains('button', 'Sign in').click()

        // Wait for API call
        cy.wait('@loginFailure')

        // Verify error message appears
        cy.contains('Invalid credentials').should('be.visible')
    })

    it('should login successfully with correct credentials', () => {
        // Mock successful login API call
        cy.intercept('POST', '**/api/v1/auth/login', {
            statusCode: 200,
            body: {
                data: {
                    id: 1,
                    email: 'System',
                    first_name: 'System',
                    last_name: 'User',
                    permissions: [],
                    roles: [],
                    teams: []
                }
            }
        }).as('loginSuccess')

        // Enter System username and correct password
        cy.get('#email').type('System')
        cy.get('#password').type('StrongPass!123')

        // Submit form
        cy.contains('button', 'Sign in').click()

        // Wait for API call
        cy.wait('@loginSuccess')

        // Verify redirection to inboxes page
        cy.url().should('include', '/inboxes/assigned')
    })

    it('should validate email format', () => {
        // Enter invalid email and a password
        cy.get('#email').type('invalid-email')
        cy.get('#password').type('password')

        // Submit form
        cy.contains('button', 'Sign in').click()

        // Check for validation error (matching the error message with a trailing period)
        cy.contains('Invalid email address').should('be.visible')
    })

    it('should validate empty password', () => {
        // Enter email but no password
        cy.get('#email').type('valid@example.com')

        // Submit form
        cy.contains('button', 'Sign in').click()

        // Check for validation error (matching the error message with a trailing period)
        cy.contains('Password cannot be empty').should('be.visible')
    })

    it('should show loading state during login', () => {
        // Mock slow API response
        cy.intercept('POST', '**/api/v1/auth/login', {
            statusCode: 200,
            body: {
                data: {
                    id: 1,
                    email: 'System',
                    first_name: 'System',
                    last_name: 'User',
                    permissions: [],
                    roles: [],
                    teams: []
                }
            },
            delay: 1000
        }).as('slowLogin')

        // Enter credentials
        cy.get('#email').type('System')
        cy.get('#password').type('StrongPass!123')

        // Submit form
        cy.contains('button', 'Sign in').click()

        // Check if loading state is shown
        cy.contains('Logging in...').should('be.visible')
        cy.get('.animate-spin').should('be.visible')

        // Wait for API call to finish
        cy.wait('@slowLogin')
    })
})

existingInstallation('Logout', () => {
    beforeEach(() => {
        cy.viewport(1440, 900)
        cy.login()
        cy.visit('/inboxes/assigned')
    })

    const clickLogout = () => {
        cy.contains('button', 'System').click()
        cy.contains('[role="menuitem"]', /log\s*out/i).click()
    }

    it('keeps the session active when logout fails', () => {
        cy.intercept('POST', '**/api/v1/auth/logout', {
            statusCode: 500,
            body: { message: 'Logout failed. Try again.' }
        }).as('logoutFailure')
        clickLogout()
        cy.wait('@logoutFailure')
        cy.contains('Logout failed. Try again.').should('be.visible')
        cy.location('pathname').should('eq', '/inboxes/assigned')
        cy.request('/api/v1/agents/me').its('status').should('eq', 200)
    })

    it('requires a CSRF token and ignores the old GET logout route', () => {
        cy.request({
            method: 'POST',
            url: '/api/v1/auth/logout',
            failOnStatusCode: false
        }).its('status').should('eq', 403)
        cy.request({ url: '/logout', failOnStatusCode: false })
        cy.request('/api/v1/agents/me').its('status').should('eq', 200)
    })

    it('preserves the login when an SSO provider is unavailable', () => {
        cy.getCookie('libredesk_session').then((cookie) => {
            cy.request({ url: '/api/v1/oidc/999999/login', followRedirect: false })
                .its('status').should('eq', 302)
            cy.getCookie('libredesk_session').its('value').should('eq', cookie.value)
        })
        cy.request('/api/v1/agents/me').its('status').should('eq', 200)
    })

    it('destroys the session before returning to the login page', () => {
        cy.intercept('POST', '**/api/v1/auth/logout').as('logout')
        clickLogout()
        cy.wait('@logout').its('response.statusCode').should('eq', 200)
        cy.location('pathname').should('eq', '/')
        cy.get('#email').should('be.visible')
        cy.request({
            url: '/api/v1/agents/me',
            failOnStatusCode: false
        }).its('status').should('eq', 401)
    })
})

existingInstallation('Existing installation setup', () => {
    it('keeps setup closed and preserves sign in on an existing installation', () => {
        cy.request('/api/v1/setup').its('body.data.required').should('eq', false)
        cy.visit('/setup')
        cy.location('pathname').should('eq', '/')
        cy.get('#email').should('be.visible')
        cy.login()
        cy.request('/api/v1/agents/me').its('status').should('eq', 200)
    })
})

freshInstallation('Fresh installation setup', () => {
    it('creates the first admin and completes setup after a delivered reply', () => {
        cy.viewport(1440, 1000)
        const code = Cypress.env('SETUP_CODE')
        const password = 'SetupPass!123'
        const api = (method, path, body) => cy.getCookie('csrf_token').then((cookie) =>
            cy.request({ method, url: path, body, headers: { 'X-CSRFTOKEN': cookie.value } })
        )
        cy.visit('/')
        cy.location('pathname').should('eq', '/setup')
        cy.get('#token').type('x'.repeat(40), { log: false })
        cy.get('#first_name').type('Setup Admin')
        cy.get('#email').type('setup-admin@example.com')
        cy.get('#password').type(password, { log: false })
        cy.get('#confirmation').type(password, { log: false })
        cy.get('[role="alert"]').should('not.exist')
        cy.contains('button', 'Create admin account').click()
        cy.contains('[role="alert"]', 'Setup code is invalid or expired').should('be.visible')
        cy.get('#token').clear()
        cy.get('#token').type(code, { log: false })
        cy.get('#confirmation').type('{enter}', { log: false })
        cy.location('pathname').should('eq', '/')
        cy.get('#email').type('setup-admin@example.com')
        cy.get('#password').type(`${password}{enter}`, { log: false })
        cy.location('pathname').should('eq', '/inboxes/assigned')
        cy.contains('Check your workspace settings').should('be.visible')

        cy.visit('/admin/general')
        cy.get('form button[type="submit"]').click()
        cy.get('form button[type="submit"]').should('not.be.disabled')
        cy.request('/api/v1/setup/checklist').its('body.data.workspace').should('eq', true)

        cy.visit('/admin/inboxes/new')
        cy.contains('button', 'Live Chat').click()
        cy.get('[data-section="basics"] > h3 > button').click()
        cy.get('input[name="name"]').type('Setup test inbox')
        cy.get('input[name="config.brand_name"]').type('Support')
        cy.get('form button[type="submit"]').click()
        cy.location('pathname').should('eq', '/admin/inboxes')
        api('GET', '/api/v1/inboxes').then(({ body }) => {
            const inbox = body.data.find((item) => item.name === 'Setup test inbox')
            expect(inbox).to.have.property('uuid')
            cy.request({
                method: 'POST',
                url: '/api/v1/widget/chat/conversations/init',
                headers: { 'X-Libredesk-Inbox-ID': inbox.uuid },
                body: { message: 'Can you help me finish setup?' }
            }).then(({ body }) => {
                const conversation = body.data.conversation.uuid
                const token = body.data.session_token
                api('POST', `/api/v1/conversations/${conversation}/messages`, {
                    message: 'Setup is working.', private: false, sender_type: 'agent'
                })
                cy.visit('/inboxes/assigned')
                cy.contains('button', 'Check progress').click()
                cy.contains('button', 'Skip for now').click()
                cy.contains('Complete your setup').should('not.exist')
                cy.request({
                    url: `/api/v1/widget/chat/conversations/${conversation}`,
                    headers: {
                        Authorization: `Bearer ${token}`,
                        'X-Libredesk-Inbox-ID': inbox.uuid
                    }
                }).then(({ body }) => {
                    expect(JSON.stringify(body.data.messages)).to.include('Setup is working.')
                })
            })
        })
        cy.request('/api/v1/setup').its('body.data.required').should('eq', false)
        cy.request({
            method: 'POST', url: '/api/v1/setup/account', failOnStatusCode: false,
            body: { token: code, first_name: 'Second admin', email: 'second@example.com', password }
        }).its('status').should('eq', 409)
        cy.visit('/setup')
        cy.location('pathname').should('eq', '/inboxes/assigned')
    })
})
