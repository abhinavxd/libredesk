<template>
  <div class="h-full">
    <div class="flex flex-col space-y-5">
      <div class="space-y-1">
        <span class="sub-title">{{ $t('account.publicAvatar') }}</span>
        <p class="text-muted-foreground text-xs">{{ $t('account.changeAvatar') }}</p>
      </div>
      <div class="flex space-x-5">
        <Avatar class="size-28">
          <AvatarImage :src="userStore.avatar" alt="Cropped Image" />
          <AvatarFallback>{{ userStore.getInitials }}</AvatarFallback>
        </Avatar>

        <div class="flex flex-col space-y-5 justify-center">
          <input
            ref="uploadInput"
            type="file"
            hidden
            accept="image/jpg, image/jpeg, image/png"
            @change="selectFile"
          />
          <Button class="w-28" @click="selectAvatar"> {{ $t('account.chooseAFile') }} </Button>
          <Button class="w-28" @click="removeAvatar" variant="destructive">
            {{ $t('account.removeAvatar') }}
          </Button>
        </div>
      </div>

      <Button class="w-28" @click="saveUser" size="sm" :isLoading="isSaving">
        {{ $t('globals.messages.saveChanges') }}
      </Button>

      <!-- Cropped dialog -->
      <Dialog :open="showCropper">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle class="text-xl">{{ $t('account.cropAvatar') }}</DialogTitle>
            <DialogDescription />
          </DialogHeader>

          <VuePictureCropper
            :boxStyle="{
              width: '100%',
              height: '400px',
              backgroundColor: '#f8f8f8',
              margin: 'auto'
            }"
            :img="newUserAvatar"
            :options="{ viewMode: 1, dragMode: 'crop', aspectRatio: 1 }"
          />
          <DialogFooter class="sm:justify-end">
            <Button variant="secondary" @click="closeDialog">
              {{ $t('globals.messages.close') }}
            </Button>
            <Button @click="getResult">{{ $t('globals.messages.save') }}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  </div>
</template>

<script setup>
import { useUserStore } from '../../../stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { ref } from 'vue'
import VuePictureCropper, { cropper } from 'vue-picture-cropper'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const emitter = useEmitter()
const { t } = useI18n()
const isSaving = ref(false)
const userStore = useUserStore()
const uploadInput = ref(null)
const newUserAvatar = ref('')
const showCropper = ref(false)
let croppedBlob = null
let avatarFile = null

const selectAvatar = () => {
  uploadInput.value.click()
}

const selectFile = (event) => {
  newUserAvatar.value = ''
  const { files } = event.target
  if (!files || !files.length) return
  avatarFile = files[0]
  const reader = new FileReader()
  reader.readAsDataURL(avatarFile)
  reader.onload = () => {
    newUserAvatar.value = String(reader.result)
    showCropper.value = true
    uploadInput.value.value = ''
  }
}

const closeDialog = () => {
  showCropper.value = false
}

const getResult = async () => {
  if (!cropper) return
  croppedBlob = await cropper.getBlob()
  if (!croppedBlob) return
  userStore.setAvatar(URL.createObjectURL(croppedBlob))
  showCropper.value = false
}

const saveUser = async () => {
  const formData = new FormData()
  formData.append('files', croppedBlob, 'avatar.png')
  try {
    isSaving.value = true
    await api.updateCurrentUser(formData)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSaving.value = false
  }
}

const removeAvatar = async () => {
  croppedBlob = null
  try {
    await api.deleteUserAvatar()
    userStore.clearAvatar()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('account.avatarRemoved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
