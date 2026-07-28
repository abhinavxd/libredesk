<template>
  <div
    class="editor-wrapper relative flex flex-col h-full min-h-0"
    :class="{ 'pointer-events-none': disabled }"
  >
    <EditorContent :editor="editor" class="native-html flex-1 min-h-0 overflow-y-auto pb-20" />

    <div
      v-if="editor"
      class="editor-toolbar absolute bottom-4 left-1/2 z-10 w-max max-w-[calc(100%-2rem)] -translate-x-1/2 rounded-xl border bg-background p-1 shadow-lg"
    >
      <EditorToolbar
        :editor="editor"
        show-article-tools
        :enable-inline-images="enableInlineImages"
        @open-link="linkDialog?.open()"
        @open-youtube="youtubeDialog?.open()"
        @open-image="imageInput?.click()"
      />
    </div>

    <input
      ref="imageInput"
      type="file"
      accept="image/*"
      multiple
      class="hidden"
      @change="onImageInputChange"
    />

    <EditorLinkDialog ref="linkDialog" :editor="editor" allow-button />
    <EditorYoutubeDialog ref="youtubeDialog" :editor="editor" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { EditorContent } from '@tiptap/vue-3'
import EditorToolbar from './EditorToolbar.vue'
import EditorLinkDialog from './EditorLinkDialog.vue'
import EditorYoutubeDialog from './EditorYoutubeDialog.vue'
import { buildArticleExtensions } from './editorExtensions'
import { useTextEditor } from './useTextEditor'

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })

const props = defineProps({
  placeholder: String,
  insertContent: String,
  autoFocus: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
  enableInlineImages: { type: Boolean, default: false },
  linkedModel: { type: String, default: 'messages' }
})

const emit = defineEmits(['send', 'filesDropped'])

const linkDialog = ref(null)
const youtubeDialog = ref(null)
const imageInput = ref(null)

const { editor, insertImages, extractMentions, focus } = useTextEditor({
  props,
  extensions: buildArticleExtensions({ getPlaceholder: () => props.placeholder }),
  htmlContent,
  textContent,
  emit
})

const onImageInputChange = (event) => {
  const files = Array.from(event.target.files || [])
  if (files.length > 0) insertImages(files)
  event.target.value = ''
}

defineExpose({ focus, extractMentions })
</script>

<style lang="scss" src="./editorStyles.scss"></style>

<style lang="scss">
.tiptap {
  // Link styled as a call-to-action button
  a.hc-button {
    display: inline-block;
    padding: 0.5rem 1rem;
    background: hsl(var(--primary));
    color: hsl(var(--primary-foreground));
    border-radius: 0.375rem;
    font-weight: 500;
    text-decoration: none;

    &:hover {
      color: hsl(var(--primary-foreground));
    }
  }

  // Callout blocks
  .hc-callout {
    position: relative;
    margin: 1rem 0;
    padding: 0.75rem 1rem 0.75rem 2.75rem;
    border-radius: 6px;

    &::before {
      position: absolute;
      left: 0.85rem;
      top: 0.8rem;
      width: 1.25rem;
      height: 1.25rem;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #fff;
      font-weight: 700;
      font-size: 0.8rem;
      line-height: 1;
    }

    > :last-child { margin-bottom: 0; }
  }
  .hc-callout-info { background: #eff6ff; &::before { content: "i"; background: #3b82f6; } }
  .hc-callout-success { background: #f0fdf4; &::before { content: "✓"; background: #22c55e; } }
  .hc-callout-warning { background: #fffbeb; &::before { content: "!"; background: #f59e0b; } }
  .hc-callout-danger { background: #fef2f2; &::before { content: "!"; background: #ef4444; } }

  // Collapsible section (native <details>)
  details.hc-details {
    margin: 1rem 0;
    padding: 0.5rem 0.85rem;
    border: 1px solid hsl(var(--border));
    border-radius: 8px;

    > summary.hc-details-summary {
      font-weight: 600;
      cursor: pointer;
    }

    // Keep the body visible/editable regardless of the native open state.
    > .hc-details-content {
      display: block !important;
      margin-top: 0.5rem;

      > :last-child { margin-bottom: 0; }
    }
  }
}
</style>
