export function convertTextToHtml (text) {
    const div = document.createElement('div')
    div.innerText = text
    return div.innerHTML.replace(/\n/g, '<br>')
}

export function parseJWT (token) {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    return JSON.parse(atob(base64))
}
