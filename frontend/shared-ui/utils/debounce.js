export function debounce (fn, delay) {
    let timeout
    return function (...args) {
        clearTimeout(timeout)
        timeout = setTimeout(() => fn(...args), delay)
    }
}
