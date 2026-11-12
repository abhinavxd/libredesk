<template>
  <form novalidate class="webhook-form space-y-6 w-full">
    <FormField name="is_active" v-slot="{ value, handleChange }" v-if="!isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox
              :aria-label="i18n._('Active')"
              :checked="value"
              @update:checked="handleChange"
            />
            <Label>{{ i18n._('Active') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid class="webhook-settings-grid">
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="My Webhook" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="url">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('URL', 'URLs', 1) }}</FormLabel>
          <FormControl>
            <Input type="url" placeholder="https://your-app.com/webhook" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField name="events" v-slot="{ componentField, handleChange }">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Event', 'Events', 2) }}</FormLabel>
        <FormDescription>
          {{ i18n._('Select the events you want to subscribe to.') }}
        </FormDescription>
        <FormControl>
          <div class="space-y-6">
            <div
              v-for="group in webhookEvents"
              :key="group.name"
              class="rounded-md border border-border bg-card"
            >
              <div class="border-b border-border bg-muted/30 px-5 py-3">
                <h4 class="ui-subheading">{{ group.name }}</h4>
              </div>
              <div class="p-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div
                  v-for="event in group.events"
                  :key="event.value"
                  class="flex items-start space-x-3"
                >
                  <Checkbox
                    :id="`${eventId}-${event.value}`"
                    :checked="componentField.modelValue?.includes(event.value)"
                    @update:checked="
                      (checked) =>
                        handleEventChange(
                          checked,
                          event.value,
                          handleChange,
                          componentField.modelValue
                        )
                    "
                  />
                  <Label
                    :for="`${eventId}-${event.value}`"
                    class="font-normal text-sm cursor-pointer"
                    >{{ event.label }}</Label
                  >
                </div>
              </div>
            </div>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid class="webhook-settings-grid">
      <FormField v-slot="{ componentField }" name="secret">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Secret', 'Secrets', 1) }}</FormLabel>
          <FormControl>
            <Input type="password" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._('Optional secret key for webhook signature verification.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <!-- Form submit button slot -->
    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import { useId } from 'vue'
import { Label } from '@shared-ui/components/ui/label'
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { ref } from 'vue'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useI18n } from '@shared-ui/i18n'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'

defineProps({
  form: {
    type: Object,
    required: true
  },
  isNewForm: {
    type: Boolean
  }
})

const i18n = useI18n()
const eventId = useId()

const webhookEvents = ref([
  {
    name: i18n._('{count, plural, one {Conversation} other {Conversations}}'),
    events: [
      {
        value: 'conversation.created',
        label: 'Conversation created'
      },
      {
        value: 'conversation.status_changed',
        label: 'Conversation status changed'
      },
      {
        value: 'conversation.tags_changed',
        label: 'Conversation tags changed'
      },
      {
        value: 'conversation.assigned',
        label: 'Conversation assigned'
      },
      {
        value: 'conversation.unassigned',
        label: 'Conversation unassigned'
      }
    ]
  },
  {
    name: i18n._('{count, plural, one {Message} other {Messages}}'),
    events: [
      {
        value: 'message.created',
        label: 'Message created'
      },
      {
        value: 'message.updated',
        label: 'Message updated'
      }
    ]
  }
])

// If checked add event to the list, if unchecked remove it and call handleChange
const handleEventChange = (checked, eventName, handleChange, currentEvents) => {
  const events = currentEvents || []
  handleChange(checked ? [...events, eventName] : events.filter((e) => e !== eventName))
}
</script>

<style scoped>
.webhook-form {
  container-type: inline-size;
}

.webhook-settings-grid.form-grid {
  grid-template-columns: minmax(0, 1fr);
}

@container (min-width: 32rem) {
  .webhook-settings-grid.form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
