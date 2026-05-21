import { ref, readonly } from 'vue'
import { useEmitter } from './useEmitter'
import { EMITTER_EVENTS } from '../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../api'

/**
 * Composable for handling file uploads
 * @param {Object} options - Configuration options
 * @param {Function} options.onFileUploadSuccess - Callback when file upload succeeds (uploadedFile)
 * @param {Function} options.onUploadError - Optional callback when file upload fails (file, error)
 * @param {string} options.linkedModel - The linked model for the upload
 * @param {Array} options.mediaFiles - Optional external array to manage files (if not provided, internal array is used)
 */
export function useFileUpload (options = {}) {
    const {
        onFileUploadSuccess,
        onUploadError,
        linkedModel,
        mediaFiles: externalMediaFiles
    } = options

    const emitter = useEmitter()
    const uploadingFiles = ref([])
    const isUploading = ref(false)
    const internalMediaFiles = ref([])

    const mediaFiles = externalMediaFiles || internalMediaFiles

    /**
     * Returns the media record or null on failure (toast fires).
     *
     * `inline: true` flags `disposition=inline` server-side; without it an
     * editor-embedded image would also surface as a downloadable attachment
     * under the message (MessageBubble filters by disposition).
     *
     * @param {File} file
     * @param {{ inline?: boolean }} opts
     */
    const upload = async (file, { inline = false } = {}) => {
        try {
            const resp = await api.uploadMedia({
                files: file,
                inline,
                linked_model: linkedModel
            })
            return resp.data.data
        } catch (error) {
            if (onUploadError) {
                onUploadError(file, error)
            } else {
                emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
                    variant: 'destructive',
                    description: handleHTTPError(error).message
                })
            }
            return null
        }
    }

    /**
     * Handles the file upload process when files are selected.
     * Uploads each file to the server and adds them to the mediaFiles array.
     * @param {Event} event - The file input change event containing selected files
     */
    const handleFileUpload = (event) => {
        const files = Array.from(event.target.files)
        uploadingFiles.value = files
        isUploading.value = true

        for (const file of files) {
            upload(file).then((uploadedFile) => {
                if (uploadedFile) {
                    if (Array.isArray(mediaFiles.value)) {
                        mediaFiles.value.push(uploadedFile)
                    } else {
                        mediaFiles.push(uploadedFile)
                    }
                    if (onFileUploadSuccess) {
                        onFileUploadSuccess(uploadedFile)
                    }
                }
                uploadingFiles.value = uploadingFiles.value.filter((f) => f.name !== file.name)
                if (uploadingFiles.value.length === 0) {
                    isUploading.value = false
                }
            })
        }
    }

    /**
     * Handles the file delete event.
     * Removes the file from the mediaFiles array.
     * @param {String} uuid - The UUID of the file to delete
     */
    const handleFileDelete = (uuid) => {
        if (Array.isArray(mediaFiles.value)) {
            mediaFiles.value = [
                ...mediaFiles.value.filter((item) => item.uuid !== uuid)
            ]
        } else {
            const index = mediaFiles.findIndex((item) => item.uuid === uuid)
            if (index > -1) {
                mediaFiles.splice(index, 1)
            }
        }
    }

    /**
     * Upload files programmatically (without event)
     * @param {File[]} files - Array of files to upload
     */
    const uploadFiles = (files) => {
        const mockEvent = { target: { files } }
        handleFileUpload(mockEvent)
    }

    /**
     * Clear all media files
     */
    const clearMediaFiles = () => {
        if (Array.isArray(mediaFiles.value)) {
            mediaFiles.value = []
        } else {
            mediaFiles.length = 0
        }
    }

    /**
     * Replace all media files with new files
     * @param {Array} files - Array of file objects to set
     */
    const setMediaFiles = (files) => {
        if (Array.isArray(mediaFiles.value)) {
            mediaFiles.value = files
        } else {
            mediaFiles.length = 0
            mediaFiles.push(...files)
        }
    }

    return {
        // State
        uploadingFiles: readonly(uploadingFiles),
        isUploading: readonly(isUploading),
        mediaFiles: externalMediaFiles ? readonly(mediaFiles) : readonly(internalMediaFiles),

        // Methods
        upload,
        handleFileUpload,
        handleFileDelete,
        uploadFiles,
        clearMediaFiles,
        setMediaFiles
    }
}
