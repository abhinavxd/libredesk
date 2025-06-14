<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="t('globals.terms.agent')" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.description') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="t('admin.role.roleForAllSupportAgents')"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div>
      <div class="mb-5 text-lg">{{ $t('admin.role.setPermissionsForThisRole') }}</div>

      <div class="space-y-6">
        <div
          v-for="entity in permissions"
          :key="entity.name"
          class="rounded border border-border bg-card"
        >
          <div class="border-b border-border bg-muted/30 px-5 py-3">
            <h4 class="font-medium text-card-foreground">{{ entity.name }}</h4>
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
import { watch, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { Checkbox } from '@/components/ui/checkbox'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useI18n } from 'vue-i18n'
import { permissions as perms } from '@/constants/permissions.js'

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
  isLoading: {
    type: Boolean,
    required: false
  }
})

const { t } = useI18n()

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.messages.save')
})

const permissions = ref([
  {
    name: t('globals.terms.conversation'),
    permissions: [
      { name: perms.CONVERSATIONS_READ, label: t('admin.role.conversations.read') },
      { name: perms.CONVERSATIONS_WRITE, label: t('admin.role.conversations.write') },
      {
        name: perms.CONVERSATIONS_READ_ASSIGNED,
        label: t('admin.role.conversations.readAssigned')
      },
      { name: perms.CONVERSATIONS_READ_ALL, label: t('admin.role.conversations.readAll') },
      {
        name: perms.CONVERSATIONS_READ_UNASSIGNED,
        label: t('admin.role.conversations.readUnassigned')
      },
      {
        name: perms.CONVERSATIONS_READ_TEAM_INBOX,
        label: t('admin.role.conversations.readTeamInbox')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_USER_ASSIGNEE,
        label: t('admin.role.conversations.updateUserAssignee')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_TEAM_ASSIGNEE,
        label: t('admin.role.conversations.updateTeamAssignee')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_PRIORITY,
        label: t('admin.role.conversations.updatePriority')
      },
      {
        name: perms.CONVERSATIONS_UPDATE_STATUS,
        label: t('admin.role.conversations.updateStatus')
      },
      { name: perms.CONVERSATIONS_UPDATE_TAGS, label: t('admin.role.conversations.updateTags') },
      { name: perms.MESSAGES_READ, label: t('admin.role.messages.read') },
      { name: perms.MESSAGES_WRITE, label: t('admin.role.messages.write') },
      { name: perms.VIEW_MANAGE, label: t('admin.role.view.manage') }
    ]
  },
  {
    name: t('globals.terms.admin'),
    permissions: [
      { name: perms.GENERAL_SETTINGS_MANAGE, label: t('admin.role.generalSettings.manage') },
      {
        name: perms.NOTIFICATION_SETTINGS_MANAGE,
        label: t('admin.role.notificationSettings.manage')
      },
      { name: perms.STATUS_MANAGE, label: t('admin.role.status.manage') },
      { name: perms.OIDC_MANAGE, label: t('admin.role.oidc.manage') },
      { name: perms.TAGS_MANAGE, label: t('admin.role.tags.manage') },
      { name: perms.MACROS_MANAGE, label: t('admin.role.macros.manage') },
      { name: perms.USERS_MANAGE, label: t('admin.role.users.manage') },
      { name: perms.TEAMS_MANAGE, label: t('admin.role.teams.manage') },
      { name: perms.AUTOMATIONS_MANAGE, label: t('admin.role.automations.manage') },
      { name: perms.INBOXES_MANAGE, label: t('admin.role.inboxes.manage') },
      { name: perms.ROLES_MANAGE, label: t('admin.role.roles.manage') },
      { name: perms.TEMPLATES_MANAGE, label: t('admin.role.templates.manage') },
      { name: perms.REPORTS_MANAGE, label: t('admin.role.reports.manage') },
      { name: perms.BUSINESS_HOURS_MANAGE, label: t('admin.role.businessHours.manage') },
      { name: perms.SLA_MANAGE, label: t('admin.role.sla.manage') },
      { name: perms.AI_MANAGE, label: t('admin.role.ai.manage') },
      { name: perms.CUSTOM_ATTRIBUTES_MANAGE, label: t('admin.role.customAttributes.manage') },
      { name: perms.ACTIVITY_LOGS_MANAGE, label: t('admin.role.activityLog.manage') },
      { name: perms.WEBHOOKS_MANAGE, label: t('admin.role.webhooks.manage') }
    ]
  },
  {
    name: t('globals.terms.contact'),
    permissions: [
      { name: perms.CONTACTS_READ_ALL, label: t('admin.role.contacts.readAll') },
      { name: perms.CONTACTS_READ, label: t('admin.role.contacts.read') },
      { name: perms.CONTACTS_WRITE, label: t('admin.role.contacts.write') },
      { name: perms.CONTACTS_BLOCK, label: t('admin.role.contacts.block') },
      { name: perms.CONTACT_NOTES_READ, label: t('admin.role.contactNotes.read') },
      { name: perms.CONTACT_NOTES_WRITE, label: t('admin.role.contactNotes.write') },
      { name: perms.CONTACT_NOTES_DELETE, label: t('admin.role.contactNotes.delete') }
    ]
  }
])

const selectedPermissions = ref([])

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
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
    form.setValues(newValues)
    selectedPermissions.value = newValues.permissions || []
  },
  { deep: true, immediate: true }
)
</script>
