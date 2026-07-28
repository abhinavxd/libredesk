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

            <div class="flex gap-2 items-center">
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
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" class="h-7 w-7">
                    <MoreVerticalIcon class="h-4 w-4" />
                    <span class="sr-only">{{ t('globals.terms.openMenu') }}</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" class="w-[200px]">
                  <DropdownMenuItem
                    v-if="userStore.can('contacts:block')"
                    class="cursor-pointer"
                    @click="showBlockConfirmation = true"
                  >
                    <ShieldOffIcon v-if="contact.enabled" class="mr-2" size="15" />
                    <ShieldCheckIcon v-else class="mr-2" size="15" />
                    {{ t(contact.enabled ? 'globals.messages.block' : 'globals.messages.unblock') }}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    v-if="userStore.can('contacts:export')"
                    class="cursor-pointer"
                    @click="exportContact"
                  >
                    <DownloadIcon class="mr-2" size="15" />
                    {{ t('contact.exportData') }}
                  </DropdownMenuItem>
                  <template v-if="userStore.can('contacts:delete')">
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      class="text-destructive cursor-pointer"
                      @click="showDeleteConfirmation = true"
                    >
                      <Trash2Icon class="mr-2" size="15" />
                      {{ t('contact.deleteContact') }}
                    </DropdownMenuItem>
                  </template>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <div
              v-if="contact.external_user_id"
              class="flex items-center gap-1.5 text-xs text-muted-foreground"
            >
              <IdCardIcon size="14" class="flex-shrink-0" />
              {{ contact.external_user_id }}
            </div>

            <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
              <CalendarIcon size="14" class="flex-shrink-0" />
              {{ $t('globals.terms.createdOn') }}
              {{ contact.created_at ? format(new Date(contact.created_at), 'PPP') : 'N/A' }}
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

      <Dialog :open="showDeleteConfirmation" @update:open="showDeleteConfirmation = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader class="gap-y-3">
            <DialogTitle>{{ t('contact.deleteContact') }}</DialogTitle>
            <DialogDescription>{{ t('contact.deleteConfirm') }}</DialogDescription>
          </DialogHeader>
          <div class="flex justify-end space-x-2 pt-4">
            <Button variant="outline" @click="showDeleteConfirmation = false">
              {{ t('globals.messages.cancel') }}
            </Button>
            <Button variant="destructive" @click="confirmDelete">
              {{ t('globals.messages.delete') }}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  </ContactDetail>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator
} from '@shared-ui/components/ui/dropdown-menu'
import { useUserStore } from '../../stores/user'
import {
  ShieldOffIcon,
  ShieldCheckIcon,
  IdCardIcon,
  CalendarIcon,
  DownloadIcon,
  Trash2Icon,
  MoreVerticalIcon
} from 'lucide-vue-next'
import ContactDetail from '@/layouts/contact/ContactDetail.vue'
import api from '../../api'
import ContactForm from '@/features/contact/ContactForm.vue'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import { createFormSchema } from '../../features/contact/formSchema.js'
import { useEmitter } from '../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { downloadBlobResponse, parseBlobError } from '@shared-ui/utils/file'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'

const { t } = useI18n()
const emitter = useEmitter()
const route = useRoute()
const router = useRouter()
const formLoading = ref(false)
const contact = ref(null)
const showBlockConfirmation = ref(false)
const showDeleteConfirmation = ref(false)
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

async function confirmDelete() {
  showDeleteConfirmation.value = false
  try {
    formLoading.value = true
    await api.deleteContact(contact.value.id)
    emitToast(t('contact.deletedSuccessfully'))
    router.push({ name: 'contacts' })
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
}

async function exportContact() {
  try {
    const response = await api.exportContact(contact.value.id)
    downloadBlobResponse(response, `contact-${contact.value.id}-data.json`)
  } catch (err) {
    showError(await parseBlobError(err))
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
