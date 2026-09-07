<template>
  <div
    class="relative group w-28 h-28"
    :class="disabled ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'"
    @click="triggerFileInput"
  >
    <Avatar class="size-28">
      <AvatarImage :src="src || ''" />
      <AvatarFallback>{{ initials }}</AvatarFallback>
    </Avatar>

    <!-- Hover Overlay -->
    <div
      class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer rounded-full"
    >
      <span class="text-white font-semibold">{{ label }}</span>
    </div>

    <!-- Delete Icon -->
    <button
      v-if="src"
      type="button"
      :disabled="disabled"
      class="absolute top-1 right-1 rounded-full p-0.5 bg-destructive text-destructive-foreground shadow-md z-10 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer disabled:cursor-not-allowed"
      aria-label="Remove avatar"
      @click.stop="emit('remove')"
    >
      <X size="14" />
    </button>

    <!-- File Input -->
    <input
      ref="fileInput"
      type="file"
      class="hidden"
      accept="image/png,image/jpeg,image/jpg"
      @change="handleChange"
    />

    <!-- Crop dialog -->
    <Dialog :open="showCropper" @update:open="!$event && closeCropper()">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle class="text-xl">{{ t('account.cropAvatar') }}</DialogTitle>
          <DialogDescription />
        </DialogHeader>

        <VuePictureCropper
          :boxStyle="{
            width: '100%',
            height: '400px',
            backgroundColor: 'hsl(var(--muted))',
            margin: 'auto'
          }"
          :img="cropSource"
          :options="{ viewMode: 1, dragMode: 'crop', aspectRatio: 1 }"
        />

        <DialogFooter class="sm:justify-end">
          <Button type="button" variant="secondary" @click="closeCropper">
            {{ t('globals.messages.cancel') }}
          </Button>
          <Button type="button" @click="applyCrop">{{ t('globals.messages.save') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarImage, AvatarFallback } from '.'
import { X } from 'lucide-vue-next'
import VuePictureCropper, { cropper } from 'vue-picture-cropper'
import { Button } from '../button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '../dialog'

const props = defineProps({
  src: String,
  initials: String,
  label: {
    type: String,
    default: 'Upload'
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['upload', 'remove'])
const { t } = useI18n()
const fileInput = ref(null)
const showCropper = ref(false)
const cropSource = ref('')

function triggerFileInput() {
  if (props.disabled) return
  fileInput.value?.click()
}

function handleChange(e) {
  const file = e.target.files[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    cropSource.value = String(reader.result)
    showCropper.value = true
  }
  reader.readAsDataURL(file)
}

function closeCropper() {
  showCropper.value = false
  cropSource.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

async function applyCrop() {
  if (!cropper) return
  const blob = await cropper.getBlob()
  if (!blob) return
  emit('upload', new File([blob], 'avatar.png', { type: blob.type || 'image/png' }))
  closeCropper()
}
</script>
