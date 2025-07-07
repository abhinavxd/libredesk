<template>
  <div class="flex gap-4 h-[28rem]">
    <div :class="cn('flex flex-col', isPreview ? 'w-1/2' : 'w-full')">
      <div class="mb-2 text-sm font-medium">Content</div>
      <div
        ref="editorRef"
        class="border rounded dark:border-gray-600 flex-1 overflow-y-scroll"
      ></div>
    </div>
    <div class="w-1/2 flex flex-col" v-if="isPreview">
      <div class="mb-2 text-sm font-medium">Preview</div>
      <iframe
        ref="previewRef"
        class="w-full flex-1 border rounded dark:border-gray-600 bg-white"
        sandbox=""
      ></iframe>
    </div>
  </div>
</template>

<script setup>
import { onMounted, watch, useTemplateRef } from 'vue'
import { EditorView, basicSetup } from 'codemirror'
import { html } from '@codemirror/lang-html'
import { oneDark } from '@codemirror/theme-one-dark'
import { cn } from '@/lib/utils'

const props = defineProps({
  modelValue: { type: String, default: '' },
  isPreview: { type: Boolean, default: true },
  customScript: { type: String, default: '' },
  disabled: Boolean
})

const emit = defineEmits(['update:modelValue'])
const editorRef = useTemplateRef('editorRef')
const previewRef = useTemplateRef('previewRef')
let editorView = null

const updatePreview = (code, customScript) => {
  if (!previewRef.value) return

  let combinedCode = code
  if (customScript) {
    if (customScript.includes('{{ template "content" . }}')) {
      combinedCode = customScript.replace('{{ template "content" . }}', code)
    } else {
      combinedCode = customScript + code
    }
  }
  previewRef.value.srcdoc = combinedCode
}

onMounted(() => {
  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches

  editorView = new EditorView({
    doc: props.modelValue || '',
    extensions: [
      basicSetup,
      html(),
      ...(isDark ? [oneDark] : []),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          const code = update.state.doc.toString()
          emit('update:modelValue', code)
          updatePreview(code, props.customScript)
        }
      })
    ],
    parent: editorRef.value
  })

  updatePreview(props.modelValue, props.customScript)
})

watch(
  () => props.modelValue,
  (newVal) => {
    if (editorView && newVal !== editorView.state.doc.toString()) {
      editorView.dispatch({
        changes: { from: 0, to: editorView.state.doc.length, insert: newVal }
      })
    }
  }
)

watch(
  () => [props.modelValue, props.customScript],
  () => {
    updatePreview(props.modelValue, props.customScript)
  }
)
</script>
