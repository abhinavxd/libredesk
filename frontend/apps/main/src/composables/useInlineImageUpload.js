import { useFileUpload } from './useFileUpload'

// Must match ResizableImage's renderHTML output.
const INLINE_IMAGE_MARKER = 'class="inline-image"'
const UPLOAD_PLACEHOLDER_MARKER = 'data-upload-placeholder'

export const hasInlineImage = (html) => (html || '').includes(INLINE_IMAGE_MARKER)
export const hasPendingInlineUpload = (html) => (html || '').includes(UPLOAD_PLACEHOLDER_MARKER)

/**
 * `getEditor` is a function (not a ref) because the editor doesn't exist
 * when the composable is called - `useEditor()`'s `editorProps` needs
 * `handlePaste` and `handleDrop` available up front.
 *
 * `isInlineEnabled` is a getter so per-conversation channel changes are
 * picked up live. When false, images route through `onOtherFiles`: used
 * for livechat, where signed URLs inside stored HTML age out but
 * attachment URLs get re-signed on every fetch.
 *
 * @param {Object} options
 * @param {() => Object} options.getEditor
 * @param {() => boolean} [options.isInlineEnabled]
 * @param {(files: File[]) => void} [options.onOtherFiles]
 * @param {string} [options.linkedModel='messages']
 * @param {number} [options.maxInlineImages=50]
 */
export function useInlineImageUpload ({
    getEditor,
    isInlineEnabled = () => true,
    onOtherFiles,
    linkedModel = 'messages',
    maxInlineImages = 50
} = {}) {
    const { upload } = useFileUpload({ linkedModel })

    const countInlineImages = () => {
        const editor = getEditor()
        if (!editor) return 0
        let count = 0
        editor.state.doc.descendants((node) => {
            if (node.type.name === 'image') count++
        })
        return count
    }

    const remainingSlots = () => Math.max(0, maxInlineImages - countInlineImages())

    const newUploadId = () =>
        typeof crypto?.randomUUID === 'function'
            ? crypto.randomUUID()
            : `ul-${Date.now()}-${Math.random().toString(36).slice(2)}`

    // Right-to-left so positions stay valid across deletes; harmless for
    // setNodeMarkup which doesn't shift positions.
    const mutateImagesByUploadId = (uploadId, mutate) => {
        const editor = getEditor()
        if (!editor) return
        const matches = []
        editor.state.doc.descendants((node, pos) => {
            if (node.type.name === 'image' && node.attrs.uploadId === uploadId) {
                matches.push({ pos, node })
            }
        })
        if (matches.length === 0) return
        const tr = editor.state.tr
        for (const { pos, node } of matches.sort((a, b) => b.pos - a.pos)) {
            mutate(tr, pos, node)
        }
        editor.view.dispatch(tr)
    }

    const replacePlaceholder = (uploadId, src) =>
        mutateImagesByUploadId(uploadId, (tr, pos, node) => {
            tr.setNodeMarkup(pos, undefined, {
                ...node.attrs,
                src,
                uploading: false,
                uploadId: null,
                uploadName: null
            })
        })

    const removePlaceholder = (uploadId) =>
        mutateImagesByUploadId(uploadId, (tr, pos, node) => {
            tr.delete(pos, pos + node.nodeSize)
        })

    // Single insertContent with N nodes - not a loop of setImage calls,
    // which would each NodeSelection-select the inserted image and the
    // next call would replace it instead of appending.
    const uploadAndInsertInOrder = async (files) => {
        const editor = getEditor()
        if (!editor || files.length === 0) return
        const pending = files.map((file) => ({ file, uploadId: newUploadId() }))
        const nodes = pending.map(({ file, uploadId }) => ({
            type: 'image',
            attrs: { src: '', uploading: true, uploadId, uploadName: file.name }
        }))
        editor.chain().focus().insertContent(nodes).run()

        await Promise.all(
            pending.map(async ({ file, uploadId }) => {
                const media = await upload(file, { inline: true })
                if (media?.url) replacePlaceholder(uploadId, media.url)
                else removePlaceholder(uploadId)
            })
        )
    }

    const acceptImages = (images) => {
        const allowed = images.slice(0, remainingSlots())
        if (allowed.length > 0) uploadAndInsertInOrder(allowed)
    }

    const dispatchFiles = (event, fileList) => {
        const imageFiles = []
        const otherFiles = []
        for (const file of fileList) {
            // Force SVG into other.
            if (file.type.startsWith('image/') && file.type !== 'image/svg+xml') {
                imageFiles.push(file)
            } else {
                otherFiles.push(file)
            }
        }
        if (imageFiles.length === 0 && otherFiles.length === 0) return false
        event.preventDefault()
        if (isInlineEnabled()) {
            acceptImages(imageFiles)
            if (otherFiles.length > 0 && onOtherFiles) onOtherFiles(otherFiles)
        } else if (onOtherFiles && (imageFiles.length > 0 || otherFiles.length > 0)) {
            onOtherFiles([...imageFiles, ...otherFiles])
        }
        return true
    }

    const handlePaste = (view, event) => {
        const data = event.clipboardData
        if (!data) return false

        // OS-level file paste (file manager): `files` reliably exposes all
        // entries, unlike `items` which some browsers only populate with
        // the first one.
        if (data.files && data.files.length > 0) {
            return dispatchFiles(event, data.files)
        }

        // Rich-content pastes (Google Docs, Word, web pages) carry text/html
        // alongside any image data. Let ProseMirror handle so we don't strip
        // the text.
        const types = Array.from(data.types || [])
        if (types.includes('text/html') || types.includes('text/plain')) return false

        const filesFromItems = []
        for (const item of data.items || []) {
            if (item.kind !== 'file') continue
            const file = item.getAsFile()
            if (file) filesFromItems.push(file)
        }
        if (filesFromItems.length === 0) return false
        return dispatchFiles(event, filesFromItems)
    }

    const handleDrop = (view, event) => {
        const files = event.dataTransfer?.files
        if (!files || files.length === 0) return false
        return dispatchFiles(event, files)
    }

    return { handlePaste, handleDrop }
}
