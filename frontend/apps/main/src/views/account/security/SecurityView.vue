<template>
  <div class="max-w-xl space-y-6">
    <div class="space-y-2">
      <h2 class="sub-title">{{ t('twoFactor.title') }}</h2>
      <p class="text-sm text-muted-foreground">{{ t('twoFactor.description') }}</p>
    </div>
    <p v-if="loading" role="status">{{ t('globals.terms.loading') }}</p>
    <template v-else-if="loaded">
      <div v-if="stage === 'overview'" class="space-y-4">
        <p role="status" class="text-sm">
          {{ t(status.enabled ? 'globals.terms.enabled' : 'globals.terms.disabled') }}
        </p>
        <template v-if="status.enabled">
          <p class="text-sm text-muted-foreground">
            {{ t('twoFactor.codesRemaining', { count: status.recovery_codes_remaining }) }}
          </p>
          <div class="flex flex-wrap gap-3">
            <Button
              id="security-regenerate"
              type="button"
              variant="outline"
              @click="start('regenerate')"
            >
              {{ t('twoFactor.regenerateCodes') }}
            </Button>
            <Button type="button" variant="outline" @click="start('disable')">
              {{ t('globals.messages.disable') }}
            </Button>
          </div>
        </template>
        <Button v-else id="security-setup" type="button" @click="start('setup')">
          {{ t('twoFactor.setUp') }}
        </Button>
      </div>

      <form
        v-else-if="stage === 'password' || stage === 'scan'"
        @submit.prevent="submit"
        class="space-y-5"
      >
        <template v-if="stage === 'scan'">
          <p id="setup-hint" class="text-sm text-muted-foreground">{{ t('twoFactor.scanHint') }}</p>
          <img :src="setup.qr_code" :alt="t('twoFactor.qrAlt')" width="256" height="256" />
          <details class="text-sm">
            <summary class="cursor-pointer">{{ t('twoFactor.manualSetup') }}</summary>
            <p class="mt-2 text-muted-foreground">{{ t('twoFactor.manualHint') }}</p>
            <code class="mt-2 block break-all select-all">{{ setup.secret }}</code>
          </details>
        </template>
        <template v-else>
          <p class="text-sm text-muted-foreground">
            {{
              t(
                action === 'disable'
                  ? 'twoFactor.disableHint'
                  : action === 'regenerate'
                    ? 'twoFactor.regenerateHint'
                    : 'twoFactor.passwordHint'
              )
            }}
          </p>
          <div class="space-y-2">
            <Label for="security-password">{{ t('globals.terms.password') }}</Label>
            <Input
              id="security-password"
              v-model="password"
              type="password"
              autocomplete="current-password"
            />
          </div>
        </template>
        <div
          v-if="stage === 'scan' || (action !== 'setup' && status.verification_required)"
          class="space-y-2"
        >
          <Label for="security-code">{{
            t(stage === 'scan' ? 'globals.terms.authenticationCode' : 'twoFactor.codeOrRecovery')
          }}</Label>
          <Input
            id="security-code"
            v-model="code"
            type="text"
            :inputmode="stage === 'scan' ? 'numeric' : 'text'"
            autocomplete="one-time-code"
            :maxlength="stage === 'scan' ? 6 : 35"
            autocapitalize="none"
            :spellcheck="false"
            :aria-describedby="stage === 'scan' ? 'setup-hint security-error' : 'security-error'"
          />
        </div>
        <div class="flex gap-3">
          <Button
            type="submit"
            :isLoading="busy"
            :disabled="busy"
            :variant="action === 'disable' ? 'destructive' : 'default'"
          >
            {{
              t(
                stage === 'scan'
                  ? 'globals.messages.enable'
                  : action === 'disable'
                    ? 'globals.messages.disable'
                    : action === 'regenerate'
                      ? 'twoFactor.regenerateCodes'
                      : 'globals.terms.continue'
              )
            }}
          </Button>
          <Button type="button" variant="outline" :disabled="busy" @click="reset">
            {{ t('globals.messages.cancel') }}
          </Button>
        </div>
      </form>

      <div v-else-if="stage === 'recovery'" class="space-y-4">
        <h3 class="font-medium" tabindex="-1" id="recovery-heading">
          {{ t('twoFactor.saveCodes') }}
        </h3>
        <p class="text-sm text-muted-foreground">{{ t('twoFactor.saveCodesHint') }}</p>
        <pre class="rounded-md border p-4 text-sm overflow-x-auto select-all">{{
          recoveryCodes.join('\n')
        }}</pre>
        <div class="flex flex-wrap gap-3">
          <Button type="button" variant="outline" @click="downloadCodes">{{
            t('globals.terms.download')
          }}</Button>
          <Button type="button" @click="reset">{{ t('twoFactor.codesSaved') }}</Button>
        </div>
      </div>
    </template>
    <p id="security-error" role="alert" class="text-sm text-destructive">{{ error }}</p>
  </div>
