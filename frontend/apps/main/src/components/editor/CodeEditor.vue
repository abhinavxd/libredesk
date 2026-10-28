<template>
    <div ref="codeEditor" @click="editorView?.focus()" :class="readOnly ? 'w-full border rounded-md' : 'w-full h-[28rem] border rounded-md'" />
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick, useTemplateRef } from 'vue'
import { EditorView, basicSetup } from 'codemirror'
import { Compartment } from '@codemirror/state'
import { html } from '@codemirror/lang-html'
import { javascript } from '@codemirror/lang-javascript'
import { oneDark } from '@codemirror/theme-one-dark'
import { useColorMode } from '@vueuse/core'

const props = defineProps({
    modelValue: { type: String, default: '' },
    language: { type: String, default: 'html' },
    disabled: Boolean,
    readOnly: Boolean
})

const emit = defineEmits(['update:modelValue'])
const data = ref('')
const mode = useColorMode()
const theme = new Compartment()
let editorView = null
const codeEditor = useTemplateRef('codeEditor')

const initCodeEditor = (body) => {
    const langExtension = props.language === 'javascript' ? javascript() : html()
    const isEditable = !props.disabled && !props.readOnly

    editorView = new EditorView({
        doc: body,
        extensions: [
            basicSetup,
            langExtension,
            theme.of(mode.value === 'dark' ? oneDark : []),
            EditorView.editable.of(isEditable),
            EditorView.theme({
                '&': { height: props.readOnly ? 'auto' : '100%' },
                '.cm-editor': { height: props.readOnly ? 'auto' : '100%' },
                '.cm-scroller': { overflow: 'auto' }
            }),
            EditorView.updateListener.of((update) => {
                if (!update.docChanged || props.readOnly) return
                const v = update.state.doc.toString()
                emit('update:modelValue', v)
                data.value = v

            })
        ],
        parent: codeEditor.value
    })

    if (!props.readOnly) {
        nextTick(() => {
            editorView?.focus()
        })
    }
}

onMounted(() => {
    initCodeEditor(props.modelValue || '')
})

onBeforeUnmount(() => editorView?.destroy())

watch(mode, (value) => {
    editorView?.dispatch({ effects: theme.reconfigure(value === 'dark' ? oneDark : []) })
})

watch(() => props.modelValue, (newVal) => {
    if (newVal !== data.value) {
        editorView?.dispatch({
            changes: { from: 0, to: editorView.state.doc.length, insert: newVal }
        })
    }
})
</script>
