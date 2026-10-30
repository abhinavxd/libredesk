<template>
  <form @submit="onSubmit" class="space-y-8">
    <FormGrid>
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="description">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Description', 'Descriptions', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>
    <fieldset class="sla-durations min-w-0 space-y-3" :aria-describedby="durationHintId">
      <FormGrid class="sla-duration-grid">
        <FormField v-slot="{ componentField }" name="first_response_time">
          <FormItem>
            <FormLabel>{{ i18n._('First response time') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="6h" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="resolution_time">
          <FormItem>
            <FormLabel>{{ i18n._('Resolution time') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="24h" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="next_response_time">
          <FormItem>
            <FormLabel>{{ i18n._('Next response time') }}</FormLabel>
            <FormControl>
              <Input type="text" placeholder="30m" v-bind="componentField" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </FormGrid>
      <p :id="durationHintId" class="ui-description">
        {{ i18n._('Duration in hours or minutes. Example: 1h, 30m, 1h30m') }}
      </p>
    </fieldset>

    <!-- Notifications Section -->
    <div class="space-y-6">
      <div class="flex flex-wrap items-start justify-between gap-3 pb-3 border-b">
        <div class="space-y-1">
          <h3 class="ui-heading">
            {{ i18n._('Alert configuration') }}
          </h3>
          <p class="ui-description">
            {{ i18n._('Set up alert triggers and recipients') }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <Button type="button" variant="outline" size="sm" @click="addNotification('breach')">
            <Plus class="w-4 h-4" />
            {{ i18n._('Add breach alert') }}
          </Button>
          <Button type="button" variant="outline" size="sm" @click="addNotification('warning')">
            <Plus class="w-4 h-4" />
            {{ i18n._('Add warning alert') }}
          </Button>
        </div>
      </div>

      <div v-if="form.values.notifications?.length > 0" class="space-y-3">
        <Card
          v-for="(notification, index) in form.values.notifications"
          :key="index"
          class="sla-alert-card min-w-0 shadow-none"
        >
          <FormField :name="`notifications.${index}.type`" v-slot="{ componentField }">
            <Input v-bind="componentField" type="hidden" />
          </FormField>
          <CardHeader class="flex-row items-center justify-between gap-3 space-y-0 p-4 pb-4 sm:p-5 sm:pb-4">
            <div class="flex min-w-0 items-center gap-3">
              <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md"
                :class="
                  notification.type === 'warning'
                    ? 'bg-warning/10 text-warning-600'
                    : 'bg-destructive/10 text-destructive'
                "
              >
                <CircleAlert
                  v-if="notification.type === 'warning'"
                  class="h-4 w-4"
                  aria-hidden="true"
                />
                <Timer v-else class="h-4 w-4" aria-hidden="true" />
              </span>
              <div class="min-w-0 space-y-1">
                <h4 class="ui-subheading">
                  {{
                    notification.type === 'warning' ? i18n.pgettext('SLA warning threshold', 'Warning') : i18n._('Breach')
                  }}
                </h4>
              </div>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              :aria-label="i18n._('Remove')"
              @click.prevent="removeNotification(index)"
              class="h-8 w-8 shrink-0 text-muted-foreground"
            >
              <X class="h-4 w-4" aria-hidden="true" />
            </Button>
          </CardHeader>
          <CardContent class="p-4 pt-0 sm:p-5 sm:pt-0">
            <FormGrid class="sla-alert-grid items-start">
              <div class="min-w-0 space-y-5">
                <FormField :name="`notifications.${index}.metric`" v-slot="{ componentField }">
                  <FormItem>
                    <FormLabel>{{ i18n.ngettext('SLA metric', 'SLA metrics', 1) }}</FormLabel>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue :placeholder="i18n._('Select SLA metric')" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="all">{{ i18n._('All') }}</SelectItem>
                            <SelectItem value="first_response">{{
                              i18n._('First response time')
                            }}</SelectItem>
                            <SelectItem value="next_response">{{
                              i18n._('Next response time')
                            }}</SelectItem>
                            <SelectItem value="resolution">{{
                              i18n._('Resolution time')
                            }}</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField
                  :name="`notifications.${index}.recipients`"
                  v-slot="{ componentField, handleChange, meta }"
                >
                  <FormItem>
                    <FormLabel>{{ i18n._('Alert recipients') }}</FormLabel>
                    <FormControl>
                      <SelectAgentCombobox
                        multiple
                        exclude-ai-assistants
                        :prepend-items="[
                          { label: i18n._('Assigned user'), value: 'assigned_user' }
                        ]"
                        :placeholder="i18n._('Start typing to search...')"
                        :model-value="componentField.modelValue"
                        @update:model-value="(value) => handleChange(value, meta.validated)"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>
              <div class="min-w-0 space-y-5">
                <FormField
                  v-if="notification.type === 'breach'"
                  :name="`notifications.${index}.time_delay_type`"
                  v-slot="{ componentField }"
                >
                  <FormItem>
                    <FormLabel>{{ i18n._('Trigger timing') }}</FormLabel>
                    <FormControl>
                      <Select v-bind="componentField">
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="immediately">{{
                              i18n._('Immediately on breach')
                            }}</SelectItem>
                            <SelectItem value="after">{{
                              i18n._('After specific duration')
                            }}</SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
                <FormField
                  v-if="shouldShowTimeDelay(index)"
                  :name="`notifications.${index}.time_delay`"
                  v-slot="{ componentField, handleChange, meta }"
                >
                  <FormItem>
                    <FormLabel>
                      {{
                        notification.type === 'warning'
                          ? i18n._('Advance warning')
                          : i18n._('Follow up delay')
                      }}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="text"
                        placeholder="10m"
                        :name="componentField.name"
                        :model-value="componentField.modelValue"
                        @update:model-value="(value) => handleChange(value, meta.validated)"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>
            </FormGrid>
          </CardContent>
        </Card>
      </div>

      <!-- Empty State -->
      <div
        v-else
        class="flex flex-col items-center justify-center p-8 space-y-3 rounded-md bg-muted/30 border border-dashed"
      >
        <Bell class="w-8 h-8 text-muted-foreground" />
        <p class="ui-description">{{ i18n._('No alerts configured') }}</p>
      </div>
    </div>

    <Button type="submit" :disabled="isLoading" :isLoading="isLoading" class="mt-6">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { Card, CardHeader, CardContent } from '@shared-ui/components/ui/card'
import { watch, computed, useId } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema'
import { Button } from '@shared-ui/components/ui/button'
import { X, Plus, Timer, CircleAlert, Bell } from 'lucide-vue-next'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { useI18n } from '@shared-ui/i18n'
import SelectAgentCombobox from '@main/components/combobox/SelectAgentCombobox.vue'
import { Input } from '@shared-ui/components/ui/input'

const durationHintId = useId()

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.initialValues.id ? i18n._('Save') : i18n._('Create'))
  )
})

