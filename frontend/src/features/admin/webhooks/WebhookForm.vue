<template>
  <form class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="My Webhook" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="url">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.url') }}</FormLabel>
        <FormControl>
          <Input type="url" placeholder="https://your-app.com/webhook" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="events" v-slot="{ componentField, handleChange }">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.event', 2) }}</FormLabel>
        <FormDescription>
          {{ $t('admin.webhook.events.description') }}
        </FormDescription>
        <FormControl>
          <div class="space-y-6">
            <div
              v-for="eventGroup in webhookEvents"
              :key="eventGroup.name"
              class="rounded border border-border bg-card"
            >
              <div class="border-b border-border bg-muted/30 px-5 py-3">
                <h4 class="font-medium text-card-foreground">{{ eventGroup.name }}</h4>
              </div>

              <div class="p-5">
                <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  <div
                    v-for="event in eventGroup.events"
                    :key="event.value"
                    class="flex items-start space-x-3"
                  >
                    <Checkbox
                      :checked="componentField.modelValue?.includes(event.value)"
                      @update:checked="
                        (newValue) =>
                          handleEventChange(
                            newValue,
                            event.value,
                            handleChange,
                            componentField.modelValue
                          )
                      "
                    />
                    <label class="font-normal text-sm">{{ event.label }}</label>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="secret">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.secret') }}</FormLabel>
        <FormControl>
          <Input type="password" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ $t('admin.webhook.secret.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="is_active" v-slot="{ value, handleChange }" v-if="!isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>{{ $t('globals.terms.active') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Form submit button slot -->
    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import { ref } from 'vue'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { useI18n } from 'vue-i18n'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

defineProps({
  form: {
    type: Object,
    required: true
  },
  isNewForm: {
    type: Boolean
  }
})

const { t } = useI18n()

const webhookEvents = ref([
  {
    name: t('globals.terms.conversation'),
    events: [
      {
        value: 'conversation.created',
        label: 'Conversation Created'
      },
      {
        value: 'conversation.status_changed',
        label: 'Conversation Status Changed'
      },
      {
        value: 'conversation.tags_changed',
        label: 'Conversation Tags Changed'
      },
      {
        value: 'conversation.assigned',
        label: 'Conversation Assigned'
      },
      {
        value: 'conversation.unassigned',
        label: 'Conversation Unassigned'
      }
    ]
  },
  {
    name: t('globals.terms.message'),
    events: [
      {
        value: 'message.created',
        label: 'Message Created'
      },
      {
        value: 'message.updated',
        label: 'Message Updated'
      }
    ]
  }
])

const handleEventChange = (checked, eventName, handleChange, currentEvents) => {
  const events = currentEvents || []
  let newEvents

  if (checked) {
    newEvents = [...events, eventName]
  } else {
    newEvents = events.filter((event) => event !== eventName)
  }

  handleChange(newEvents)
}
</script>
