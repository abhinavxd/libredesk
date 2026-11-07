<template>
  <AuthLayout>
    <Card class="bg-card box">
      <CardContent class="p-6 space-y-5">
        <div class="space-y-2 text-center">
          <CardTitle class="ui-display-heading">{{ t('setup.createAdmin') }}</CardTitle>
          <p class="text-sm text-muted-foreground">{{ t('setup.codeHelp') }}</p>
        </div>
        <form @submit.prevent="submit" class="space-y-4">
          <div v-for="field in fields" :key="field.name" class="space-y-2">
            <Label :for="field.name">{{ t(field.label) }}</Label>
            <Input
              :id="field.name"
              v-model="form[field.name]"
              :type="field.type"
              :autocomplete="field.autocomplete"
              :maxlength="field.maxlength"
              :autocapitalize="field.name === 'email' ? 'none' : undefined"
              :spellcheck="field.name === 'email' || field.name === 'token' ? false : undefined"
              required
            />
          </div>
          <p class="text-sm text-muted-foreground">{{ t('setup.passwordHelp') }}</p>
          <Error v-if="error" :errorMessage="error" role="alert" />
          <Button type="submit" class="w-full" :isLoading="loading" :disabled="loading">
            {{ t('setup.createAdmin') }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </AuthLayout>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import AuthLayout from '@/layouts/auth/AuthLayout.vue'
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardContent, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Error } from '@shared-ui/components/ui/error'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const error = ref('')
const form = reactive({ token: '', first_name: '', email: '', password: '', confirmation: '' })
const fields = [
  { name: 'token', label: 'setup.code', type: 'text', autocomplete: 'off', maxlength: 40 },
  { name: 'first_name', label: 'globals.terms.name', type: 'text', autocomplete: 'given-name', maxlength: 100 },
  { name: 'email', label: 'globals.terms.email', type: 'email', autocomplete: 'username', maxlength: 320 },
  { name: 'password', label: 'globals.terms.password', type: 'password', autocomplete: 'new-password', maxlength: 72 },
  { name: 'confirmation', label: 'auth.confirmPassword', type: 'password', autocomplete: 'new-password', maxlength: 72 }
]

onMounted(async () => {
  try {
    const response = await api.getSetupStatus()
    if (!response.data.data.required) router.replace({ name: 'login' })
  } catch (err) {
    error.value = handleHTTPError(err).message
  }
})

const submit = async () => {
  if (loading.value) return
  error.value = ''
  if (form.password !== form.confirmation) {
    error.value = t('auth.passwordsDoNotMatch')
    return
  }
  loading.value = true
  try {
    await api.createSetupAccount({
      token: form.token.trim(), first_name: form.first_name.trim(), email: form.email.trim(), password: form.password
    })
    form.password = ''
    form.confirmation = ''
    form.token = ''
    await router.replace({ name: 'login' })
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
}
</script>
