describe('PWA notification preferences', () => {
  it('serves installable assets', () => {
    cy.request('/manifest.webmanifest').then(({ body, headers, status }) => {
      expect(status).to.eq(200)
      expect(headers['content-type']).to.include('application/manifest+json')
      expect(body.name).to.eq('Libredesk')
      expect(body.display).to.eq('standalone')
      expect(body.icons).to.have.length(2)
    })

    cy.request('/sw.js').then(({ body, status }) => {
      expect(status).to.eq(200)
      expect(body).to.include("addEventListener('push'")
      expect(body).to.include("addEventListener('notificationclick'")
      expect(body).not.to.include("addEventListener('fetch'")
    })
  })
})
