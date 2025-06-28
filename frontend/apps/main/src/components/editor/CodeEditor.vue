<template>
    <div ref="codeEditor" id="code-editor" class="code-editor" />
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import CodeFlask from 'codeflask'

const props = defineProps({
    modelValue: { type: String, default: '' },
    language: { type: String, default: 'html' },
    disabled: Boolean
})

const emit = defineEmits(['update:modelValue'])
const codeEditor = ref(null)
const data = ref('')
const flask = ref(null)

const initCodeEditor = (body) => {
    const el = document.createElement('code-flask')
    el.attachShadow({ mode: 'open' })
    el.shadowRoot.innerHTML = `
      <style>
        .codeflask .codeflask__flatten {
          font-size: 15px;
          white-space: pre-wrap;
          word-break: break-word;
        }
        .codeflask .codeflask__lines { background: #fafafa; z-index: 10; }
        .codeflask .token.tag { font-weight: bold; }
        .codeflask .token.attr-name { color: #111; }
        .codeflask .token.attr-value { color: #000 !important; }
      </style>
      <div id="area"></div>
    `
    codeEditor.value.appendChild(el)

    flask.value = new CodeFlask(el.shadowRoot.getElementById('area'), {
        language: props.language,
        lineNumbers: false,
        styleParent: el.shadowRoot,
        readonly: props.disabled
    })

    flask.value.onUpdate((v) => {
        emit('update:modelValue', v)
        data.value = v
    })

    flask.value.updateCode(body)

    nextTick(() => {
        document.querySelector('code-flask').shadowRoot.querySelector('textarea').focus()
    })
}

onMounted(() => {
    initCodeEditor(props.modelValue || '')
})

watch(() => props.modelValue, (newVal) => {
    if (newVal !== data.value) {
        flask.value.updateCode(newVal)
    }
})
</script>