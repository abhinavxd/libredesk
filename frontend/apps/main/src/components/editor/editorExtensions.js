import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Link from '@tiptap/extension-link'
import Mention from '@tiptap/extension-mention'
import Underline from '@tiptap/extension-underline'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import ResizableImage from './extensions/ResizableImage'
import ActivationKey from './extensions/ActivationKey'
import mentionSuggestion from './mentionSuggestion'

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
        parseHTML: () => tableStyle
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
        parseHTML: () => tableCellStyle
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
        parseHTML: () => tableHeaderStyle
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

const sharedExtensions = ({
  getPlaceholder,
  imageInline = false,
  restrictResources = false,
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
    restrictResources,
    HTMLAttributes: { class: 'inline-image', style: 'max-width: 100%; height: auto;' },
    allowBase64: false
  }),
  Placeholder.configure({ placeholder: () => getPlaceholder?.() }),
  CustomLink
]

export function buildConversationExtensions({ getPlaceholder }) {
  return [
    ...sharedExtensions({ getPlaceholder, restrictResources: true }),
    CustomMention.configure({
      HTMLAttributes: { class: 'ld-mention' },
      suggestion: mentionSuggestion
    }),
    CustomTable.configure({ resizable: false }),
    TableRow,
    CustomTableCell,
    CustomTableHeader,
    ActivationKey
  ]
}
