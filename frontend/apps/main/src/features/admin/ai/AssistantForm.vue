<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="i18n._('Enabled')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ i18n._('The display name customers see when they chat with this assistant.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="space-y-2">
      <div class="text-sm font-medium leading-none">{{ i18n._('Avatar') }}</div>
      <AvatarUpload
        :src="avatarPreview"
        :initials="avatarInitials"
        :label="i18n._('Upload')"
        @upload="onAvatarUpload"
        @remove="onAvatarRemove"
      />
    </div>

    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Description', 'Descriptions', 1) }}</FormLabel>
        <FormControl>
          <Textarea
            rows="3"
            :placeholder="i18n._('Handles order status and returns questions')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('For your own reference only; customers never see this.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="expectation">
      <FormItem>
        <FormLabel>{{ i18n._('Expectation message') }}</FormLabel>
        <FormControl>
          <Textarea
            rows="2"
            :placeholder="i18n._('I\'\'m an AI assistant. A human agent can join if you need one.')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('Shown to the customer while this assistant handles the chat. It disappears on handoff to a human.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormGrid compact>
      <FormField v-slot="{ componentField }" name="tone">
        <FormItem>
          <FormLabel>{{ i18n._('Tone') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="(label, tone) in tones" :key="tone" :value="tone">
                    {{ label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="response_length">
        <FormItem>
          <FormLabel>{{ i18n._('Response length') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="(label, length) in responseLengths" :key="length" :value="length">
                    {{ label }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="max_turns">
        <FormItem>
          <FormLabel>{{ i18n._('Max turns') }}</FormLabel>
          <FormControl>
            <Input type="number" min="1" max="20" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ i18n._('Number of back-and-forth replies before the assistant hands off. Between 1 and 20.') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </FormGrid>

    <FormField v-slot="{ componentField, handleChange }" name="languages">
      <FormItem>
        <FormLabel>{{ i18n.ngettext('Language', 'Languages', 2) }}</FormLabel>
        <FormControl>
          <TagsInput :modelValue="componentField.modelValue" @update:modelValue="handleChange">
            <TagsInputItem v-for="item in componentField.modelValue" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput :placeholder="i18n._('Type a language and press Enter')" />
          </TagsInput>
        </FormControl>
        <FormDescription>{{ i18n._('Languages the assistant is allowed to reply in. It uses the customer\'\'s language when it is one of these; otherwise it replies in the first one. Leave empty to always match the customer\'\'s language.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange, meta }" name="citations_enabled">
      <FormItem>
        <SwitchField
          :title="i18n._('Include article references')"
          :description="i18n._('Link to public help center articles used in chat and email replies. Private knowledge snippets are never listed.')"
          :checked="componentField.modelValue"
          @update:checked="(value) => handleChange(value, meta.validated)"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="handoff_enabled">
      <FormItem>
        <SwitchField
          :title="i18n._('Offer handoff to a human')"
          :description="i18n._('When off, the assistant never offers to connect the customer with a human agent and tells them it cannot help with questions outside its knowledge.')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="fallback_team_id">
      <FormItem>
        <FormLabel>{{ i18n._('Fallback team') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="none">{{ i18n.pgettext('No assigned team', 'None') }}</SelectItem>
                <SelectItem v-for="team in teams" :key="team.id" :value="String(team.id)">
                  {{ team.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ i18n._('Team the conversation is handed off to when the assistant cannot help.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="instructions">
      <FormItem>
        <FormLabel>{{ i18n._('Instructions') }}</FormLabel>
        <FormControl>
          <Textarea
            rows="6"
            :placeholder="i18n._('Always confirm the order ID before looking anything up. Keep replies under 3 sentences.')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('Custom system prompt that guides how this assistant responds.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="guardrails">
      <FormItem>
        <FormLabel>{{ i18n._('Guardrails') }}</FormLabel>
        <FormControl>
          <Textarea
            rows="6"
            :placeholder="i18n._('Never issue a refund without a human agent\'\'s approval. Never share other customers\'\' order details.')"
            v-bind="componentField"
          />
        </FormControl>
        <FormDescription>{{ i18n._('Rules the assistant must never break.') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="space-y-2">
      <div class="text-sm font-medium leading-none">{{ i18n._('Tools') }}</div>
      <p class="ui-description">{{ i18n._('Tools this assistant is allowed to call. It can only use the ones you grant here. Knowledge snippets are shared across all assistants automatically.') }}</p>
      <div class="space-y-3 mt-3">
        <div v-for="tool in tools" :key="tool.id" class="flex items-center space-x-2">
          <Checkbox
            :id="`tool-${tool.id}`"
            :checked="selectedToolIds.includes(tool.id)"
            @update:checked="(checked) => toggleTool(tool.id, checked)"
          />
          <label :for="`tool-${tool.id}`" class="text-sm font-medium leading-none cursor-pointer">
            {{ tool.name }}
          </label>
        </div>
        <p v-if="!tools.length" class="ui-description">
          {{ i18n._('No tools yet. Create tools under AI to grant them here.') }}
        </p>
      </div>
    </div>

    <div class="flex justify-end mt-10">
      <Button type="submit" :isLoading="formLoading">
        {{ isEditing ? i18n._('Save') : i18n._('Create') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { computed, ref, watch, onMounted } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@shared-ui/components/ui/tags-input'
import { AvatarUpload } from '@shared-ui/components/ui/avatar'
import SwitchField from '@shared-ui/components/SwitchField.vue'
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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form/index.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import api from '@/api'

const tones = computed(() => ({
  friendly: i18n._('Friendly'),
  professional: i18n._('Professional'),
  neutral: i18n._('Neutral'),
  casual: i18n._('Casual')
}))
const responseLengths = computed(() => ({
  concise: i18n._('Concise'),
  balanced: i18n._('Balanced'),
  detailed: i18n._('Detailed')
}))

const props = defineProps({
  initialValues: { type: Object, default: () => ({}) },
  isEditing: { type: Boolean, default: false },
  submitForm: { type: Function, required: true }
})

const i18n = useI18n()
const emitter = useEmitter()
const formLoading = ref(false)
const tools = ref([])
const teams = ref([])
const selectedToolIds = ref([])
const avatarFile = ref(null)
const avatarPreview = ref('')
const removeAvatar = ref(false)

const avatarInitials = computed(
  () => (form.values.name || '').trim().charAt(0).toUpperCase() || 'A'
)

const onAvatarUpload = (file) => {
  if (!file) return
  avatarFile.value = file
  avatarPreview.value = URL.createObjectURL(file)
  removeAvatar.value = false
}

const onAvatarRemove = () => {
  avatarFile.value = null
  avatarPreview.value = ''
  removeAvatar.value = true
}

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z
        .string({ required_error: i18n._('{name} Required') })
        .min(1, { message: i18n._('{name} Required') }),
      description: z.string().optional(),
      expectation: z.string().optional(),
      tone: z.string().optional(),
      response_length: z.string().optional(),
      max_turns: z.coerce
        .number()
        .int()
        .min(1, { message: i18n._('Number of back-and-forth replies before the assistant hands off. Between 1 and 20.') })
        .max(20, { message: i18n._('Number of back-and-forth replies before the assistant hands off. Between 1 and 20.') }),
      fallback_team_id: z.string().optional(),
      handoff_enabled: z.boolean().optional(),
      citations_enabled: z.boolean().optional(),
      languages: z.array(z.string()).optional(),
      instructions: z.string().optional(),
      guardrails: z.string().optional(),
      enabled: z.boolean().optional()
    })
  ),
  initialValues: {
    name: '',
    description: '',
    expectation: '',
    tone: 'friendly',
    response_length: 'balanced',
    max_turns: 6,
    fallback_team_id: 'none',
    handoff_enabled: true,
    citations_enabled: true,
    languages: [],
    instructions: '',
    guardrails: '',
    enabled: true
  }
})

const toggleTool = (id, checked) => {
  if (checked) {
    if (!selectedToolIds.value.includes(id)) selectedToolIds.value.push(id)
  } else {
    selectedToolIds.value = selectedToolIds.value.filter((t) => t !== id)
  }
}

watch(
  () => props.initialValues,
  (values) => {
    form.setValues(
      {
        name: values.name || '',
        description: values.description || '',
        expectation: values.expectation || '',
        tone: values.tone || 'friendly',
        response_length: values.response_length || 'balanced',
        max_turns: values.max_turns ?? 6,
        fallback_team_id: values.fallback_team_id ? String(values.fallback_team_id) : 'none',
        handoff_enabled: values.handoff_enabled ?? true,
        citations_enabled: values.citations_enabled ?? !props.isEditing,
        languages: [...(values.languages || [])],
        instructions: values.instructions || '',
        guardrails: values.guardrails || '',
        enabled: values.enabled ?? true
      },
      false
    )
    selectedToolIds.value = [...(values.tool_ids || [])]
    avatarFile.value = null
    avatarPreview.value = values.avatar_url || ''
    removeAvatar.value = false
    form.setErrors({})
  },
  { immediate: true, deep: true }
)

onMounted(async () => {
  try {
    const [toolsResp, teamsResp] = await Promise.all([api.getAITools(), api.getTeamsCompact()])
    tools.value = (toolsResp.data.data || []).filter((tool) => tool.enabled)
    teams.value = teamsResp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
})

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    const payload = {
      name: values.name,
      description: values.description || '',
      expectation: values.expectation || '',
      tone: values.tone || 'friendly',
      response_length: values.response_length || 'balanced',
      max_turns: Number(values.max_turns),
      fallback_team_id:
        values.fallback_team_id && values.fallback_team_id !== 'none'
          ? Number(values.fallback_team_id)
          : null,
      handoff_enabled: !!values.handoff_enabled,
      citations_enabled: !!values.citations_enabled,
      languages: values.languages || [],
      instructions: values.instructions || '',
      guardrails: values.guardrails || '',
      enabled: !!values.enabled,
      remove_avatar: removeAvatar.value,
      tool_ids: [...selectedToolIds.value]
    }
    const formData = new FormData()
    formData.append('data', JSON.stringify(payload))
    if (avatarFile.value) {
      formData.append('files', avatarFile.value)
    }
    await props.submitForm(formData)
  } finally {
    formLoading.value = false
  }
})
</script>
