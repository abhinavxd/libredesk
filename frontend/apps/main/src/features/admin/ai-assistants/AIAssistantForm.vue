<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <!-- Enabled Field -->
    <FormField v-slot="{ componentField, handleChange }" name="enabled" v-if="!isNewForm">
      <FormItem class="flex flex-row items-center justify-between rounded-lg border p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ t('globals.terms.enabled') }}</FormLabel>
          <FormDescription>{{ t('ai.assistant.enabledDescription') }}</FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- Name Field -->
    <FormField v-slot="{ componentField }" name="first_name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }} <span class="text-red-500">*</span></FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="t('ai.assistant.namePlaceholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ t('ai.assistant.nameDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Avatar url -->
    <FormField v-slot="{ componentField }" name="avatar_url">
      <FormItem>
        <FormLabel>{{ t('globals.terms.avatar') }} {{ t('globals.terms.url') }}</FormLabel>
        <FormControl>
          <Input
            type="url"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage></FormMessage>
      </FormItem>
    </FormField>

    <!-- Product Name Field -->
    <FormField v-slot="{ componentField }" name="product_name">
      <FormItem>
        <FormLabel
          >{{ t('ai.assistant.productName') }} <span class="text-red-500">*</span></FormLabel
        >
        <FormControl>
          <Input
            type="text"
            :placeholder="t('ai.assistant.productNamePlaceholder')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ t('ai.assistant.productNameDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Product Description Field -->
    <FormField v-slot="{ componentField }" name="product_description">
      <FormItem>
        <FormLabel
          >{{ t('ai.assistant.productDescription') }} <span class="text-red-500">*</span></FormLabel
        >
        <FormControl>
          <Textarea
            :placeholder="t('ai.assistant.productDescriptionPlaceholder')"
            v-bind="componentField"
            rows="4"
          />
        </FormControl>
        <FormDescription>{{ t('ai.assistant.productDescriptionDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Answer Length Field -->
    <FormField v-slot="{ componentField }" name="answer_length">
      <FormItem>
        <FormLabel
          >{{ t('ai.assistant.answerLength') }} <span class="text-red-500">*</span></FormLabel
        >
        <Select v-bind="componentField">
          <FormControl>
            <SelectTrigger>
              <SelectValue :placeholder="t('ai.assistant.selectAnswerLength')" />
            </SelectTrigger>
          </FormControl>
          <SelectContent>
            <SelectItem value="concise">{{ t('ai.assistant.answerLengthConcise') }}</SelectItem>
            <SelectItem value="medium">{{ t('ai.assistant.answerLengthMedium') }}</SelectItem>
            <SelectItem value="long">{{ t('ai.assistant.answerLengthLong') }}</SelectItem>
          </SelectContent>
        </Select>
        <FormDescription>{{ t('ai.assistant.answerLengthDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Answer Tone Field -->
    <FormField v-slot="{ componentField }" name="answer_tone">
      <FormItem>
        <FormLabel
          >{{ t('ai.assistant.answerTone') }} <span class="text-red-500">*</span></FormLabel
        >
        <Select v-bind="componentField">
          <FormControl>
            <SelectTrigger>
              <SelectValue :placeholder="t('ai.assistant.selectAnswerTone')" />
            </SelectTrigger>
          </FormControl>
          <SelectContent>
            <SelectItem value="neutral">{{ t('ai.assistant.answerToneNeutral') }}</SelectItem>
            <SelectItem value="friendly">{{ t('ai.assistant.answerToneFriendly') }}</SelectItem>
            <SelectItem value="professional">{{
              t('ai.assistant.answerToneProfessional')
            }}</SelectItem>
            <SelectItem value="humorous">{{ t('ai.assistant.answerToneHumorous') }}</SelectItem>
          </SelectContent>
        </Select>
        <FormDescription>{{ t('ai.assistant.answerToneDescription') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Enable Handoff Checkbox -->
    <FormField v-slot="{ componentField, handleChange }" name="hand_off">
      <FormItem class="flex flex-row items-center justify-between rounded-lg border p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ t('ai.assistant.enableHandoff') }}</FormLabel>
          <FormDescription>{{ t('ai.assistant.enableHandoffDescription') }}</FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- Hand off team (conditional) -->
    <FormField v-slot="{ componentField }" name="hand_off_team" v-if="form.values.hand_off">
      <FormItem>
        <FormLabel>{{ t('ai.assistant.conversationHandoffTeam') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <FormControl>
              <SelectTrigger>
                <SelectValue
                  :placeholder="
                    t('globals.messages.select', { name: t('globals.terms.team').toLowerCase() })
                  "
                />
              </SelectTrigger>
            </FormControl>
            <SelectContent>
              <SelectItem
                v-for="opt in teamStore.options"
                :key="opt.value"
                :value="parseInt(opt.value)"
              >
                {{ opt.label }}
              </SelectItem>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Submit Button -->
    <div class="flex justify-end">
      <Button type="submit" :disabled="formLoading">
        <template v-if="formLoading">
          <LoaderCircle class="w-4 h-4 mr-2 animate-spin" />
        </template>
        {{ isNewForm ? t('globals.messages.create') : t('globals.messages.update') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Switch } from '@shared-ui/components/ui/switch'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { LoaderCircle } from 'lucide-vue-next'
import { createFormSchema } from './formSchema.js'
import { useTeamStore } from '@/stores/team'

const { t } = useI18n()
const teamStore = useTeamStore()
const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  isNewForm: {
    type: Boolean,
    default: false
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const formLoading = computed(() => props.isLoading)

const formSchema = toTypedSchema(createFormSchema(t))

const form = useForm({
  validationSchema: formSchema,
  initialValues: {
    first_name: '',
    last_name: '',
    avatar_url: '',
    product_name: '',
    product_description: '',
    answer_length: 'medium',
    answer_tone: 'friendly',
    hand_off: false,
    hand_off_team: null,
    enabled: true,
    ...props.initialValues
  }
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

// Parse meta fields if editing an existing assistant
onMounted(() => {
  if (!props.isNewForm && props.initialValues?.meta) {
    try {
      const meta =
        typeof props.initialValues.meta === 'string'
          ? JSON.parse(props.initialValues.meta)
          : props.initialValues.meta

      if (meta) {
        form.setFieldValue('product_name', meta.product_name || '')
        form.setFieldValue('product_description', meta.product_description || '')
        form.setFieldValue('answer_length', meta.answer_length || 'medium')
        form.setFieldValue('answer_tone', meta.answer_tone || 'friendly')
        form.setFieldValue('hand_off', meta.hand_off || false)
        form.setFieldValue('hand_off_team', meta.hand_off_team || null)
      }
    } catch (e) {
      console.warn('Failed to parse AI assistant meta:', e)
    }
  }
})

// Watch for changes in initialValues (for edit mode)
watch(
  () => props.initialValues,
  (newValues) => {
    if (newValues && Object.keys(newValues).length > 0) {
      form.resetForm({
        values: {
          first_name: newValues.first_name || '',
          last_name: newValues.last_name || '',
          avatar_url: newValues.avatar_url || '',
          hand_off: newValues.hand_off ?? false,
          hand_off_team: newValues.hand_off_team || null,
          enabled: newValues.enabled ?? true,
          ...newValues
        }
      })
    }
  },
  { deep: true, immediate: true }
)
</script>
