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
              :label="i18n._('Upload')"
            />

            <div class="flex gap-2 justify-start items-center">
              <h2 class="ui-display-heading">
                {{ contact.first_name }} {{ contact.last_name }}
              </h2>
              <Badge v-if="contact.type" variant="secondary">
                {{
                  contact.type === 'visitor'
                    ? i18n.pgettext('Contact type', 'Visitor')
                    : i18n.pgettext('Contact type', 'Contact')
                }}
              </Badge>
              <Badge v-if="!contact.enabled" variant="destructive" class="gap-1">
                <ShieldOffIcon size="12" />
                {{ i18n._('Blocked') }}
              </Badge>
              <DropdownMenu v-if="canOpenActionsMenu">
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" class="h-7 w-7">
                    <MoreVerticalIcon class="h-4 w-4" />
                    <span class="sr-only">{{ i18n._('Open menu') }}</span>
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
                    {{ contact.enabled ? i18n._('Block') : i18n._('Unblock') }}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    v-if="userStore.can('contacts:export')"
                    class="cursor-pointer"
                    @click="exportContact"
                  >
                    <DownloadIcon class="mr-2" size="15" />
                    {{ i18n._('Export data') }}
                  </DropdownMenuItem>
                  <template v-if="userStore.can('contacts:delete')">
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      class="text-destructive cursor-pointer"
                      @click="showDeleteConfirmation = true"
                    >
                      <Trash2Icon class="mr-2" size="15" />
                      {{ i18n._('Delete contact') }}
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

            <div
              v-for="identity in contact.channel_identities"
              :key="identity.channel + identity.identifier"
              class="flex items-center gap-1.5 text-xs text-muted-foreground"
            >
              <WhatsAppIcon
                v-if="identity.channel === 'whatsapp'"
                class="size-3.5 flex-shrink-0"
              />
              <IdCardIcon v-else size="14" class="flex-shrink-0" />
              {{ identity.channel === 'whatsapp' ? '+' + identity.identifier : identity.identifier }}
            </div>

            <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
              <CalendarIcon size="14" class="flex-shrink-0" />
              {{ i18n._('Created on') }}
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

      <AlertDialog :open="showBlockConfirmation" @update:open="(v) => (showBlockConfirmation = v)">
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {{ contact?.enabled ? i18n._('Block contact') : i18n._('Unblock contact') }}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {{ contact?.enabled ? i18n._('Are you sure you want to block this contact? They won\'\'t be able to chat, and incoming emails from their address will be rejected. This also blocks incoming emails from any other contacts that use the same email.') : i18n._('Are you sure you want to unblock this contact? They can chat again, and incoming emails from their address will be accepted.') }}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
            <AlertDialogAction
              :variant="contact?.enabled ? 'destructive' : 'default'"
              @click="confirmToggleBlock"
            >
              {{ contact?.enabled ? i18n._('Block') : i18n._('Unblock') }}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        :open="showDeleteConfirmation"
        @update:open="(v) => (showDeleteConfirmation = v)"
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{{ i18n._('Delete contact') }}</AlertDialogTitle>
            <AlertDialogDescription>{{ i18n._('Are you sure you want to delete this contact? All their conversations, messages, notes, and attachments will be permanently deleted. This cannot be undone.') }}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" @click="confirmDelete">
              {{ i18n._('Delete') }}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  </ContactDetail>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { format } from 'date-fns'
import { useI18n } from '@shared-ui/i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator
} from '@shared-ui/components/ui/dropdown-menu'
import { useUserStore } from '@/stores/user'
import { useContactStore } from '@/stores/contact'
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
import api from '@/api'
import ContactForm from '@/features/contact/ContactForm.vue'
import ContactNotes from '@/features/contact/ContactNotes.vue'
import WhatsAppIcon from '@/components/icons/WhatsAppIcon.vue'
import { createFormSchema } from '@/features/contact/formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS, CONTACT_ACTIONS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { downloadBlobResponse, parseBlobError } from '@shared-ui/utils/file'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'

const i18n = useI18n()
const emitter = useEmitter()
const route = useRoute()
const router = useRouter()
const formLoading = ref(false)
const contact = ref(null)
const showBlockConfirmation = ref(false)
const showDeleteConfirmation = ref(false)
const userStore = useUserStore()
const contactStore = useContactStore()

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n))
})

const canOpenActionsMenu = computed(
  () =>
    userStore.can('contacts:block') ||
    userStore.can('contacts:export') ||
    userStore.can('contacts:delete')
)

const breadcrumbLinks = [
  { path: 'contacts', label: i18n._('{count, plural, one {Contact} other {Contacts}}', 2) },
  { path: '', label: i18n._('Edit contact') }
]

const paletteActions = {
  [CONTACT_ACTIONS.TOGGLE_BLOCK]: () => (showBlockConfirmation.value = true),
  [CONTACT_ACTIONS.EXPORT]: () => exportContact(),
  [CONTACT_ACTIONS.DELETE]: () => (showDeleteConfirmation.value = true)
}
const onPaletteAction = (action) => paletteActions[action]?.()

onMounted(() => {
  emitter.on(EMITTER_EVENTS.CONTACT_ACTION, onPaletteAction)
  fetchContact()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.CONTACT_ACTION, onPaletteAction)
  contactStore.clearCurrent()
})

async function fetchContact() {
  formLoading.value = true
  try {
    const { data } = await api.getContact(route.params.id)
    contact.value = data.data
    contactStore.setCurrent(data.data)
    form.setValues(data.data, false)
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
      contact.value.enabled ? i18n._('Contact unblocked successfully') : i18n._('Contact blocked successfully')
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
    emitToast(i18n._('Deleted successfully'))
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
    emitToast(i18n._('Changes saved'))
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
    emitToast(i18n._('Avatar updated successfully'))
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