const i18n = useI18n()
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: {
    name: '',
    description: '',
    first_response_time: '',
    resolution_time: '',
    notifications: []
  }
})

const shouldShowTimeDelay = (index) => {
  const notification = form.values.notifications?.[index]
  if (!notification) return false
  return notification.type === 'warning' || notification.time_delay_type === 'after'
}

const addNotification = (type) => {
  const notifications = [...(form.values.notifications || [])]
  notifications.push({
    type: type,
    time_delay_type: type === 'warning' ? 'before' : 'immediately',
    time_delay: type === 'warning' ? '10m' : '',
    recipients: [],
    metric: 'all'
  })
  form.setFieldValue('notifications', notifications)
}

const removeNotification = (index) => {
  const notifications = [...form.values.notifications]
  notifications.splice(index, 1)
  form.setFieldValue('notifications', notifications)
}

watch(
  () => props.initialValues,
  (newValues) => {
    if (!newValues || Object.keys(newValues).length === 0) {
      form.resetForm()
      return
    }

    const transformedNotifications = (newValues.notifications || []).map((notification) => ({
      ...notification,
      // Default value, notification applies to all metrics unless specified.
      metric: notification.metric || 'all',
      time_delay_type:
        notification.type === 'warning'
          ? 'before'
          : notification.time_delay
            ? 'after'
            : 'immediately'
    }))

    form.setValues(
      {
        ...newValues,
        notifications: transformedNotifications
      },
      false
    )
  },
  { immediate: true, deep: true }
)

const onSubmit = form.handleSubmit((values) => {
  const payload = {
    ...values,
    notifications: values.notifications.map((notification) => ({
      ...notification,
      time_delay: notification.time_delay_type === 'immediately' ? '' : notification.time_delay
    }))
  }
  props.submitForm(payload)
})

// watch(
//   () => form.errors,
//   (errors) => {
//     if (Object.keys(errors).length > 0) {
//       console.log('Form has errors', errors)
//     }
//   },
//   { deep: true }
// )
</script>

<style scoped>
.sla-durations,
.sla-alert-card {
  container-type: inline-size;
}

.sla-duration-grid.form-grid,
.sla-alert-grid.form-grid {
  grid-template-columns: minmax(0, 1fr);
}

@container (min-width: 32rem) {
  .sla-duration-grid.form-grid,
  .sla-alert-grid.form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
