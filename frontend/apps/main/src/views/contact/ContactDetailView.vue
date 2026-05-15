<template>
  <ContactDetail>
    <div class="flex flex-col mx-auto items-start">
      <div class="mb-6" v-if="userStore.can('contacts:read_all')">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>

      <div
        v-if="contact"
        class="flex justify-center space-y-4 w-full"
        :class="{ 'loading-fade': formLoading }"
      >
        <div class="flex flex-col w-full mt-12">
          <div class="flex flex-col space-y-2">
            <AvatarUpload
              @upload="onUpload"
              @remove="onRemove"
              :src="contact.avatar_url || ''"
              :initials="getInitials"
              :label="t('globals.messages.upload')"
            />

            <div class="flex gap-2 justify-start items-center">
              <h2 class="text-2xl font-bold text-foreground">
                {{ contact.first_name }} {{ contact.last_name }}
              </h2>
              <Badge v-if="contact.type" variant="secondary">
                {{
                  contact.type === 'visitor'
                    ? $t('contact.type.visitor')
                    : $t('contact.type.contact')
                }}
              </Badge>
            </div>

            <div
              v-if="contact.external_user_id"
              class="flex items-center gap-1.5 text-xs text-muted-foreground"
            >
              <IdCardIcon size="14" class="flex-shrink-0" />
              {{ formatExternalId(contact.external_user_id) }}
            </div>

            <div
              v-if="telegramUsername"
              class="flex items-center gap-1.5 text-xs text-muted-foreground"
            >
              <SendIcon size="14" class="flex-shrink-0" />
              <a
                :href="'https://t.me/' + telegramUsername"
                target="_blank"
                rel="noopener noreferrer"
                class="hover:underline"
              >
                @{{ telegramUsername }}
              </a>
            </div>

            <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
              <CalendarIcon size="14" class="flex-shrink-0" />
              {{ $t('globals.terms.createdOn') }}
              {{ contact.created_at ? format(new Date(contact.created_at), 'PPP') : 'N/A' }}
            </div>

            <div class="w-30 pt-3">
              <Button
                :variant="contact.enabled ? 'destructive' : 'outline'"
                @click="showBlockConfirmation = true"
                size="sm"
              >
                <ShieldOffIcon v-if="contact.enabled" size="18" />
                <ShieldCheckIcon v-else size="18" />
                {{ t(contact.enabled ? 'globals.messages.block' : 'globals.messages.unblock') }}
              </Button>
            </div>
          </div>

          <div class="mt-12 space-y-10">
            <ContactForm :formLoading="formLoading" :onSubmit="onSubmit" />
            <ContactNotes :contactId="contact.id" v-if="userStore.can('contact_notes:read')" />
          </div>
        </div>
      </div>

      <Spinner v-if="formLoading" />

      <Dialog :open="showBlockConfirmation" @update:open="showBlockConfirmation = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader class="gap-y-3">
            <DialogTitle>
              {{ contact?.enabled ? t('contact.blockContact') : t('contact.unblockContact') }}
            </DialogTitle>
            <DialogDescription>
              {{ contact?.enabled ? t('contact.blockConfirm') : t('contact.unblockConfirm') }}
            </DialogDescription>
          </DialogHeader>
          <div class="flex justify-end space-x-2 pt-4">
            <Button variant="outline" @click="showBlockConfirmation = false">
              {{ t('globals.messages.cancel') }}
            </Button>
            <Button
              :variant="contact?.enabled ? 'destructive' : 'default'"
              @click="confirmToggleBlock"
            >
              {{ contact?.enabled ? t('globals.messages.block') : t('globals.messages.unblock') }}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  </ContactDetail>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { format } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { useUserStore } from '../../stores/user'
import { ShieldOffIcon, ShieldCheckIcon, IdCardIcon, CalendarIcon, SendIcon } from 'lucide-vue-next'
import ContactDetail from '@/layouts/contact/ContactDetail.vue'
import api from '../../api'
import ContactForm from '@/features/contact/ContactForm.vue'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import { createFormSchema } from '../../features/contact/formSchema.js'
import { useEmitter } from '../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'

const { t } = useI18n()
const emitter = useEmitter()
const route = useRoute()
const formLoading = ref(false)
const contact = ref(null)
const showBlockConfirmation = ref(false)
const userStore = useUserStore()

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const breadcrumbLinks = [
  { path: 'contacts', label: t('globals.terms.contact', 2) },
  { path: '', label: t('contact.editContact') }
]

onMounted(fetchContact)

async function fetchContact() {
  formLoading.value = true
  try {
    const { data } = await api.getContact(route.params.id)
    contact.value = data.data
    form.setValues(data.data)
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
}

const getInitials = computed(() => {
  if (!contact.value) return ''
  const { first_name = '', last_name = '' } = contact.value
  return `${first_name.charAt(0).toUpperCase()}${last_name.charAt(0).toUpperCase()}`
})

// Strip "telegram_" prefix from external_user_id for display.
function formatExternalId(extId) {
  if (!extId) return ''
  if (extId.startsWith('telegram_')) return extId.replace('telegram_', '')
  return extId
}

// Extract Telegram username from external_user_id pattern.
const telegramUsername = computed(() => {
  if (!contact.value?.external_user_id) return ''
  if (!contact.value.external_user_id.startsWith('telegram_')) return ''
  // Username is not stored in contact directly, but we can check custom_attributes or meta.
  // For now, we don't have it here. Will show only if available.
  return ''
})

async function confirmToggleBlock() {
  showBlockConfirmation.value = false
  await toggleBlock()
}

async function toggleBlock() {
  try {
    await api.blockContact(contact.value.id, {
      enabled: !contact.value.enabled
    })
    await fetchContact()
    emitToast(
      contact.value.enabled ? t('contact.unblockedSuccessfully') : t('contact.blockedSuccessfully')
    )
  } catch (err) {
    showError(err)
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await api.updateContact(contact.value.id, { ...values })
    await fetchContact()
    emitToast(t('globals.messages.savedSuccessfully'))
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
})

async function onUpload(file) {
  try {
    formLoading.value = true
    const formData = new FormData()
    formData.append('files', file)
    formData.append('first_name', form.values.first_name)
    formData.append('last_name', form.values.last_name)
    formData.append('email', form.values.email)
    formData.append('phone_number', form.values.phone_number)
    formData.append('phone_number_country_code', form.values.phone_number_country_code)
    formData.append('country', form.values.country || '')
    formData.append('enabled', form.values.enabled)
    const { data } = await api.updateContact(contact.value.id, formData)
    contact.value.avatar_url = data.avatar_url
    form.setFieldValue('avatar_url', data.avatar_url)
    emitToast(t('toast.avatarUpdated'))
    fetchContact()
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
}

async function onRemove() {
  contact.value.avatar_url = null
  form.setFieldValue('avatar_url', null)
  await onUpload(null)
}

function emitToast(description) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description })
}

function showError(err) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })
}
</script>
