<template>
  <form @submit="onSubmit" novalidate class="team-form space-y-6">
    <FormField name="emoji" v-slot="{ componentField }">
      <FormItem ref="emojiPickerContainer" class="relative w-20">
        <FormLabel>{{ i18n._('Emoji') }}</FormLabel>
        <FormControl>
          <Button
            type="button"
            variant="outline"
            class="w-full font-normal"
            @click="toggleEmojiPicker"
          >
            {{ componentField.modelValue }}
          </Button>
          <div v-if="isEmojiPickerVisible" class="absolute z-10 mt-2">
            <EmojiPicker :native="true" @select="onSelectEmoji" class="w-[300px]" />
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <FormGrid class="team-settings-grid">
      <FormField v-slot="{ componentField }" name="name">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
          <FormControl>
            <Input type="text" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
      <FormField name="conversation_assignment_type" v-slot="{ componentField }">
        <FormItem>
          <FormLabel>{{ i18n._('Auto assignment type') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Select an assignment type')" />
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
          <FormDescription>{{ i18n._('Round robin: Conversations are assigned to team members in a round-robin fashion. Manual: Conversations are to be picked by team members.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="max_auto_assigned_conversations">
        <FormItem>
          <FormLabel>{{ i18n._('Maximum auto-assigned conversations') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="0" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._('Maximum number of conversations that can be auto-assigned to an agent. Only conversations in statuses with the \u0022Open\u0022 category count toward this limit. Set to 0 for unlimited.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
      <FormField v-slot="{ componentField }" name="timezone">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Timezone', 'Timezones', 1) }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Select timezone')" />
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
          <FormDescription>{{ i18n._('Team\'\'s timezone will be used to calculate SLA.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="business_hours_id">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('Business hour', 'Business hours', 2) }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Select business hours')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem :value="0">{{ i18n.pgettext('No business hours', 'None') }}</SelectItem>
                  <SelectItem v-for="bh in businessHours" :key="bh.id" :value="bh.id">
                    {{ bh.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormDescription>{{ i18n._('Default business hours for the team, will be used to calculate SLA.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="sla_policy_id">
        <FormItem>
          <FormLabel>{{ i18n.ngettext('SLA policy', 'SLA policies', 1) }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue :placeholder="i18n._('Select policy')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem :value="0">{{ i18n.pgettext('No SLA policy', 'None') }}</SelectItem>
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
          <FormDescription>{{ i18n._('SLA policy to be auto applied to conversations, when conversations are assigned to this team.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { ref, watch, onMounted, computed, defineAsyncComponent } from 'vue'
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
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useSlaStore } from '@main/stores/sla.js'
import { timeZones } from '@main/constants/timezones.js'
import api from '@main/api/index.js'
import { useI18n } from '@shared-ui/i18n'

const EmojiPicker = defineAsyncComponent(async () => {
  const [mod] = await Promise.all([import('vue3-emoji-picker'), import('vue3-emoji-picker/css')])
  return mod.default
})

const i18n = useI18n()
const emitter = useEmitter()
const slaStore = useSlaStore()
const assignmentTypes = computed(() => [
  { value: 'Round robin', label: i18n._('Round robin') },
  { value: 'Manual', label: i18n._('Manual') }
])
const businessHours = ref([])

const props = defineProps({
  initialValues: { type: Object, required: false },
  submitForm: { type: Function, required: true },
  submitLabel: { type: String, default: '' },
  isNewForm: { type: Boolean, default: false },
  isLoading: { type: Boolean }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.isNewForm ? i18n._('Create') : i18n._('Save'))
  )
})

const DEFAULT_TEAM_EMOJI = '👥'

const form = useForm({
  initialValues: { emoji: DEFAULT_TEAM_EMOJI },
  validationSchema: toTypedSchema(createTeamFormSchema(i18n))
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
            title: i18n._('Unauthorized'),
            variant: 'destructive',
            description: i18n._('You do not have permission to view business hours.')
          }
        : {
            title: i18n._('Could not fetch business hours'),
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
    sla_policy_id: values.sla_policy_id > 0 ? values.sla_policy_id : null
  })
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return
    form.setValues({ ...newValues, emoji: newValues.emoji || DEFAULT_TEAM_EMOJI }, false)
  },
  { immediate: true }
)

function toggleEmojiPicker() {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

function onSelectEmoji(emoji) {
  form.setFieldValue('emoji', emoji.i || emoji)
  isEmojiPickerVisible.value = false
}
</script>

<style scoped>
.team-form {
  container-type: inline-size;
}

.team-settings-grid.form-grid {
  grid-template-columns: minmax(0, 1fr);
}

@container (min-width: 32rem) {
  .team-settings-grid.form-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
