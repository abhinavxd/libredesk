<template>
  <form @submit="onSubmit" class="space-y-6">
    <FormField name="emoji" v-slot="{ componentField }">
      <FormItem ref="emojiPickerContainer" class="relative">
        <FormLabel>Emoji</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" @click="toggleEmojiPicker" />
          <div v-if="isEmojiPickerVisible" class="absolute z-10 mt-2">
            <EmojiPicker :native="true" @select="onSelectEmoji" class="w-[300px]" />
          </div>
        </FormControl>
        <FormDescription>Display emoji for this team.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Name" v-bind="componentField" />
        </FormControl>
        <FormDescription>Select an unique name for the team.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="conversation_assignment_type" v-slot="{ componentField }">
      <FormItem>
        <FormLabel>Auto assignment type</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select a assignment type" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="at in assignmentTypes" :key="at" :value="at">
                  {{ at }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>
          Round robin: Conversations are assigned to team members in a round-robin fashion. <br />
          Manual: Conversations are to be picked by team members.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="max_auto_assigned_conversations">
      <FormItem>
        <FormLabel>Maximum auto-assigned conversations</FormLabel>
        <FormControl>
          <Input type="number" placeholder="0" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          Maximum number of conversations that can be auto-assigned to an agent, conversations in
          "Resolved" or "Closed" states do not count toward this limit. Set to 0 for unlimited.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="timezone">
      <FormItem>
        <FormLabel>Timezone</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select a timezone" />
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
        <FormDescription>Team's timezone will be used to calculate SLA.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="business_hours_id">
      <FormItem>
        <FormLabel>Business hours</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select business hours" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem :value = 0>None</SelectItem>
                <SelectItem v-for="bh in businessHours" :key="bh.id" :value="bh.id">
                  {{ bh.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription
          >Default business hours for the team, will be used to calculate SLA.</FormDescription
        >
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="sla_policy_id">
      <FormItem>
        <FormLabel>SLA policy</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select policy" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem :value= 0>None</SelectItem>
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
        <FormDescription
          >SLA policy to be auto applied to conversations, when conversations are assigned to this
          team.</FormDescription
        >
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { teamFormSchema } from './teamFormSchema.js'
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

const emitter = useEmitter()
const slaStore = useSlaStore()
const assignmentTypes = ['Round robin', 'Manual']
const businessHours = ref([])

const props = defineProps({
  initialValues: { type: Object, required: false },
  submitForm: { type: Function, required: true },
  submitLabel: { type: String, default: 'Submit' },
  isLoading: { type: Boolean }
})

const form = useForm({
  validationSchema: toTypedSchema(teamFormSchema)
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
    // If unauthorized (no permission), show a toast message.
    const toastPayload =
      error.response.status === 403
        ? {
            title: 'Unauthorized',
            variant: 'destructive',
            description: 'You do not have permission to view business hours.'
          }
        : {
            title: 'Could not fetch business hours',
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
