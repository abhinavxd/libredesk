import StarterKit from '@tiptap/starter-kit'
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
import mentionSuggestion from './mentionSuggestion'

// Inline table styling so it survives email clients that strip <style>.
const CustomTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; border: 1px solid #dee2e6 !important; width: 100%; margin:0; table-layout: fixed; border-collapse: collapse; position:relative; border-radius: 0.25rem;'
      }
    }
  }
})

const CustomTableCell = TableCell.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; border: 1px solid #dee2e6 !important; box-sizing: border-box !important; min-width: 1em !important; padding: 6px 8px !important; vertical-align: top !important;'
      }
    }
  }
})

const CustomTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          '; background-color: #f8f9fa !important; color: #212529 !important; font-weight: bold !important; text-align: left !important; border: 1px solid #dee2e6 !important; padding: 6px 8px !important;'
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

const sharedExtensions = ({ getPlaceholder }) => [
  StarterKit.configure(),
  Underline,
  ResizableImage.configure({
    HTMLAttributes: { class: 'inline-image', style: 'max-width: 100%; height: auto;' },
    allowBase64: false
  }),
  Placeholder.configure({ placeholder: () => getPlaceholder?.() }),
  CustomLink,
  CustomMention.configure({
    HTMLAttributes: { class: 'ld-mention' },
    suggestion: mentionSuggestion
  })
]

export function buildConversationExtensions({ getPlaceholder }) {
  return [
    ...sharedExtensions({ getPlaceholder }),
    CustomTable.configure({ resizable: false }),
    TableRow,
    CustomTableCell,
    CustomTableHeader
  ]
}

// Articles render inside their own themed CSS, so plain tables are fine here.
export function buildArticleExtensions({ getPlaceholder }) {
  return [
    ...sharedExtensions({ getPlaceholder }),
    Table.configure({ resizable: false }),
    TableRow,
    TableCell,
    TableHeader,
    Youtube.configure({ nocookie: true, width: 640, height: 360 }),
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
    Callout,
    Details,
    DetailsSummary,
    DetailsContent
  ]
}
