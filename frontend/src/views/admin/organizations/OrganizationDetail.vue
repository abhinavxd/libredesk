<template>
  <div>
    <div class="mb-5">
      <CustomBreadcrumb :links="breadcrumbLinks" />
    </div>

    <Spinner v-if="isLoading" />

    <div v-else class="space-y-6">
      <!-- Organization Info Card -->
      <Card>
        <CardHeader>
          <div class="flex justify-between items-start">
            <div>
              <CardTitle class="text-2xl">{{ organization.name }}</CardTitle>
              <CardDescription v-if="organization.email_domain" class="mt-1">
                {{ organization.email_domain }}
              </CardDescription>
            </div>
            <div class="flex gap-2">
              <Button variant="outline" @click="editOrganization">
                <Pencil class="w-4 h-4 mr-2" />
                {{ $t('globals.messages.edit') }}
              </Button>
              <Button variant="destructive" @click="() => (deleteAlertOpen = true)">
                <Trash2 class="w-4 h-4 mr-2" />
                {{ $t('globals.messages.delete') }}
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div v-if="organization.website">
              <Label class="text-sm text-muted-foreground">{{
                $t('globals.terms.website')
              }}</Label>
              <a
                :href="organization.website"
                target="_blank"
                rel="noopener noreferrer"
                class="text-blue-600 hover:text-blue-800 hover:underline"
              >
                {{ organization.website }}
              </a>
            </div>

            <div v-if="organization.phone">
              <Label class="text-sm text-muted-foreground">{{ $t('globals.terms.phone') }}</Label>
              <p>{{ organization.phone }}</p>
            </div>

            <div>
              <Label class="text-sm text-muted-foreground">{{
                $t('globals.terms.createdAt')
              }}</Label>
              <p>{{ format(new Date(organization.created_at), 'PPpp') }}</p>
            </div>

            <div>
              <Label class="text-sm text-muted-foreground">{{
                $t('globals.terms.updatedAt')
              }}</Label>
              <p>{{ format(new Date(organization.updated_at), 'PPpp') }}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Member Contacts Card -->
      <Card>
        <CardHeader>
          <CardTitle>{{ $t('globals.terms.members') }}</CardTitle>
          <CardDescription>
            {{ $t('globals.messages.contactsInOrganization', { count: contacts.length }) }}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="contacts.length === 0" class="text-center py-8 text-muted-foreground">
            <Users class="w-12 h-12 mx-auto mb-2 opacity-50" />
            <p>{{ $t('globals.messages.noContactsInOrganization') }}</p>
          </div>

          <div v-else class="space-y-2">
            <div
              v-for="contact in contacts"
              :key="contact.id"
              class="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 cursor-pointer transition-colors"
              @click="viewContact(contact.id)"
            >
              <div class="flex items-center gap-3">
                <Avatar>
                  <AvatarImage :src="contact.avatar_url || ''" :alt="contact.first_name" />
                  <AvatarFallback>
                    {{ getInitials(contact.first_name, contact.last_name) }}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <p class="font-medium">{{ contact.first_name }} {{ contact.last_name }}</p>
                  <p class="text-sm text-muted-foreground">{{ contact.email }}</p>
                </div>
              </div>
              <Button variant="ghost" size="sm">
                <ChevronRight class="w-4 h-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Delete Confirmation Dialog -->
    <AlertDialog :open="deleteAlertOpen" @update:open="deleteAlertOpen = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ $t('globals.messages.deleteOrganizationConfirmation') }}
            <span v-if="contacts.length > 0" class="block mt-2 text-amber-600">
              {{ $t('globals.messages.organizationHasContacts', { count: contacts.length }) }}
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction @click="handleDelete" class="bg-red-600 hover:bg-red-700">
            {{ $t('globals.messages.delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { format } from 'date-fns'
import {
  Pencil,
  Trash2,
  Users,
  ChevronRight
} from 'lucide-vue-next'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import api from '@/api'

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const isLoading = ref(false)
const organization = ref({})
const contacts = ref([])
const deleteAlertOpen = ref(false)

const breadcrumbLinks = [
  { path: 'organization-list', label: t('globals.terms.organization', 2) },
  { path: '', label: organization.value.name || t('globals.terms.details') }
]

onMounted(async () => {
  await fetchOrganization()
})

const fetchOrganization = async () => {
  try {
    isLoading.value = true
    const resp = await api.getOrganization(props.id)
    organization.value = resp.data.data
    contacts.value = resp.data.data.contacts || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const editOrganization = () => {
  router.push({ name: 'edit-organization', params: { id: props.id } })
}

const viewContact = (contactId) => {
  router.push({ name: 'contact-detail', params: { id: contactId } })
}

const handleDelete = async () => {
  try {
    await api.deleteOrganization(props.id)
    deleteAlertOpen.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.deletedSuccessfully')
    })
    router.push({ name: 'organization-list' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const getInitials = (firstName, lastName) => {
  if (!firstName && !lastName) return ''
  if (!firstName) return lastName.charAt(0).toUpperCase()
  if (!lastName) return firstName.charAt(0).toUpperCase()
  return `${firstName.charAt(0).toUpperCase()}${lastName.charAt(0).toUpperCase()}`
}
</script>
