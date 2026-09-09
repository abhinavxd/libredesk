self.addEventListener('push', (event) => {
  const notification = event.data.json()
  event.waitUntil(self.registration.showNotification(notification.title, {
    body: notification.body,
    tag: notification.tag,
    icon: '/images/pwa-icon-192.png',
    badge: '/images/pwa-icon-192.png',
    data: { url: notification.url }
  }))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const target = new URL(event.notification.data.url, self.location.origin)
  if (target.origin !== self.location.origin) return

  event.waitUntil(self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (clients) => {
    const client = clients[0]
    if (!client) return self.clients.openWindow(target.href)
    if ('navigate' in client) await client.navigate(target.href)
    return client.focus()
  }))
})
