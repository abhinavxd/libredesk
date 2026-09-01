<template>
  <div class="h-full">
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

    <ApiKeyManager
      class="max-w-xl mt-8"
      :api-key="userStore.user.api_key"
      :last-used-at="userStore.user.api_key_last_used_at"
      :description="t('account.apiKey.description')"
      :empty-label="t('account.apiKey.noKey')"
      :generate-fn="api.generateMyAPIKey"
      :revoke-fn="api.revokeMyAPIKey"
      :fetch-fn="api.getMyAPIKey"
      @updated="onApiKeyUpdated"
    />

    <div class="max-w-xl mt-8 space-y-3">
      <div class="space-y-1">
        <span class="sub-title">{{ t('account.totp.title') }}</span>
        <p class="text-muted-foreground text-xs">{{ t('account.totp.description') }}</p>
      </div>
      <p v-if="userStore.user.totp_enabled" class="text-sm">{{ t('account.totp.enabled') }}</p>
      <div v-if="totpSecret" class="space-y-2">
        <p class="text-xs text-muted-foreground">{{ t('account.totp.secret') }}</p>
        <p class="font-mono text-sm break-all">{{ totpSecret }}</p>
        <Input v-model.trim="totpConfirmCode" inputmode="numeric" autocomplete="one-time-code" :placeholder="t('account.totp.code')" />
        <Button type="button" :disabled="totpBusy" @click="confirmTotp">{{ t('account.totp.confirm') }}</Button>
      </div>
      <Button
        v-else-if="!userStore.user.totp_enabled"
        type="button"
        variant="outline"
        :disabled="totpBusy"
        @click="enrollTotp"
      >
        {{ t('account.totp.enable') }}
      </Button>
      <Button
        v-if="userStore.user.totp_enabled"
        type="button"
        variant="outline"
        :disabled="totpBusy"
        @click="disableTotp"
      >
        {{ t('account.totp.disable') }}
      </Button>
    </div>
  </div>
</template>

<script setup>
import { useUserStore } from '../../../stores/user'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import { ref } from 'vue'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'
import ApiKeyManager from '@/features/account/ApiKeyManager.vue'

const emitter = useEmitter()
const { t } = useI18n()
const isSaving = ref(false)
const userStore = useUserStore()
const pendingFile = ref(null)
const totpBusy = ref(false)
const totpSecret = ref('')
const totpConfirmCode = ref('')

const toastErr = (error) => {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(error).message
  })
}

const enrollTotp = async () => {
  totpBusy.value = true
  try {
    const { data } = await api.enrollTOTP()
    totpSecret.value = data.data?.secret || ''
  } catch (error) {
    toastErr(error)
  } finally {
    totpBusy.value = false
  }
}

const confirmTotp = async () => {
  if (!totpConfirmCode.value) return
  totpBusy.value = true
  try {
    await api.confirmTOTP({ code: totpConfirmCode.value })
    totpSecret.value = ''
    totpConfirmCode.value = ''
    await userStore.getCurrentUser()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('account.totp.enabled') })
  } catch (error) {
    toastErr(error)
  } finally {
    totpBusy.value = false
  }
}

const disableTotp = async () => {
  totpBusy.value = true
  try {
    await api.disableTOTP()
    await userStore.getCurrentUser()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('account.totp.disabled') })
  } catch (error) {
    toastErr(error)
  } finally {
    totpBusy.value = false
  }
}

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

const onApiKeyUpdated = ({ api_key, api_key_last_used_at }) => {
  userStore.user.api_key = api_key
  userStore.user.api_key_last_used_at = api_key_last_used_at
}
</script>
