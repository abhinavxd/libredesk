<template>
  <AuthLayout>
    <Card class="bg-card box">
      <CardContent class="p-6 space-y-6">
        <div class="space-y-2 text-center">
          <CardTitle class="text-3xl font-bold text-foreground">{{
            t('auth.resetPassword')
          }}</CardTitle>
          <p class="text-muted-foreground">{{ t('auth.enterEmailForReset') }}</p>
        </div>

        <form @submit.prevent="requestResetAction" class="space-y-4">
          <div class="space-y-2">
            <Label for="email" class="text-sm font-medium text-foreground">{{
              t('globals.terms.email')
            }}</Label>
            <Input
              id="email"
              type="email"
              :placeholder="t('auth.enterEmail')"
              v-model.trim="resetForm.email"
              :class="{ 'border-destructive': emailHasError }"
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
              {{ t('auth.sending') }}
            </span>
            <span v-else>{{ t('auth.sendResetLink') }}</span>
          </Button>
        </form>

        <Error
          v-if="errorMessage"
          :errorMessage="errorMessage"
          :border="true"
          class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded text-sm"
        />

        <div class="text-center">
          <router-link
            to="/"
            class="text-sm text-primary hover:text-primary/80 transition-all duration-200 ease-in-out"
          >
            {{ t('auth.backToLogin') }}
          </router-link>
        </div>
      </CardContent>
    </Card>
  </AuthLayout>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '../../utils/http'
import api from '../../api'
import { validateEmail } from '../../utils/strings'
import { useTemporaryClass } from '../../composables/useTemporaryClass'
import { Button } from '@shared-ui/components/ui/button'
import { Error } from '@shared-ui/components/ui/error'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { EMITTER_EVENTS } from '../../constants/emitterEvents.js'
import { useEmitter } from '../../composables/useEmitter'
import { Label } from '@shared-ui/components/ui/label'
import { useI18n } from 'vue-i18n'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'

const errorMessage = ref('')
const { t } = useI18n()
const isLoading = ref(false)
const emitter = useEmitter()
const router = useRouter()
const resetForm = ref({
  email: ''
})

const validateForm = () => {
  if (!validateEmail(resetForm.value.email)) {
    errorMessage.value = t('globals.messages.invalidEmailAddress')
    useTemporaryClass('reset-password-container', 'animate-shake')
    return false
  }
  return true
}

const requestResetAction = async () => {
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true

  try {
    await api.resetPassword({
      email: resetForm.value.email
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('auth.checkEmailForReset')
    })
    router.push({ name: 'login' })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
    errorMessage.value = handleHTTPError(err).message
    useTemporaryClass('reset-password-container', 'animate-shake')
  } finally {
    isLoading.value = false
  }
}

const emailHasError = computed(() => {
  return !validateEmail(resetForm.value.email) && resetForm.value.email !== ''
})
</script>
