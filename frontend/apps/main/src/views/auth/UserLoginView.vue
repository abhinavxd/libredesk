<template>
  <AuthLayout>
    <Card
      class="bg-card box"
      :class="{ 'animate-shake': shakeCard }"
      id="login-container"
      ref="cardRef"
    >
      <CardContent class="p-6 space-y-5">
        <div class="space-y-1 text-center">
          <CardTitle class="ui-display-heading">
            {{ appSettingsStore.public_config?.['app.site_name'] || 'libredesk' }}
          </CardTitle>
          <p class="text-sm text-muted-foreground">{{ i18n._('Sign in to your account') }}</p>
        </div>

        <div v-if="enabledOIDCProviders.length" class="space-y-3">
          <Button
            v-for="oidcProvider in enabledOIDCProviders"
            :key="oidcProvider.id"
            variant="outline"
            type="button"
            @click="redirectToOIDC(oidcProvider)"
            class="w-full"
          >
            <img
              :src="oidcProvider.logo_url"
              :alt="oidcProvider.name"
              width="20"
              v-if="oidcProvider.logo_url"
            />
            {{ oidcProvider.name }}
          </Button>

          <div v-if="localLoginEnabled" class="relative">
            <div class="absolute inset-0 flex items-center">
              <span class="w-full border-t border-border"></span>
            </div>
            <div class="relative flex justify-center text-xs uppercase">
              <span class="px-2 text-muted-foreground bg-card">{{ i18n._('Or continue with') }}</span>
            </div>
          </div>
        </div>

        <form v-if="localLoginEnabled" @submit.prevent="loginAction" class="space-y-3">
          <div class="space-y-2">
            <Label for="email" class="text-muted-foreground">{{ i18n.ngettext('Email', 'Emails', 1) }}</Label>
            <Input
              id="email"
              type="text"
              autofocus
              autocomplete="username"
              autocapitalize="none"
              spellcheck="false"
              :aria-invalid="emailHasError || undefined"
              v-model.trim="loginForm.email"
              :class="{ 'border-destructive': emailHasError }"
            />
          </div>

          <div class="space-y-2">
            <Label for="password" class="text-muted-foreground">
              {{ i18n.ngettext('Password', 'Passwords', 1) }}
            </Label>
            <div class="relative">
              <Input
                id="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                :aria-invalid="passwordHasError || undefined"
                v-model="loginForm.password"
                :class="{ 'border-destructive': passwordHasError }"
                class="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                :aria-label="showPassword ? i18n._('Hide password') : i18n._('Show password')"
                class="absolute right-0 top-0 text-muted-foreground"
                @click="showPassword = !showPassword"
              >
                <Eye v-if="!showPassword" class="w-5 h-5" aria-hidden="true" />
                <EyeOff v-else class="w-5 h-5" aria-hidden="true" />
              </Button>
            </div>
          </div>

          <div class="flex items-center justify-between">
            <router-link
              to="/reset-password"
              class="text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              {{ i18n._('Forgot password?') }}
            </router-link>
          </div>

          <Button class="w-full" :disabled="isLoading" type="submit">
            <span v-if="isLoading" class="flex items-center justify-center">
              <div
                class="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin mr-3"
              ></div>
              {{ i18n._('Logging in...') }}
            </span>
            <span v-else>{{ i18n._('Sign in') }}</span>
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
import { defineMessage as msg } from '@lingui/core/macro'

import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import { validateEmail } from '@shared-ui/utils/string'
import { applyTemporaryClass } from '@/utils/temporary-class'
import { Button } from '@shared-ui/components/ui/button'
import { Error } from '@shared-ui/components/ui/error'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { useEmitter } from '@main/composables/useEmitter'
import { useUserStore } from '@main/stores/user'
import { useI18n } from '@shared-ui/i18n'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useAppSettingsStore } from '@main/stores/appSettings'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'
import { Eye, EyeOff } from 'lucide-vue-next'

const emitter = useEmitter()
const i18n = useI18n()
const errorMessage = ref('')
const isLoading = ref(false)
const router = useRouter()
const userStore = useUserStore()
const shakeCard = ref(false)
const submitted = ref(false)
const showPassword = ref(false)
const loginForm = ref({
  email: '',
  password: ''
})
const oidcProviders = ref([])
const appSettingsStore = useAppSettingsStore()

