<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
    <WebhookForm @submit.prevent="onSubmit" :form="form" :isNewForm="isNewForm">
      <template #footer>
        <div class="flex space-x-3">
          <Button type="submit" :isLoading="formLoading">
            {{ isNewForm ? t('globals.messages.create') : t('globals.messages.update') }}
          </Button>
          <Button
            v-if="!isNewForm"
            type="button"
            variant="outline"
            :isLoading="testLoading"
            @click="handleTestWebhook"
          >
            {{
              $t('globals.messages.send', {
                name: t('globals.terms.test').toLowerCase()
              })
            }}
          </Button>
        </div>
      </template>
    </WebhookForm>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import WebhookForm from '@/features/admin/webhooks/WebhookForm.vue'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '@/features/admin/webhooks/formSchema.js'

const router = useRouter()
const { t } = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const formLoading = ref(false)
const testLoading = ref(false)

const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    name: '',
    url: '',
    events: [],
    secret: '',
    is_active: true,
    headers: '{}'
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true

    let toastDescription = ''
    if (props.id) {
      // If secret contains dummy characters, clear it so backend knows to keep existing secret
      if (values.secret && values.secret.includes('•')) {
        values.secret = ''
      }
      await api.updateWebhook(props.id, values)
      toastDescription = t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.webhook')
      })
    } else {
      await api.createWebhook(values)
      router.push({ name: 'webhook-list' })
      toastDescription = t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.webhook')
      })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: toastDescription
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})

const handleTestWebhook = async () => {
  if (!props.id) return

  try {
    testLoading.value = true
    await api.testWebhook(props.id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: t('globals.messages.sentSuccessfully', {
        name: t('globals.terms.webhook')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    testLoading.value = false
  }
}

const breadCrumLabel = () => {
  return props.id ? t('globals.messages.edit') : t('globals.messages.new')
}

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: 'webhook-list', label: t('globals.terms.webhook') },
  { path: '', label: breadCrumLabel() }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getWebhook(props.id)
      form.setValues(resp.data.data)
      // The secret is already masked by the backend, no need to modify it here
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  }
})
</script>
