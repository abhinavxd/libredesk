<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <Spinner v-if="isLoading" />
        <NotificationsForm :initial-values="initialValues" :submit-form="submitForm" />
      </div>
    </template>

    <template #help>
      <p>Configure SMTP server settings for sending email notifications to team members.</p>
      <p>
        Once configured, teammates receive automated alerts for conversation assignments, SLA
        breaches, and other important events.
      </p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'

import NotificationsForm from './NotificationSettingForm.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'

const initialValues = ref({})
const isLoading = ref(false)
const emitter = useEmitter()

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
      title: 'Error',
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
      title: 'Success',
      description:
        'Settings updated successfully, Please restart the app for changes to take effect.'
    })
    await getNotificationSettings()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