// Demo build has the credentials prefilled.
const isDemoBuild = import.meta.env.VITE_DEMO_BUILD === 'true'

const demoCredentials = {
  email: 'demo@libredesk.io',
  password: 'demo@libredesk.io'
}

const oidcErrorMessages = {
  oidc_invalid_client: msg({ message: 'Sign-in is unavailable due to a configuration problem. Ask your administrator to check the SSO client ID and secret.' }),
  oidc_access_denied: msg({ message: 'The sign-in was cancelled or denied at the identity provider. Try signing in again.' }),
  oidc_session_expired: msg({ message: 'Your sign-in session expired. Try signing in again.' }),
  oidc_no_account: msg({ message: 'There is no agent account for the email your SSO provider returned. Contact your administrator for access.' }),
  oidc_account_disabled: msg({ message: 'Your account is disabled, Please contact administrator' }),
  oidc_login_failed: msg({ message: 'Sign-in failed. Try again, and contact your administrator if it keeps happening.' })
}

onMounted(async () => {
  // Prefill the login form with demo credentials if it's a demo build
  if (isDemoBuild) {
    loginForm.value.email = demoCredentials.email
    loginForm.value.password = demoCredentials.password
  }
  fetchOIDCProviders()
  showOIDCError()
})

const showOIDCError = () => {
  const { error, ...query } = router.currentRoute.value.query
  if (!error) return
  errorMessage.value = i18n._(
    Object.hasOwn(oidcErrorMessages, error) ? oidcErrorMessages[error] : msg({ message: 'Sign-in failed. Try again, and contact your administrator if it keeps happening.' })
  )
  applyTemporaryClass('login-container', 'animate-shake')
  router.replace({ query })
}

const fetchOIDCProviders = async () => {
  try {
    const config = appSettingsStore.public_config
    if (config && config['app.sso_providers']) {
      oidcProviders.value = config['app.sso_providers'] || []
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const redirectToOIDC = (provider) => {
  // Pass the 'next' parameter to OIDC login if it exists
  const nextParam = router.currentRoute.value.query.next
  const url = nextParam
    ? `/api/v1/oidc/${provider.id}/login?next=${encodeURIComponent(nextParam)}`
    : `/api/v1/oidc/${provider.id}/login`
  window.location.href = url
}

const validateForm = () => {
  if (!validateEmail(loginForm.value.email) && loginForm.value.email !== 'System') {
    errorMessage.value = i18n._('Invalid email address')
    applyTemporaryClass('login-container', 'animate-shake')
    return false
  }
  if (!loginForm.value.password) {
    errorMessage.value = i18n._('Password cannot be empty')
    applyTemporaryClass('login-container', 'animate-shake')
    return false
  }
  return true
}

const loginAction = () => {
  submitted.value = true
  if (!validateForm()) return

  errorMessage.value = ''
  isLoading.value = true

  api
    .login({
      email: loginForm.value.email,
      password: loginForm.value.password
    })
    .then((resp) => {
      if (resp?.data?.data) {
        userStore.setCurrentUser(resp.data.data)
      }
      // Also fetch general setting as user's logged in.
      appSettingsStore.fetchSettings('general')

      // Redirect to the 'next' parameter if it exists
      const nextParam = router.currentRoute.value.query.next
      if (nextParam) {
        router.push(nextParam)
      } else {
        router.push({ name: 'inboxes' })
      }
    })
    .catch((error) => {
      errorMessage.value = handleHTTPError(error).message
      applyTemporaryClass('login-container', 'animate-shake')
    })
    .finally(() => {
      isLoading.value = false
    })
}

const enabledOIDCProviders = computed(() => {
  return oidcProviders.value.filter((provider) => !provider.disabled)
})

const localLoginEnabled = computed(
  () => appSettingsStore.public_config?.['auth.local_login_enabled'] !== false
)

const emailHasError = computed(() => {
  if (!submitted.value) return false
  const email = loginForm.value.email
  return email !== 'System' && !validateEmail(email)
})

const passwordHasError = computed(() => submitted.value && !loginForm.value.password)
</script>
