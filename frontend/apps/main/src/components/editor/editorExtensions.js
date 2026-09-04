import StarterKit from '@tiptap/starter-kit'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { createLowlight } from 'lowlight'
import { codeGrammars } from './codeLanguages'
import Placeholder from '@tiptap/extension-placeholder'
import Link from '@tiptap/extension-link'
import Mention from '@tiptap/extension-mention'
import Youtube from '@tiptap/extension-youtube'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import ResizableImage from './extensions/ResizableImage'
import { Callout } from './extensions/Callout'
import { Details, DetailsSummary, DetailsContent } from './extensions/Collapsible'
import { TrailingNode } from './extensions/TrailingNode'
import mentionSuggestion from './mentionSuggestion'

const lowlight = createLowlight(codeGrammars)

// Inline table styling so it survives email clients that strip <style>.
const tableStyle =
  'border: 1px solid #dee2e6 !important; width: 100%; margin:0; table-layout: fixed; border-collapse: collapse; position:relative; border-radius: 0.25rem;'
const tableCellStyle =
  'border: 1px solid #dee2e6 !important; box-sizing: border-box !important; min-width: 1em !important; padding: 6px 8px !important; vertical-align: top !important;'
const tableHeaderStyle =
  'background-color: #f8f9fa !important; color: #212529 !important; font-weight: bold !important; text-align: left !important; border: 1px solid #dee2e6 !important; padding: 6px 8px !important;'

const CustomTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        default: tableStyle,
        parseHTML: (element) => (element.getAttribute('style') || '') + '; ' + tableStyle
      }
    }
  }
})

const CustomTableCell = TableCell.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        default: tableCellStyle,
        parseHTML: (element) => (element.getAttribute('style') || '') + '; ' + tableCellStyle
      }
    }
  }
})

const CustomTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        default: tableHeaderStyle,
        parseHTML: (element) => (element.getAttribute('style') || '') + '; ' + tableHeaderStyle
      }
    }
  }
})

// Preserve a class attribute so links can be styled as buttons.
const CustomLink = Link.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      class: {
        default: null,
        parseHTML: (element) => element.getAttribute('class'),
        renderHTML: (attributes) => {
          if (!attributes.class) return {}
          return { class: attributes.class }
        }
      }
    }
  }
})

// text-align does nothing on the iframe, so it is mirrored onto the wrapper. parseHTML still reads the iframe copy.
const CustomYoutube = Youtube.extend({
  addOptions() {
    return { ...this.parent?.(), embedTitle: 'YouTube video' }
  },

  renderHTML(props) {
    const rendered = this.parent?.(props)
    const align = props.node.attrs.textAlign
    if (align) rendered[1] = { ...rendered[1], style: `text-align: ${align}` }
    // Without a title the embed is announced as an unnamed frame.
    const iframe = rendered[2]
    if (iframe) iframe[1] = { title: this.options.embedTitle, ...iframe[1] }
    return rendered
  }
})

// Carry a 'type' attribute to distinguish agent from team mentions.
const CustomMention = Mention.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      type: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-type'),
        renderHTML: (attributes) => {
          if (!attributes.type) return {}
          return { 'data-type': attributes.type }
        }
      }
    }
  }
})

const sharedExtensions = ({
  getPlaceholder,
  imageInline = false,
  headingLevels,
  starterKit = {}
}) => [
  StarterKit.configure({
    ...(headingLevels ? { heading: { levels: headingLevels } } : {}),
    ...starterKit
  }),
  Underline,
  ResizableImage.configure({
    inline: imageInline,
    HTMLAttributes: { class: 'inline-image', style: 'max-width: 100%; height: auto;' },
    allowBase64: false
  }),
  Placeholder.configure({ placeholder: () => getPlaceholder?.() }),
  CustomLink
]

export function buildConversationExtensions({ getPlaceholder }) {
  return [
    ...sharedExtensions({ getPlaceholder }),
    CustomMention.configure({
      HTMLAttributes: { class: 'ld-mention' },
      suggestion: mentionSuggestion
    }),
    CustomTable.configure({ resizable: false }),
    TableRow,
    CustomTableCell,
    CustomTableHeader
  ]
}

// Articles render inside their own themed CSS, so plain tables are fine here.
// Images are inline there so the paragraph text-align buttons can position them.
export function buildArticleExtensions({ getPlaceholder, embedTitle, defaultSummary }) {
  return [
    // The article title is the page's h1, so the body starts at h2.
    ...sharedExtensions({
      getPlaceholder,
      imageInline: true,
      headingLevels: [2, 3, 4],
      starterKit: { codeBlock: false }
    }),
    CodeBlockLowlight.configure({ lowlight, defaultLanguage: null }),
    Table.configure({ resizable: false }),
    TableRow,
    TableCell,
    TableHeader,
    CustomYoutube.configure({ nocookie: true, width: 640, height: 360, embedTitle }),
    TextAlign.configure({ types: ['heading', 'paragraph', 'youtube'] }),
    Callout,
    Details.configure({ defaultSummary }),
    DetailsSummary,
    DetailsContent,
    TrailingNode
  ]
}
