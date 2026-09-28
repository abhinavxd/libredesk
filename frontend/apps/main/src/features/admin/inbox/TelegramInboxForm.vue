<template>
  <form novalidate class="space-y-6 w-full" @submit="onSubmit">
    <div
      v-if="initialValues.webhook_error"
      role="alert"
      class="box border-destructive/40 bg-destructive/5 p-3 text-sm flex items-start gap-2"
    >
      <TriangleAlert aria-hidden="true" class="size-4 shrink-0 text-destructive" />
      <span>{{ initialValues.webhook_error }}</span>
    </div>
    <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.telegram.description') }}</p>
    <div class="box space-y-4 p-4">
      <h3 class="font-semibold">{{ $t('globals.terms.general') }}</h3>
      <div class="grid gap-4 md:grid-cols-2">
        <FormField
          v-slot="{ componentField, handleChange, meta }"
          name="name"
          :validate-on-blur="false"
          :validate-on-change="false"
          :validate-on-input="false"
          :validate-on-model-update="false"
        >
          <FormItem>
            <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
            <FormControl>
              <Input
                v-bind="componentField"
                @update:model-value="(v) => handleChange(v, meta.validated)"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField
          v-slot="{ componentField, handleChange, meta }"
          name="reopen_window_hours"
          :validate-on-blur="false"
          :validate-on-change="false"
          :validate-on-input="false"
          :validate-on-model-update="false"
        >
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.reopenWindow') }}</FormLabel>
            <FormControl>
              <Input
                v-bind="componentField"
                type="number"
                min="0"
                step="1"
                @update:model-value="(v) => handleChange(v, meta.validated)"
              />
            </FormControl>
            <FormDescription>{{ $t('admin.inbox.reopenWindow.description') }}</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
      <FormField
        v-for="field in switches"
        :key="field.name"
        v-slot="{ componentField, handleChange, meta }"
        :name="field.name"
      >
        <FormItem>
          <div class="flex items-center justify-between gap-4">
            <div>
              <FormLabel>{{ $t(field.label) }}</FormLabel>
              <FormDescription v-if="field.description">{{
                $t(field.description)
              }}</FormDescription>
            </div>
            <FormControl>
              <Switch
                type="button"
                :checked="componentField.modelValue"
                @update:checked="(v) => handleChange(v, meta.validated)"
              />
            </FormControl>
          </div>
        </FormItem>
      </FormField>
    </div>
    <div class="box space-y-4 p-4">
      <h3 class="font-semibold">{{ $t('globals.terms.telegram') }}</h3>
      <FormField
        v-slot="{ componentField, handleChange, meta }"
        name="config.bot_token"
        :validate-on-blur="false"
        :validate-on-change="false"
        :validate-on-input="false"
        :validate-on-model-update="false"
      >
        <FormItem>
          <FormLabel>{{ $t('globals.terms.botToken') }}</FormLabel>
          <FormControl>
            <Input
              v-bind="componentField"
              type="password"
              autocomplete="new-password"
              @update:model-value="(v) => handleChange(v, meta.validated)"
            />
          </FormControl>
          <FormDescription>
            <i18n-t keypath="admin.inbox.telegram.botToken.description">
              <template #botFather>
                <a
                  href="https://t.me/BotFather"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-link underline underline-offset-4"
                  >BotFather</a
                >
              </template>
            </i18n-t>
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
      <div v-if="initialValues.webhook_url" class="space-y-2">
        <Label for="telegram-webhook-url">{{ $t('globals.terms.callbackURL') }}</Label>
        <div class="flex items-center gap-2">
          <Input
            class="min-w-0 flex-1 font-mono text-xs"
            id="telegram-webhook-url"
            :model-value="initialValues.webhook_url"
            readonly
          />
          <CopyButton
            class="shrink-0"
            :text="initialValues.webhook_url"
            :aria-label="$t('globals.terms.copy')"
          />
        </div>
      </div>
      <p class="text-sm text-muted-foreground">
        {{ $t('admin.inbox.telegram.webhook.description') }}
      </p>
    </div>
    <div class="box space-y-4 p-4">
      <h3 class="font-semibold">{{ $t('globals.messages.automaticReplies') }}</h3>
      <FormField
        v-for="field in messageFields"
        :key="field.name"
        v-slot="{ componentField, handleChange, meta }"
        :name="field.name"
        :validate-on-blur="false"
        :validate-on-change="false"
        :validate-on-input="false"
        :validate-on-model-update="false"
      >
        <FormItem>
          <FormLabel>{{ $t(field.label) }}</FormLabel>
          <FormControl>
            <Textarea
              v-bind="componentField"
              :rows="3"
              @update:model-value="(value) => handleChange(value, meta.validated)"
            />
          </FormControl>
          <FormDescription>{{ $t(field.description) }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
      <div v-show="form.values.config?.away_message?.trim()" class="grid gap-4 md:grid-cols-2">
        <FormField
          v-slot="{ componentField, handleChange, meta }"
          name="config.business_hours_id"
          :validate-on-blur="false"
          :validate-on-change="false"
          :validate-on-input="false"
          :validate-on-model-update="false"
        >
          <FormItem>
            <FormLabel>{{ $t('globals.terms.businessHour', 2) }}</FormLabel>
            <FormControl>
              <Select
                v-bind="componentField"
                @update:model-value="(value) => handleChange(value, meta.validated)"
              >
                <SelectTrigger
                  ><SelectValue :placeholder="$t('admin.general.businessHours.placeholder')"
                /></SelectTrigger>
                <SelectContent>
                  <SelectItem :value="0">{{ $t('globals.terms.none') }}</SelectItem>
                  <SelectItem v-for="hours in businessHours" :key="hours.id" :value="hours.id">{{
                    hours.name
                  }}</SelectItem>
                </SelectContent>
              </Select>
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField
          v-slot="{ componentField, handleChange, meta }"
          name="config.timezone"
          :validate-on-blur="false"
          :validate-on-change="false"
          :validate-on-input="false"
          :validate-on-model-update="false"
        >
          <FormItem>
            <FormLabel>{{ $t('globals.terms.timezone') }}</FormLabel>
            <FormControl>
              <Select
                v-bind="componentField"
                @update:model-value="(value) => handleChange(value, meta.validated)"
              >
                <SelectTrigger
                  ><SelectValue :placeholder="$t('admin.general.timezone.placeholder')"
                /></SelectTrigger>
                <SelectContent
                  ><SelectItem v-for="(value, label) in timeZones" :key="value" :value="value">{{
                    label
                  }}</SelectItem></SelectContent
                >
              </Select>
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>
    <div class="box space-y-4 p-4">
      <FormField v-slot="{ componentField, handleChange, meta }" name="csat_enabled">
        <FormItem>
          <div class="flex items-center justify-between gap-4">
            <div>
              <FormLabel>{{ $t('admin.inbox.csatSurveys') }}</FormLabel>
              <FormDescription>{{ $t('admin.inbox.csatSurveys.description_1') }}</FormDescription>
            </div>
            <FormControl>
              <Switch
                type="button"
                :checked="componentField.modelValue"
                @update:checked="(v) => handleChange(v, meta.validated)"
              />
            </FormControl>
          </div>
        </FormItem>
      </FormField>
      <div v-show="form.values.csat_enabled" class="border-t border-border pt-4">
        <FormField
          v-slot="{ componentField, handleChange, meta }"
          name="config.csat_message"
          :validate-on-blur="false"
          :validate-on-change="false"
          :validate-on-input="false"
          :validate-on-model-update="false"
        >
          <FormItem>
            <FormLabel>{{ $t('globals.messages.surveyMessage') }}</FormLabel>
            <FormControl>
              <Textarea
                v-bind="componentField"
                :rows="3"
                @update:model-value="(value) => handleChange(value, meta.validated)"
              />
            </FormControl>
            <FormDescription>{{
              $t('admin.inbox.telegram.csatMessage.description')
            }}</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>
    <Button type="submit" class="max-md:min-h-11" :is-loading="isLoading" :disabled="isLoading">
      {{ isNewForm ? $t('globals.messages.create') : $t('globals.messages.save') }}
    </Button>
  </form>
</template>

<script setup>
import { computed, watch, ref, onMounted } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { useI18n } from 'vue-i18n'
import { TriangleAlert } from 'lucide-vue-next'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormDescription,
  FormMessage
} from '@shared-ui/components/ui/form'
import api from '@main/api'
import { timeZones } from '@main/constants/timezones'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Button } from '@shared-ui/components/ui/button'
import { Switch } from '@shared-ui/components/ui/switch'
import CopyButton from '@main/components/button/CopyButton.vue'
import { createFormSchema } from './telegramFormSchema'

