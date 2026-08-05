<template>
  <AuthLayout>
    <Card class="bg-card box" id="set-password-container">
      <CardContent class="p-6 space-y-5">
        <div class="space-y-1 text-center">
          <CardTitle class="text-2xl font-bold text-foreground">{{
            t('auth.setNewPassword')
          }}</CardTitle>
          <p class="text-sm text-muted-foreground">{{ t('auth.enterNewPasswordTwice') }}</p>
        </div>

        <form @submit.prevent="setPasswordAction" class="space-y-3">
          <div class="space-y-2">
            <Label for="password" class="text-muted-foreground">
              {{
                t('auth.newPassword')
              }}
            </Label>
            <div class="relative">
              <Input
                id="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="new-password"
                v-model="passwordForm.password"
                :class="{ 'border-destructive': passwordHasError }"
                class="pr-10"
              />
              <button
                type="button"
                :aria-label="showPassword ? t('auth.hidePassword') : t('auth.showPassword')"
                class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground"
                @click="showPassword = !showPassword"
              >
                <Eye v-if="!showPassword" class="w-5 h-5" />
                <EyeOff v-else class="w-5 h-5" />
              </button>
            </div>
          </div>

          <div class="space-y-2">
            <Label for="confirmPassword" class="text-muted-foreground">
              {{ t('auth.confirmPassword') }}
            </Label>
            <div class="relative">
              <Input
                id="confirmPassword"
                :type="showConfirmPassword ? 'text' : 'password'"
                autocomplete="new-password"
                v-model="passwordForm.confirmPassword"
                :class="{ 'border-destructive': confirmPasswordHasError }"
                class="pr-10"
              />
              <button
                type="button"
                :aria-label="showConfirmPassword ? t('auth.hidePassword') : t('auth.showPassword')"
                class="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground"
                @click="showConfirmPassword = !showConfirmPassword"
              >
                <Eye v-if="!showConfirmPassword" class="w-5 h-5" />
                <EyeOff v-else class="w-5 h-5" />
              </button>
            </div>
          </div>

          <Button
            class="w-full"
            :disabled="isLoading"
            type="submit"
          >
            <span v-if="isLoading" class="flex items-center justify-center">
              <div
                class="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin mr-3"
              ></div>
              {{ t('auth.settingPassword') }}
            </span>
            <span v-else>{{ t('auth.setNewPassword') }}</span>
          </Button>
        </form>

        <Error
          v-if="errorMessage"
          :errorMessage="errorMessage"
          :border="true"
          class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded-md text-sm"
        />
      </CardContent>
    </Card>
  </AuthLayout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../../api'
import { useEmitter } from '../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { applyTemporaryClass } from '@/utils/temporary-class'
import { Button } from '@shared-ui/components/ui/button'
import { Error } from '@shared-ui/components/ui/error'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Eye, EyeOff } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'

const PASSWORD_MIN_LENGTH = 8
const PASSWORD_MAX_LENGTH = 72

const { t } = useI18n()
const errorMessage = ref('')
const showPassword = ref(false)
const showConfirmPassword = ref(false)
const isLoading = ref(false)
const submitted = ref(false)
const router = useRouter()
const route = useRoute()
const emitter = useEmitter()
const passwordForm = ref({
  password: '',
  confirmPassword: '',
  token: ''
})

onMounted(() => {
  passwordForm.value.token = route.query.token
  if (!passwordForm.value.token) {
    router.push({ name: 'login' })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('auth.invalidResetLink')
    })
  }
})

const validateForm = () => {
  if (!passwordForm.value.password) {
    errorMessage.value = t('auth.passwordRequired')
    applyTemporaryClass('set-password-container', 'animate-shake')
    return false
  }
  if (!isPasswordLengthValid(passwordForm.value.password)) {
    errorMessage.value = t('validation.minmax', {
      min: PASSWORD_MIN_LENGTH,
      max: PASSWORD_MAX_LENGTH
    })
    applyTemporaryClass('set-password-container', 'animate-shake')
    return false
  }
  if (passwordForm.value.password !== passwordForm.value.confirmPassword) {
    errorMessage.value = t('auth.passwordsDoNotMatch')
    applyTemporaryClass('set-password-container', 'animate-shake')
    return false
  }
  return true
}

const setPasswordAction = async () => {
  submitted.value = true
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true

  try {
    await api.setPassword({
      token: passwordForm.value.token,
      password: passwordForm.value.password
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('auth.passwordSetSuccess')
    })
    router.push({ name: 'login' })
  } catch (err) {
    errorMessage.value = handleHTTPError(err).message
    applyTemporaryClass('set-password-container', 'animate-shake')
  } finally {
    isLoading.value = false
  }
}

const passwordHasError = computed(() => {
  return submitted.value && !isPasswordLengthValid(passwordForm.value.password)
})

const confirmPasswordHasError = computed(() => {
  return submitted.value && passwordForm.value.password !== passwordForm.value.confirmPassword
})

function isPasswordLengthValid(password) {
  return password.length >= PASSWORD_MIN_LENGTH && password.length <= PASSWORD_MAX_LENGTH
}
</script>
