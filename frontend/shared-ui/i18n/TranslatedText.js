import { defineComponent } from 'vue'
import { useI18n } from './index'

// Render named slots as Vue nodes, never as HTML from a catalog or user input.
export default defineComponent({
  name: 'TranslatedText',
  props: { message: { type: Object, required: true } },
  setup(props, { slots }) {
    const i18n = useI18n()
    return () => {
      const markers = Object.fromEntries(Object.keys(slots).map(name => [name, `{${name}}`]))
      return i18n._(props.message, markers).split(/(\{\w+\})/g).flatMap(part => {
        const name = /^\{(\w+)\}$/.exec(part)?.[1]
        return name && slots[name] ? slots[name]() : part
      })
    }
  },
})
