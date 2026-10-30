<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem v-auto-animate>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" :placeholder="i18n.ngettext('Agent', 'Agents', 1)" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="description">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Description', 'Descriptions', 1) }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="i18n._('Role for all support agents')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <div>
      <div class="mb-5 text-lg">{{ i18n._('Set permissions for this role') }}</div>

      <div class="space-y-6">
        <div
          v-for="entity in permissions"
          :key="entity.name"
          class="rounded-md border border-border bg-card"
        >
          <div class="border-b border-border bg-muted/30 px-5 py-3">
            <h4 class="ui-subheading">{{ entity.name }}</h4>
          </div>

          <div class="p-5">
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <FormField
                v-for="permission in entity.permissions"
                :key="permission.name"
                :name="permission.name"
              >
                <FormItem class="flex items-start space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox
                      :checked="selectedPermissions.includes(permission.name)"
                      @update:checked="(newValue) => handleChange(newValue, permission.name)"
                    />
                  </FormControl>
                  <FormLabel class="font-normal text-sm">{{ permission.label }}</FormLabel>
                </FormItem>
              </FormField>
            </div>
          </div>
        </div>
      </div>
    </div>

    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'
const readMessage = msg({ message: "View conversation", context: "Conversation permission" })


import FormGrid from '@shared-ui/components/FormGrid.vue'
import { watch, ref, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { useI18n } from '@shared-ui/i18n'
import { permissions as perms } from '@main/constants/permissions.js'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => ''
  },
  isNewForm: {
    type: Boolean,
    default: false
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})

const i18n = useI18n()

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? i18n._('Create') : i18n._('Save'))
  )
})

// TODO: Prepare this by fetching all perms from the file, so we don't have to update this manually.
const permissions = ref([
  {
    name: i18n._('{count, plural, one {Conversation} other {Conversations}}'),
    permissions: [
      { name: perms.CONVERSATIONS_READ, label: i18n._(readMessage) },
      { name: perms.CONVERSATIONS_WRITE, label: i18n._('Create conversation') },
      {
        name: perms.CONVERSATIONS_READ_ASSIGNED,
        label: i18n._('View conversations assigned to me')
      },
      { name: perms.CONVERSATIONS_READ_ALL, label: i18n._('View all conversations') },
      {
        name: perms.CONVERSATIONS_READ_UNASSIGNED,
        label: i18n._('View all unassigned conversations')
      },
      {
        name: perms.CONVERSATIONS_READ_TEAM_INBOX,
        label: i18n._('View conversations in team inbox')
      },
      {
        name: perms.CONVERSATIONS_READ_TEAM_ALL,
        label: i18n._('View your team conversations')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        label: i18n._('Assign conversations to users')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        label: i18n._('Assign conversations to teams')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_PRIORITY,
        label: i18n._('Change conversation priority')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_STATUS,
        label: i18n._('Change conversation status')
      },
      { name: perms.CONVERSATIONS_UPDATE_TAGS, label: i18n._('Add or remove conversation tags') },
      { name: perms.MESSAGES_READ, label: i18n._('View conversation messages') },
      { name: perms.MESSAGES_WRITE, label: i18n._('Send messages in conversations') },
      { name: perms.MESSAGES_WRITE_PRIVATE, label: i18n._('Send private notes in conversations') },
      { name: perms.MESSAGES_WRITE_AS_CONTACT, label: i18n._('Send messages as contact') },
      { name: perms.VIEW_MANAGE, label: i18n._('Create and manage conversation views') }
    ]
  },
  {
    name: i18n._('{count, plural, one {Admin} other {Admins}}'),
    permissions: [
      { name: perms.GENERAL_SETTINGS_MANAGE, label: i18n._('Manage general settings') },
      {
        name: perms.NOTIFICATION_SETTINGS_MANAGE,
        label: i18n._('Manage notification settings')
      },
      { name: perms.STATUS_MANAGE, label: i18n._('Manage conversation statuses') },
      { name: perms.OIDC_MANAGE, label: i18n._('Manage SSO configuration') },
      { name: perms.TAGS_MANAGE, label: i18n._('Manage tags') },
      { name: perms.MACROS_MANAGE, label: i18n._('Manage macros') },
      { name: perms.USERS_MANAGE, label: i18n._('Manage users') },
      { name: perms.TEAMS_MANAGE, label: i18n._('Manage teams') },
      { name: perms.AUTOMATIONS_MANAGE, label: i18n._('Manage automations') },
      { name: perms.INBOXES_MANAGE, label: i18n._('Manage inboxes') },
      { name: perms.ROLES_MANAGE, label: i18n._('Manage roles') },
      { name: perms.TEMPLATES_MANAGE, label: i18n._('Manage templates') },
      { name: perms.REPORTS_MANAGE, label: i18n._('Manage reports') },
      { name: perms.BUSINESS_HOURS_MANAGE, label: i18n._('Manage business hours') },
      { name: perms.SLA_MANAGE, label: i18n._('Manage SLA policies') },
      { name: perms.AI_MANAGE, label: i18n._('Manage AI features') },
      { name: perms.HELP_CENTER_MANAGE, label: i18n._('Manage help centers') },
      { name: perms.CUSTOM_ATTRIBUTES_MANAGE, label: i18n._('Manage custom attributes') },
      { name: perms.ACTIVITY_LOGS_MANAGE, label: i18n._('Manage activity log') },
      { name: perms.WEBHOOKS_MANAGE, label: i18n._('Manage webhooks') },
      { name: perms.SHARED_VIEWS_MANAGE, label: i18n._('Manage shared views') },
      { name: perms.CONTEXT_LINKS_MANAGE, label: i18n._('Manage context links') }
    ]
  },
  {
    name: i18n._('{count, plural, one {Contact} other {Contacts}}'),
    permissions: [
      { name: perms.CONTACTS_READ_ALL, label: i18n._('View all contacts') },
      { name: perms.CONTACTS_READ, label: i18n._('View contact details') },
      { name: perms.CONTACTS_WRITE, label: i18n._('Edit contact details') },
      { name: perms.CONTACTS_BLOCK, label: i18n._('Block contacts') },
      { name: perms.CONTACTS_DELETE, label: i18n._('Delete contacts') },
      { name: perms.CONTACTS_EXPORT, label: i18n._('Export contact data') },
      { name: perms.CONTACT_NOTES_READ, label: i18n._('View contact notes') },
      { name: perms.CONTACT_NOTES_WRITE, label: i18n._('Add contact notes') },
      { name: perms.CONTACT_NOTES_DELETE, label: i18n._('Delete contact notes') }
    ]
  }
])

const selectedPermissions = ref([])

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  // Filter out any permissions not part of the `perms` object.
  const validPermissions = Object.values(perms)
  selectedPermissions.value = selectedPermissions.value.filter((perm) =>
    validPermissions.includes(perm)
  )
  values.permissions = selectedPermissions.value
  props.submitForm(values)
})

const handleChange = (value, perm) => {
  if (value) {
    selectedPermissions.value.push(perm)
  } else {
    const index = selectedPermissions.value.indexOf(perm)
    if (index > -1) {
      selectedPermissions.value.splice(index, 1)
    }
  }
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues, false)
    // Copy: mutating the shared array retriggers the watcher and resets the form.
    selectedPermissions.value = [...(newValues.permissions || [])]
  },
  { deep: true, immediate: true }
)
</script>
