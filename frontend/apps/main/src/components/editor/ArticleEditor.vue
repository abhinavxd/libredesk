<template>
  <div
    class="editor-wrapper relative flex flex-col h-full min-h-0"
    :class="{ 'pointer-events-none': disabled }"
  >
    <Teleport :to="toolbarTarget" :disabled="!toolbarTarget">
      <div
        v-if="editor"
        :inert="disabled"
        class="editor-toolbar sticky top-0 z-10 mx-auto w-fit max-w-full rounded-xl border bg-background p-1 shadow-sm"
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
    </Teleport>

    <EditorContent :editor="editor" class="hc-prose flex-1 min-h-0 overflow-y-auto" />

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
import { ref, watch } from 'vue'
import { EditorContent } from '@tiptap/vue-3'
import EditorToolbar from './EditorToolbar.vue'
import EditorLinkDialog from './EditorLinkDialog.vue'
import EditorYoutubeDialog from './EditorYoutubeDialog.vue'
import { buildArticleExtensions } from './editorExtensions'
import { useTextEditor } from './useTextEditor'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const textContent = defineModel('textContent', { default: '' })
const htmlContent = defineModel('htmlContent', { default: '' })

const props = defineProps({
  placeholder: String,
  insertContent: String,
  autoFocus: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
  enableInlineImages: { type: Boolean, default: false },
  linkedModel: { type: String, default: 'messages' },
  toolbarTarget: { type: null, default: null }
})

const emit = defineEmits(['send', 'filesDropped'])

const linkDialog = ref(null)
const youtubeDialog = ref(null)
const imageInput = ref(null)

const { editor, insertImages, focus } = useTextEditor({
  extensions: buildArticleExtensions({
    getPlaceholder: () => props.placeholder,
    embedTitle: t('editor.tooltip.youtube'),
    defaultSummary: t('globals.terms.summary')
  }),
  htmlContent,
  textContent,
  autoFocus: props.autoFocus,
  editable: !props.disabled,
  insertContent: () => props.insertContent,
  isInlineEnabled: () => props.enableInlineImages,
  linkedModel: props.linkedModel,
  onSend: () => emit('send'),
  onOtherFiles: (files) => emit('filesDropped', files)
})

watch(
  () => props.disabled,
  (disabled) => editor.value?.setEditable(!disabled, false)
)

const onImageInputChange = (event) => {
  const files = Array.from(event.target.files || [])
  if (files.length > 0) insertImages(files)
  event.target.value = ''
}

defineExpose({ focus })
</script>

<style lang="scss" src="./editorStyles.scss"></style>

<style src="@public-static/article-content.css"></style>

<style lang="scss">
.tiptap {
  --hc-accent: hsl(var(--primary));
  --hc-accent-ink: color-mix(in srgb, var(--hc-accent), #0d1117 46%);
  --hc-border: hsl(var(--border));
  --hc-accent-tint: hsl(var(--primary) / 0.08);
  --hc-muted: hsl(var(--muted-foreground));

  details.hc-details > .hc-details-content {
    margin-top: 0.5rem;
  }

  // An interactive iframe steals every click in the editor. It still plays once published.
  [data-youtube-video] iframe {
    pointer-events: none;
  }

  [data-youtube-video].ProseMirror-selectednode iframe {
    outline: 2px solid hsl(var(--link));
    outline-offset: 2px;
  }
}
</style>
