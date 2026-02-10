<template>
  <AuthLayout>
    <Card class="bg-card box">
      <CardContent class="p-6 space-y-6">
        <div class="space-y-2 text-center">
          <CardTitle class="text-3xl font-bold text-foreground">{{
            t('auth.setNewPassword')
          }}</CardTitle>
          <p class="text-muted-foreground">{{ t('auth.enterNewPasswordTwice') }}</p>
        </div>

        <form @submit.prevent="setPasswordAction" class="space-y-4">
          <div class="space-y-2">
            <Label for="password" class="text-sm font-medium text-foreground">
              {{
                t('globals.messages.new', {
                  name: t('globals.terms.password').toLowerCase()
                })
              }}
            </Label>
            <Input
              id="password"
              type="password"
              autocomplete="new-password"
              :placeholder="t('auth.enterNewPassword')"
              v-model="passwordForm.password"
              :class="{ 'border-destructive': passwordHasError }"
              class="w-full bg-card border-border text-foreground placeholder:text-muted-foreground rounded py-2 px-3 focus:ring-2 focus:ring-ring focus:border-ring transition-all duration-200 ease-in-out"
            />
          </div>

          <div class="space-y-2">
            <Label for="confirmPassword" class="text-sm font-medium text-foreground">
              {{ t('auth.confirmPassword') }}
            </Label>
            <Input
              id="confirmPassword"
              type="password"
              autocomplete="new-password"
              :placeholder="t('auth.confirmNewPassword')"
              v-model="passwordForm.confirmPassword"
              :class="{ 'border-destructive': confirmPasswordHasError }"
              class="w-full bg-card border-border text-foreground placeholder:text-muted-foreground rounded py-2 px-3 focus:ring-2 focus:ring-ring focus:border-ring transition-all duration-200 ease-in-out"
            />
          </div>

          <Button
            class="w-full bg-primary hover:bg-primary/90 text-primary-foreground rounded py-2 transition-all duration-200 ease-in-out transform hover:scale-105"
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
          class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded text-sm"
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
import { useTemporaryClass } from '../../composables/useTemporaryClass'
import { Button } from '@shared-ui/components/ui/button'
import { Error } from '@shared-ui/components/ui/error'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'

const { t } = useI18n()
const errorMessage = ref('')
const isLoading = ref(false)
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
      description: 'Invalid reset link. Please request a new password reset link.'
    })
  }
})

const validateForm = () => {
  if (!passwordForm.value.password) {
    errorMessage.value = t('auth.passwordRequired')
    useTemporaryClass('set-password-container', 'animate-shake')
    return false
  }
  if (passwordForm.value.password !== passwordForm.value.confirmPassword) {
    errorMessage.value = t('auth.passwordsDoNotMatch')
    useTemporaryClass('set-password-container', 'animate-shake')
    return false
  }
  return true
}

const setPasswordAction = async () => {
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
    useTemporaryClass('set-password-container', 'animate-shake')
  } finally {
    isLoading.value = false
  }
}

const passwordHasError = computed(() => {
  return passwordForm.value.password !== '' && passwordForm.value.password.length < 8
})

const confirmPasswordHasError = computed(() => {
  return (
    passwordForm.value.confirmPassword !== '' &&
    passwordForm.value.password !== passwordForm.value.confirmPassword
  )
})
</script>
