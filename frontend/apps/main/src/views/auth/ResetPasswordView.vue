<template>
  <AuthLayout>
    <Card class="bg-card box" id="reset-password-container">
      <CardContent class="p-6 space-y-5">
        <div class="space-y-1 text-center">
          <CardTitle class="ui-display-heading">{{
            i18n._('Reset password')
          }}</CardTitle>
          <p class="text-sm text-muted-foreground">{{ i18n._('Enter your email to receive a password reset link.') }}</p>
        </div>

        <form @submit.prevent="requestResetAction" novalidate class="space-y-3">
          <div class="space-y-2">
            <Label for="email" class="text-muted-foreground">{{
              i18n.ngettext('Email', 'Emails', 1)
            }}</Label>
            <Input
              id="email"
              type="email"
              v-model.trim="resetForm.email"
              :aria-invalid="emailHasError || undefined"
              :class="{ 'border-destructive': emailHasError }"
            />
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
              {{ i18n._('Sending...') }}
            </span>
            <span v-else>{{ i18n._('Send Reset Link') }}</span>
          </Button>
        </form>

        <Error
          v-if="errorMessage"
          :errorMessage="errorMessage"
          :border="true"
          class="w-full bg-destructive/10 text-destructive border-destructive/20 p-3 rounded-md text-sm"
        />

        <div class="text-center">
          <router-link
            to="/"
            class="text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            {{ i18n._('Back to login') }}
          </router-link>
        </div>
      </CardContent>
    </Card>
  </AuthLayout>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import { validateEmail } from '@shared-ui/utils/string'
import { applyTemporaryClass } from '@/utils/temporary-class'
import { Button } from '@shared-ui/components/ui/button'
import { Error } from '@shared-ui/components/ui/error'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { Label } from '@shared-ui/components/ui/label'
import { useI18n } from '@shared-ui/i18n'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'

const errorMessage = ref('')
const i18n = useI18n()
const isLoading = ref(false)
const submitted = ref(false)
const emitter = useEmitter()
const router = useRouter()
const resetForm = ref({
  email: ''
})

const validateForm = () => {
  if (!validateEmail(resetForm.value.email)) {
    errorMessage.value = i18n._('Invalid email address')
    applyTemporaryClass('reset-password-container', 'animate-shake')
    return false
  }
  return true
}

const requestResetAction = async () => {
  submitted.value = true
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true

  try {
    await api.resetPassword({
      email: resetForm.value.email
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Check your email for the password reset link.')
    })
    router.push({ name: 'login' })
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
    errorMessage.value = handleHTTPError(err).message
    applyTemporaryClass('reset-password-container', 'animate-shake')
  } finally {
    isLoading.value = false
  }
}

const emailHasError = computed(() => {
  return submitted.value && !validateEmail(resetForm.value.email)
})

watch(emailHasError, (hasError) => {
  if (!hasError && errorMessage.value === i18n._('Invalid email address')) {
    errorMessage.value = ''
  }
})
</script>
