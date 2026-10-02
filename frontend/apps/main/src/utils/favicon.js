// Points every page icon at the site logo, or back at the bundled icons when
// the logo is cleared. index.html's links are remembered as the defaults.
export function setFavicon(url, doc = document) {
  for (const link of doc.querySelectorAll('link[rel~="icon"]')) {
    if (!('defaultHref' in link.dataset)) {
      link.dataset.defaultHref = link.getAttribute('href') || ''
      link.dataset.defaultType = link.getAttribute('type') || ''
    }
    link.setAttribute('href', url || link.dataset.defaultHref)
    // A declared type the logo doesn't match would make the browser skip it.
    if (url || !link.dataset.defaultType) link.removeAttribute('type')
    else link.setAttribute('type', link.dataset.defaultType)
  }
}
