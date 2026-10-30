<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading">
    <ContextLinkForm @submit.prevent="onSubmit" :form="form" :isNewForm="isNewForm">
      <template #footer>
        <div class="flex space-x-3">
          <Button type="submit" :isLoading="formLoading">
            {{ isNewForm ? i18n._('Create') : i18n._('Save') }}
          </Button>
        </div>
      </template>
    </ContextLinkForm>
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '@/api'
import ContextLinkForm from '@/features/admin/context-links/ContextLinkForm.vue'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Button } from '@shared-ui/components/ui/button'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import { useRouter } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '@/features/admin/context-links/formSchema.js'

const router = useRouter()
const i18n = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const formLoading = ref(false)

const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: {
    name: '',
    url_template: '',
    secret: '',
    token_expiry_seconds: 1200,
    is_active: true
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true

    if (props.id) {
      await api.updateContextLink(props.id, values)
    } else {
      await api.createContextLink(values)
      router.push({ name: 'context-link-list' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: i18n._('Changes saved')
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

const isNewForm = computed(() => !props.id)

const breadcrumbLinks = [
  { path: 'context-link-list', label: i18n._('{count, plural, one {Context link} other {Context links}}') },
  { path: '', label: props.id ? i18n._('Edit') : i18n.pgettext('New context link', 'New') }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getContextLink(props.id)
      form.setValues(resp.data.data, false)
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
