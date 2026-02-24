<template>
  <form @submit="onSubmit" class="space-y-6">
    <div class="grid gap-6 md:grid-cols-2">
    <FormField name="emoji" v-slot="{ componentField }">
      <FormItem ref="emojiPickerContainer" class="relative">
        <FormLabel>{{ $t('admin.team.emoji') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" @click="toggleEmojiPicker" />
          <div v-if="isEmojiPickerVisible" class="absolute z-10 mt-2">
            <EmojiPicker :native="true" @select="onSelectEmoji" class="w-[300px]" />
          </div>
        </FormControl>
        <FormDescription>{{ $t('admin.team.emoji.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name', 1) }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="$t('globals.terms.name', 1)" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ $t('admin.team.name.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="conversation_assignment_type" v-slot="{ componentField }">
      <FormItem>
        <FormLabel>{{ $t('admin.team.assignmentType') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="$t('admin.team.assignmentType.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="at in assignmentTypes" :key="at.value" :value="at.value">
                  {{ at.label }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ $t('admin.team.assignmentType.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="max_auto_assigned_conversations">
      <FormItem>
        <FormLabel>{{ $t('admin.team.maxAutoAssigned') }}</FormLabel>
        <FormControl>
          <Input type="number" placeholder="0" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ $t('admin.team.maxAutoAssigned.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="timezone">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.timezone', 1) }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="$t('admin.general.timezone.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="(value, label) in timeZones" :key="value" :value="value">
                  {{ label }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ $t('admin.team.timezone.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="business_hours_id">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.businessHour', 2) }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="$t('admin.general.businessHours.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem :value="0">{{ $t('globals.terms.none') }}</SelectItem>
                <SelectItem v-for="bh in businessHours" :key="bh.id" :value="bh.id">
                  {{ bh.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ $t('admin.team.businessHours.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="sla_policy_id">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.slaPolicy') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue :placeholder="$t('admin.team.slaPolicy.placeholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem :value="0">{{ $t('globals.terms.none') }}</SelectItem>
                <SelectItem
                  v-for="sla in slaStore.options"
                  :key="sla.value"
                  :value="parseInt(sla.value)"
                >
                  {{ sla.label }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ $t('admin.team.slaPolicy.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>
    </div>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createTeamFormSchema } from './teamFormSchema.js'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useSlaStore } from '../../../stores/sla.js'
import { timeZones } from '../../../constants/timezones.js'
import api from '../../../api/index.js'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const emitter = useEmitter()
const slaStore = useSlaStore()
const assignmentTypes = computed(() => [
  { value: 'Round robin', label: t('admin.team.assignmentType.roundRobin') },
  { value: 'Manual', label: t('admin.team.assignmentType.manual') }
])
const businessHours = ref([])

const props = defineProps({
  initialValues: { type: Object, required: false },
  submitForm: { type: Function, required: true },
  submitLabel: { type: String, default: '' },
  isLoading: { type: Boolean }
})

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.messages.submit')
})

const form = useForm({
  validationSchema: toTypedSchema(createTeamFormSchema(t))
})

const isEmojiPickerVisible = ref(false)
const emojiPickerContainer = ref(null)

onMounted(() => {
  fetchBusinessHours()
  onClickOutside(emojiPickerContainer, () => {
    isEmojiPickerVisible.value = false
  })
})

const fetchBusinessHours = async () => {
  try {
    const response = await api.getAllBusinessHours()
    businessHours.value = response.data.data
  } catch (error) {
    const toastPayload =
      error.response.status === 403
        ? {
            title: t('globals.terms.unAuthorized'),
            variant: 'destructive',
            description: t('admin.team.noPermissionBusinessHours')
          }
        : {
            title: t('admin.team.couldNotFetchBusinessHours'),
            variant: 'destructive',
            description: handleHTTPError(error).message
          }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, toastPayload)
  }
}

const onSubmit = form.handleSubmit((values) => {
  props.submitForm({
    ...values,
    business_hours_id: values.business_hours_id > 0 ? values.business_hours_id : null,
    sla_policy_id: values.sla_policy_id > 0 ? values.sla_policy_id: null
  })
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return
    form.setValues(newValues)
  },
  { immediate: true }
)

function toggleEmojiPicker() {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

function onSelectEmoji(emoji) {
  form.setFieldValue('emoji', emoji.i || emoji)
}
</script>
