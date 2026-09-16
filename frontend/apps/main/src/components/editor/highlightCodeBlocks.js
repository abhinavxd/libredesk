import hljs from 'highlight.js/lib/core'
import { codeGrammars } from './codeLanguages'

Object.entries(codeGrammars).forEach(([name, grammar]) => hljs.registerLanguage(name, grammar))

// Lowlight highlights via decorations, which tiptap never serializes.
export function highlightCodeBlocks(html) {
  if (!html || !html.includes('language-')) return html
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const blocks = doc.querySelectorAll('pre > code[class*="language-"]')
  if (blocks.length === 0) return html
  blocks.forEach((block) => {
    const language = block.className.match(/language-([\w-]+)/)?.[1]
    if (!language || !hljs.getLanguage(language)) return
    block.innerHTML = hljs.highlight(block.textContent, { language, ignoreIllegals: true }).value
  })
  return doc.body.innerHTML
}