const props = defineProps({
  initialValues: { type: Object, default: () => ({}) },
  submitForm: { type: Function, required: true },
  isNewForm: { type: Boolean, default: false },
  isLoading: { type: Boolean, default: false }
})
const { t } = useI18n()
const businessHours = ref([])
const emitter = useEmitter()
onMounted(async () => {
  try {
    const response = await api.getAllBusinessHours()
    businessHours.value = response.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
})
const messageFields = [
  {
    name: 'config.greeting_message',
    label: 'globals.messages.greetingMessage',
    description: 'admin.inbox.telegram.greeting.description'
  },
  {
    name: 'config.away_message',
    label: 'globals.messages.awayMessage',
    description: 'admin.inbox.telegram.away.description'
  }
]
const switches = [
  {
    name: 'enabled',
    label: 'globals.terms.enabled'
  },
  {
    name: 'prompt_tags_on_reply',
    label: 'admin.inbox.promptTagsOnReply',
    description: 'admin.inbox.promptTagsOnReply.description'
  }
]
const form = useForm({
  validationSchema: computed(() => toTypedSchema(createFormSchema(t))),
  initialValues: {
    name: '',
    enabled: true,
    csat_enabled: false,
    prompt_tags_on_reply: false,
    reopen_window_hours: 48,
    config: {
      bot_token: '',
      greeting_message: '',
      away_message: '',
      csat_message: '',
      business_hours_id: 0,
      timezone: 'UTC'
    }
  }
})
const onSubmit = form.handleSubmit((values) => props.submitForm(values))
watch(
  () => props.initialValues,
  (values) => {
    if (Object.keys(values).length)
      form.setValues({ ...values, reopen_window_hours: values.reopen_window_hours ?? 0 }, false)
  },
  { immediate: true, deep: true }
)
</script>
