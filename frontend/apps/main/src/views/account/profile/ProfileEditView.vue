<template>
  <div class="h-full">
    <div class="flex flex-col space-y-8">
      <div class="flex flex-col space-y-5">
        <div class="space-y-1">
          <span class="sub-title">{{ $t('account.publicAvatar') }}</span>
          <p class="text-muted-foreground text-xs">{{ $t('account.changeAvatar') }}</p>
        </div>
        <AvatarUpload
          :src="userStore.avatar"
          :initials="userStore.getInitials"
          :label="$t('globals.messages.upload')"
          :disabled="isSaving"
          @upload="onCropped"
          @remove="removeAvatar"
        />

        <Button
          class="self-start"
          @click="saveUser"
          :isLoading="isSaving"
          :disabled="!pendingFile"
        >
          {{ $t('globals.messages.saveChanges') }}
        </Button>
      </div>

      <div class="flex flex-col space-y-3 max-w-sm">
        <div class="space-y-1">
          <span class="sub-title">{{ $t('account.defaultInbox') }}</span>
          <p class="text-muted-foreground text-xs">{{ $t('account.defaultInboxHelp') }}</p>
        </div>
        <Select :model-value="userStore.defaultInbox" :disabled="isSavingInbox" @update:model-value="saveDefaultInbox">
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="assigned">{{ $t('globals.terms.myInbox') }}</SelectItem>
              <SelectItem value="mentioned">{{ $t('globals.terms.mention', 2) }}</SelectItem>
              <SelectItem value="unassigned">{{ $t('globals.terms.unassigned') }}</SelectItem>
              <SelectItem value="all">{{ $t('globals.messages.all') }}</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useUserStore } from '../../../stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { ref } from 'vue'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const emitter = useEmitter()
const { t } = useI18n()
const isSaving = ref(false)
const isSavingInbox = ref(false)
const userStore = useUserStore()
const pendingFile = ref(null)

const onCropped = (file) => {
  if (isSaving.value) return
  pendingFile.value = file
  userStore.setAvatar(URL.createObjectURL(file))
}

const saveUser = async () => {
  if (!pendingFile.value) return
  const formData = new FormData()
  formData.append('files', pendingFile.value, 'avatar.png')
  try {
    isSaving.value = true
    await api.updateCurrentUser(formData)
    pendingFile.value = null
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

const saveDefaultInbox = async (inbox) => {
  if (!inbox || inbox === userStore.defaultInbox || isSavingInbox.value) return
  try {
    isSavingInbox.value = true
    await userStore.updateDefaultInbox(inbox)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSavingInbox.value = false
  }
}

const removeAvatar = async () => {
  if (isSaving.value) return
  try {
    await api.deleteUserAvatar()
    pendingFile.value = null
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
