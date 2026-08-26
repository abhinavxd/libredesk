<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField, handleChange }" name="enabled">
        <FormItem class="md:col-span-2">
          <SwitchField
            :title="t('admin.portal.enable')"
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField, handleChange }" name="tickets_from_article_only">
        <FormItem class="md:col-span-2">
          <SwitchField
            :title="t('admin.portal.articleOnlyTickets')"
            :description="t('admin.portal.articleOnlyTickets.description')"
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="inbox_id">
        <FormItem>
          <FormLabel>{{ t('globals.terms.inbox') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('admin.portal.inbox.placeholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                  <SelectItem v-for="inb in inboxes" :key="inb.id" :value="inb.id.toString()">
                    {{ inb.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ t('admin.portal.inbox.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="livechat_inbox_id">
        <FormItem>
          <FormLabel>{{ t('admin.portal.livechatInbox') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('admin.portal.inbox.placeholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                  <SelectItem v-for="inb in livechatInboxes" :key="inb.id" :value="inb.id.toString()">
                    {{ inb.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ t('admin.portal.livechatInbox.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="help_center_id">
        <FormItem>
          <FormLabel>{{ t('globals.terms.helpCenter') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="t('admin.portal.helpCenter.placeholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                  <SelectItem v-for="hc in helpCenters" :key="hc.id" :value="hc.id.toString()">
                    {{ hc.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ t('admin.portal.helpCenter.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="form_id">
        <FormItem>
          <FormLabel>{{ t('admin.portalForm.default') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                  <SelectItem v-for="f in ticketForms" :key="f.id" :value="f.id.toString()">
                    {{ f.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ t('admin.portalForm.default.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <Button type="submit" :isLoading="formLoading">{{ t('globals.messages.save') }}</Button>
  </form>
</template>

<script setup>
import { watch, ref } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import SwitchField from '@shared-ui/components/SwitchField.vue'

const emitter = useEmitter()
const { t } = useI18n()
const formLoading = ref(false)
const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  inboxes: {
    type: Array,
    default: () => []
  },
  livechatInboxes: {
    type: Array,
    default: () => []
  },
  helpCenters: {
    type: Array,
    default: () => []
  },
  ticketForms: {
    type: Array,
    default: () => []
  },
  submitForm: {
    type: Function,
    required: true
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema())
})

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await props.submitForm(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }
    form.setValues(
      {
        enabled: Boolean(newValues.enabled),
        tickets_from_article_only: Boolean(newValues.tickets_from_article_only),
        inbox_id: String(newValues.inbox_id || 0),
        help_center_id: String(newValues.help_center_id || 0),
        livechat_inbox_id: String(newValues.livechat_inbox_id || 0),
        form_id: String(newValues.form_id || 0)
      },
      false
    )
  },
  { deep: true, immediate: true }
)
</script>
