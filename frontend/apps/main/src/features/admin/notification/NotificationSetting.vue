<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <NotificationsForm :initial-values="initialValues" :submit-form="submitForm" />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ i18n._('Configure SMTP server settings for sending email notifications to team members.') }}</p>
      <p>{{ i18n._('Once configured, teammates receive automated alerts for conversation assignments, SLA breaches, and other important events.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@main/api'
import AdminSplitLayout from '@main/layouts/admin/AdminSplitLayout.vue'
import { useI18n } from '@shared-ui/i18n'
import NotificationsForm from './NotificationSettingForm.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useAppSettingsStore } from '@main/stores/appSettings'

const initialValues = ref({})
const i18n = useI18n()
const isLoading = ref(false)
const emitter = useEmitter()
const appSettingsStore = useAppSettingsStore()

onMounted(() => {
  getNotificationSettings()
})

const getNotificationSettings = async () => {
  try {
    isLoading.value = true
    const resp = await api.getEmailNotificationSettings()
    initialValues.value = Object.fromEntries(
      Object.entries(resp.data.data).map(([key, value]) => [
        key.replace('notification.email.', ''),
        value
      ])
    )
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const submitForm = async (values) => {
  try {
    const updatedValues = Object.fromEntries(
      Object.entries(values).map(([key, value]) => {
        if (key === 'password' && value.includes('•')) {
          return [`notification.email.${key}`, '']
        }
        return [`notification.email.${key}`, value]
      })
    )
    await api.updateEmailNotificationSettings(updatedValues)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Settings updated successfully, Please restart the app for changes to take effect.')
    })
    await getNotificationSettings()
    appSettingsStore.fetchSettings()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
