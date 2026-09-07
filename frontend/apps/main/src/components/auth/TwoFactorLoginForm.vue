<template>
  <form @submit.prevent="verify" class="space-y-4">
    <p id="two-factor-hint" class="text-sm text-muted-foreground">
      {{ t(recovery ? 'twoFactor.recoveryLoginHint' : 'twoFactor.loginHint') }}
    </p>
    <div class="space-y-2">
      <Label for="two-factor-code">{{
        t(recovery ? 'twoFactor.recoveryCode' : 'globals.terms.authenticationCode')
      }}</Label>
      <Input
        id="two-factor-code"
        v-model="code"
        type="text"
        :inputmode="recovery ? 'text' : 'numeric'"
        :autocomplete="recovery ? 'off' : 'one-time-code'"
        :maxlength="recovery ? 35 : 6"
        autocapitalize="none"
        :spellcheck="false"
        aria-describedby="two-factor-hint two-factor-error"
        :aria-invalid="!!error"
      />
    </div>
    <p id="two-factor-error" role="alert" class="text-sm text-destructive">{{ error }}</p>
    <Button type="submit" class="w-full" :isLoading="busy" :disabled="busy">
      {{ t('globals.messages.verifyCode') }}
    </Button>
    <div class="flex flex-wrap justify-between gap-2">
      <Button type="button" variant="link" class="px-0" :disabled="busy" @click="toggleRecovery">
        {{ t(recovery ? 'twoFactor.useAuthenticator' : 'twoFactor.useRecoveryCode') }}
      </Button>
      <Button type="button" variant="link" class="px-0" :disabled="busy" @click="emit('cancel')">
        {{ t('auth.backToLogin') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'

const emit = defineEmits(['success', 'cancel'])
const { t } = useI18n()
const code = ref('')
const recovery = ref(false)
const error = ref('')
const busy = ref(false)
const focusCode = () => document.getElementById('two-factor-code')?.focus()
onMounted(focusCode)

const toggleRecovery = async () => {
  recovery.value = !recovery.value
  code.value = ''
  error.value = ''
  await nextTick()
  focusCode()
}

const verify = async () => {
  if (busy.value) return
  error.value = ''
  if (!code.value.trim()) {
    error.value = t('twoFactor.enterCode')
    focusCode()
    return
  }
  busy.value = true
  try {
    const response = await api.verifyTwoFactorLogin({ code: code.value.trim() })
    code.value = ''
    emit('success', response)
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    busy.value = false
  }
}
</script>
