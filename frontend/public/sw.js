const readPayload = (event) => {
  try {
    return event.data ? event.data.json() : null
  } catch {
    return null
  }
}

self.addEventListener('push', (event) => {
  const notification = readPayload(event)
  if (!notification?.title) return
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
  const url = event.notification.data?.url
  if (!url) return
  const target = new URL(url, self.location.origin)
  if (target.origin !== self.location.origin) return

  event.waitUntil(self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (clients) => {
    const client = clients[0]
    if (!client) return self.clients.openWindow(target.href)
    if ('navigate' in client) await client.navigate(target.href)
    return client.focus()
  }))
})
