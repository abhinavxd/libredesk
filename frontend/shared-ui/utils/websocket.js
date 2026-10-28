export const websocketURL = (path) => {
  const base = document.querySelector('meta[name="libredesk-ws-url"]')?.content
  return base ? base + path : path
}
