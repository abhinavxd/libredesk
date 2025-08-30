// cypress/e2e/login.cy.js

describe('Login Component', () => {
    beforeEach(() => {
        // Visit the login page
        cy.visit('/')

        // Mock the API response for OIDC providers
        cy.intercept('GET', '**/api/v1/config', {
            statusCode: 200,
            body: {
                data: {
                    "app.favicon_url": "http://localhost:9000/favicon.ico",
                    "app.lang": "en",
                    "app.logo_url": "http://localhost:9000/logo.png",
                    "app.site_name": "Libredesk",
                    "sso_providers": [
                        {
                            "client_id": "xx",
                            "enabled": true,
                            "id": 1,
                            "logo": "/images/google-logo.png",
                            "name": "Google",
                            "provider": "Google",
                            "provider_url": "https://accounts.google.com",
                            "redirect_uri": "http://localhost:9000/api/v1/oidc/1/finish"
                        }
                    ]
                }
            }
        }).as('getOIDCProviders')
    })

    it('should display login form', () => {
        cy.contains('h3', 'Libredesk').should('be.visible')
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
                    name: 'System User'
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
                    name: 'System User'
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