</template>

<script setup>
import { nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'

const { t } = useI18n()
const loading = ref(true)
const loaded = ref(false)
const busy = ref(false)
const error = ref('')
const status = ref({ enabled: false, recovery_codes_remaining: 0, verification_required: true })
const stage = ref('overview')
const action = ref('setup')
const password = ref('')
const code = ref('')
const setup = ref({})
const recoveryCodes = ref([])

onMounted(async () => {
  try {
    status.value = (await api.getTwoFactor()).data.data
    loaded.value = true
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
})

const start = async (nextAction) => {
  action.value = nextAction
  stage.value = 'password'
  error.value = ''
  await nextTick()
  document.getElementById('security-password')?.focus()
}

const reset = async () => {
  stage.value = 'overview'
  password.value = ''
  code.value = ''
  setup.value = {}
  recoveryCodes.value = []
  error.value = ''
  await nextTick()
  document.getElementById(status.value.enabled ? 'security-regenerate' : 'security-setup')?.focus()
}

const submit = async () => {
  if (busy.value) return
  error.value = ''
  if (!password.value) {
    error.value = t('validation.passwordCannotBeEmpty')
    return
  }
  if (
    (stage.value === 'scan' || (action.value !== 'setup' && status.value.verification_required)) &&
    !code.value.trim()
  ) {
    error.value = t('twoFactor.enterCode')
    return
  }
  busy.value = true
  try {
    const data = { password: password.value, code: code.value.trim() }
    if (action.value === 'setup' && stage.value === 'password') {
      setup.value = (await api.setupTwoFactor(data)).data.data
      stage.value = 'scan'
      await nextTick()
      document.getElementById('security-code')?.focus()
      return
    }
    if (action.value === 'disable') {
      await api.disableTwoFactor(data)
      status.value = { enabled: false, recovery_codes_remaining: 0 }
      reset()
      return
    }
    const response =
      action.value === 'setup'
        ? await api.enableTwoFactor(data)
        : await api.regenerateTwoFactorCodes(data)
    recoveryCodes.value = response.data.data.recovery_codes
    status.value = {
      enabled: true,
      recovery_codes_remaining: recoveryCodes.value.length,
      verification_required: status.value.verification_required
    }
    password.value = ''
    code.value = ''
    setup.value = {}
    stage.value = 'recovery'
    await nextTick()
    document.getElementById('recovery-heading')?.focus()
  } catch (err) {
    error.value = handleHTTPError(err).message
    // Recent verification can expire while this form is open.
    if (action.value !== 'setup') {
      try {
        status.value = (await api.getTwoFactor()).data.data
      } catch {
        status.value.verification_required = true
      }
    }
  } finally {
    busy.value = false
  }
}

const downloadCodes = () => {
  const blob = new Blob([recoveryCodes.value.join('\n') + '\n'], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'libredesk-recovery-codes.txt'
  link.click()
  URL.revokeObjectURL(url)
}
</script>
