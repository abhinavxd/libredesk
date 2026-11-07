<template>
  <div class="flex h-full min-h-0 flex-col items-center justify-center overflow-y-auto rounded-xl border border-border bg-card p-6">
    <Spinner v-if="loading && !checklist" />
    <div v-else-if="showGettingStarted" class="w-full max-w-lg space-y-5">
      <h2 class="ui-display-heading">{{ t('setup.completeYourSetup') }}</h2>
      <p class="text-sm text-muted-foreground">{{ t('setup.checklistHelp') }}</p>
      <ol class="space-y-3">
        <li v-for="step in steps" :key="step.key" class="flex items-start gap-3 rounded-lg border p-3">
          <CheckCircle v-if="step.done" class="mt-0.5 size-5 shrink-0 text-primary" aria-hidden="true" />
          <Circle v-else class="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden="true" />
          <div class="min-w-0 flex-1 space-y-1">
            <p class="text-sm font-medium">
              {{ t(step.label) }}
              <span v-if="step.done" class="text-muted-foreground">({{ t('setup.complete') }})</span>
            </p>
            <p v-if="step.help" class="text-sm text-muted-foreground">{{ t(step.help) }}</p>
            <div v-if="!step.done" class="flex flex-wrap gap-2 pt-1">
              <Button type="button" variant="outline" size="sm" @click="router.push(step.to)">
                {{ t('globals.messages.setUp') }}
              </Button>
              <Button v-if="step.key === 'notifications'" type="button" variant="ghost" size="sm" :disabled="loading" @click="skipNotifications">
                {{ t('setup.skipForNow') }}
              </Button>
            </div>
          </div>
        </li>
      </ol>
      <p class="text-sm text-muted-foreground">{{ t('setup.teammatesOptional') }}</p>
      <div class="flex flex-wrap gap-2">
        <Button type="button" variant="outline" size="sm" @click="router.push({ name: 'agent-list' })">
          {{ t('setup.inviteTeammates') }}
        </Button>
        <Button type="button" variant="ghost" size="sm" :disabled="loading" @click="refresh">
          {{ t('setup.checkProgress') }}
        </Button>
      </div>
    </div>
    <p v-else-if="!error" class="max-w-sm text-center text-sm text-muted-foreground">{{ t('conversation.placeholder') }}</p>
    <Error v-if="error" :errorMessage="error" role="alert" class="mt-4 max-w-lg" />
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { CheckCircle, Circle } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Error } from '@shared-ui/components/ui/error'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useUserStore } from '@/stores/user'
import api from '@/api'

const { t } = useI18n()
const router = useRouter()
const userStore = useUserStore()
const loading = ref(false)
const error = ref('')
const checklist = ref(null)
const steps = computed(() => {
  const status = checklist.value || {}
  const items = [
    { key: 'workspace', label: 'setup.workspace', help: 'setup.workspaceHelp', done: status.workspace, to: { name: 'general' } },
    { key: 'inbox', label: 'setup.createFirstInbox', done: status.inbox, to: { name: 'inbox-list' } },
    { key: 'notifications', label: 'setup.notifications', help: 'setup.notificationsHelp', done: status.notifications, to: '/admin/notification' },
    { key: 'conversation', label: 'setup.testConversation', help: 'setup.testConversationHelp', done: Boolean(status.conversation), to: { name: 'inbox-list' } }
  ]
  return status.managed ? items : items.filter((step) => step.key === 'inbox')
})
const showGettingStarted = computed(() => checklist.value && steps.value.some((step) => !step.done))

const refresh = async () => {
  if (!userStore.can('general_settings:manage')) return
  loading.value = true
  error.value = ''
  try {
    const response = await api.getSetupChecklist()
    checklist.value = response.data.data
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
}

const skipNotifications = async () => {
  loading.value = true
  try {
    await api.skipSetupNotifications()
    await refresh()
  } catch (err) {
    error.value = handleHTTPError(err).message
  } finally {
    loading.value = false
  }
}

watch(() => userStore.can('general_settings:manage'), (canManage) => {
  if (canManage) refresh()
}, { immediate: true })
</script>
