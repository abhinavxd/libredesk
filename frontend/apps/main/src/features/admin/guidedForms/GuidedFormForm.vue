<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="t('globals.terms.enabled')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="inbox_id">
      <FormItem>
        <FormLabel>{{ t('admin.guidedForms.inbox') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem
                  v-for="inbox in livechatInboxes"
                  :key="inbox.value"
                  :value="inbox.value"
                >
                  {{ inbox.label }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>{{ t('admin.guidedForms.inboxHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-sm font-medium leading-none">{{ t('admin.guidedForms.steps') }}</div>
          <p class="text-sm text-muted-foreground">{{ t('admin.guidedForms.stepsHint') }}</p>
        </div>
        <Button type="button" variant="outline" size="sm" @click="addStep">
          <Plus class="mr-1 h-4 w-4" />
          {{ t('admin.guidedForms.addStep') }}
        </Button>
      </div>

      <div v-for="(step, index) in steps" :key="step._key" class="border rounded-lg p-4 space-y-4">
        <div class="flex items-center justify-between">
          <span class="text-sm font-medium">{{ t('admin.guidedForms.step') }} {{ index + 1 }}</span>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            class="h-7 w-7"
            :disabled="steps.length <= 1"
            @click="removeStep(index)"
          >
            <X class="h-4 w-4" />
          </Button>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div class="space-y-2">
            <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.stepId') }}</label>
            <Input v-model="step.id" :placeholder="t('admin.guidedForms.stepIdPlaceholder')" />
            <p class="text-xs text-muted-foreground">{{ t('admin.guidedForms.stepIdHint') }}</p>
          </div>
          <div class="space-y-2">
            <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.stepType') }}</label>
            <Select v-model="step.type">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="type in stepTypes" :key="type" :value="type">
                    {{ t(`admin.guidedForms.stepTypeOption.${type}`) }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="space-y-2">
          <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.question') }}</label>
          <Textarea v-model="step.question" rows="2" />
        </div>

        <div v-if="step.type === 'choice'" class="space-y-2">
          <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.options') }}</label>
          <TagsInput :modelValue="step.options" @update:modelValue="(v) => (step.options = v)">
            <TagsInputItem v-for="item in step.options" :key="item" :value="item">
              <TagsInputItemText />
              <TagsInputItemDelete />
            </TagsInputItem>
            <TagsInputInput :placeholder="t('admin.guidedForms.optionsPlaceholder')" />
          </TagsInput>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div class="space-y-2">
            <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.saveAsAttribute') }}</label>
            <Select
              :modelValue="step.custom_attribute_id ? String(step.custom_attribute_id) : 'none'"
              @update:modelValue="(v) => (step.custom_attribute_id = v === 'none' ? 0 : Number(v))"
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">{{ t('globals.terms.none') }}</SelectItem>
                  <SelectItem
                    v-for="attr in customAttributes"
                    :key="attr.id"
                    :value="String(attr.id)"
                  >
                    {{ attr.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <p class="text-xs text-muted-foreground">{{ t('admin.guidedForms.saveAsAttributeHint') }}</p>
          </div>
          <div class="flex items-center space-x-2 pt-6">
            <Checkbox
              :id="`required-${step._key}`"
              :checked="step.required"
              @update:checked="(v) => (step.required = v)"
            />
            <label :for="`required-${step._key}`" class="text-sm font-medium leading-none cursor-pointer">
              {{ t('admin.guidedForms.requiredStep') }}
            </label>
          </div>
        </div>

        <div class="space-y-2 border-t pt-3">
          <div class="flex items-center justify-between">
            <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.branches') }}</label>
            <Button type="button" variant="outline" size="sm" @click="addBranch(step)">
              <Plus class="mr-1 h-3 w-3" />
              {{ t('admin.guidedForms.addBranch') }}
            </Button>
          </div>
          <p class="text-xs text-muted-foreground">{{ t('admin.guidedForms.branchesHint') }}</p>
          <div
            v-for="(branch, bIndex) in step.branches"
            :key="bIndex"
            class="flex items-center gap-2"
          >
            <Input
              v-model="branch.pattern"
              class="flex-1"
              :placeholder="t('admin.guidedForms.branchPatternPlaceholder')"
            />
            <span class="text-xs text-muted-foreground shrink-0">{{ t('admin.guidedForms.goesTo') }}</span>
            <Select v-model="branch.next_step_id">
              <SelectTrigger class="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="s in otherSteps(step)" :key="s.id" :value="s.id">
                    {{ s.id || t('admin.guidedForms.untitledStep') }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="h-8 w-8 shrink-0"
              @click="step.branches.splice(bIndex, 1)"
            >
              <X class="h-4 w-4" />
            </Button>
          </div>

          <div class="flex items-center gap-2 pt-1">
            <span class="text-xs text-muted-foreground shrink-0">{{ t('admin.guidedForms.otherwiseGoesTo') }}</span>
            <Select
              :modelValue="step.default_next_step_id || 'none'"
              @update:modelValue="(v) => (step.default_next_step_id = v === 'none' ? '' : v)"
            >
              <SelectTrigger class="w-56">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">{{ t('admin.guidedForms.endFlow') }}</SelectItem>
                  <SelectItem v-for="s in otherSteps(step)" :key="s.id" :value="s.id">
                    {{ s.id || t('admin.guidedForms.untitledStep') }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>
    </div>

    <div class="space-y-4 border-t pt-4">
      <div class="text-sm font-medium leading-none">{{ t('admin.guidedForms.onComplete') }}</div>

      <FormField v-slot="{ componentField }" name="on_complete_action">
        <FormItem>
          <FormLabel>{{ t('admin.guidedForms.onCompleteAction') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="team">{{ t('admin.guidedForms.onCompleteActionOption.team') }}</SelectItem>
                  <SelectItem value="ai_assistant">{{ t('admin.guidedForms.onCompleteActionOption.ai_assistant') }}</SelectItem>
                  <SelectItem value="unassigned">{{ t('admin.guidedForms.onCompleteActionOption.unassigned') }}</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.on_complete_action === 'team'"
        v-slot="{ componentField }"
        name="on_complete_team_id"
      >
        <FormItem>
          <FormLabel>{{ t('admin.guidedForms.onCompleteTeam') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem v-for="team in teams" :key="team.id" :value="String(team.id)">
                    {{ team.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField
        v-if="form.values.on_complete_action === 'ai_assistant'"
        v-slot="{ componentField }"
        name="on_complete_assistant_id"
      >
        <FormItem>
          <FormLabel>{{ t('admin.guidedForms.onCompleteAssistant') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem
                    v-for="assistant in aiAssistantStore.assistants"
                    :key="assistant.id"
                    :value="String(assistant.id)"
                  >
                    {{ assistant.name }}
                  </SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="completion_message">
        <FormItem>
          <FormLabel>{{ t('admin.guidedForms.completionMessage') }}</FormLabel>
          <FormControl>
            <Textarea rows="2" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ t('admin.guidedForms.completionMessageHint') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="flex justify-end mt-10">
      <Button type="submit" :isLoading="formLoading">
        {{ isEditing ? t('globals.messages.save') : t('globals.messages.create') }}
      </Button>
    </div>
  </form>
</template>

<script setup>
import { computed, ref, watch, onMounted } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'
import { Plus, X } from 'lucide-vue-next'
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
import { useInboxStore } from '@/stores/inbox'
import { useAIAssistantStore } from '@/stores/aiAssistant'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const stepTypes = ['text', 'choice', 'email', 'phone', 'number']

const props = defineProps({
  initialValues: { type: Object, default: () => ({}) },
  isEditing: { type: Boolean, default: false },
  submitForm: { type: Function, required: true }
})

const { t } = useI18n()
const emitter = useEmitter()
const inboxStore = useInboxStore()
const aiAssistantStore = useAIAssistantStore()
const formLoading = ref(false)
const teams = ref([])
const customAttributes = ref([])
const steps = ref([])
let stepKeySeq = 0

const livechatInboxes = computed(() => inboxStore.livechatOptions)

const newStep = () => ({
  _key: ++stepKeySeq,
  id: `step_${steps.value.length + 1}`,
  question: '',
  type: 'text',
  options: [],
  custom_attribute_id: 0,
  required: false,
  branches: [],
  default_next_step_id: ''
})

const addStep = () => {
  steps.value.push(newStep())
}

const removeStep = (index) => {
  steps.value.splice(index, 1)
}

const addBranch = (step) => {
  step.branches.push({ pattern: '', next_step_id: '' })
}

const otherSteps = (current) => steps.value.filter((s) => s !== current)

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z
        .string({ required_error: t('globals.messages.required') })
        .min(1, { message: t('globals.messages.required') }),
      inbox_id: z.string({ required_error: t('globals.messages.required') }).min(1, {
        message: t('globals.messages.required')
      }),
      on_complete_action: z.string().default('team'),
      on_complete_team_id: z.string().optional(),
      on_complete_assistant_id: z.string().optional(),
      completion_message: z.string().optional(),
      enabled: z.boolean().optional()
    })
  ),
  initialValues: {
    name: '',
    inbox_id: '',
    on_complete_action: 'team',
    on_complete_team_id: '',
    on_complete_assistant_id: '',
    completion_message: '',
    enabled: true
  }
})

watch(
  () => props.initialValues,
  (values) => {
    form.setValues(
      {
        name: values.name || '',
        inbox_id: values.inbox_id ? String(values.inbox_id) : '',
        on_complete_action: values.on_complete_action || 'team',
        on_complete_team_id: values.on_complete_team_id ? String(values.on_complete_team_id) : '',
        on_complete_assistant_id: values.on_complete_assistant_id
          ? String(values.on_complete_assistant_id)
          : '',
        completion_message: values.completion_message || '',
        enabled: values.enabled ?? true
      },
      false
    )
    steps.value = (values.steps || []).map((s) => ({
      _key: ++stepKeySeq,
      id: s.id || '',
      question: s.question || '',
      type: s.type || 'text',
      options: [...(s.options || [])],
      custom_attribute_id: s.custom_attribute_id || 0,
      required: !!s.required,
      branches: (s.branches || []).map((b) => ({ pattern: b.pattern || '', next_step_id: b.next_step_id || '' })),
      default_next_step_id: s.default_next_step_id || ''
    }))
    if (steps.value.length === 0) {
      steps.value.push(newStep())
    }
    form.setErrors({})
  },
  { immediate: true, deep: true }
)

onMounted(async () => {
  try {
    const [teamsResp] = await Promise.all([
      api.getTeamsCompact(),
      inboxStore.fetchInboxes(),
      aiAssistantStore.loadAssistants(),
      fetchCustomAttributes()
    ])
    teams.value = teamsResp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
})

const fetchCustomAttributes = async () => {
  const [contactAttrs, conversationAttrs] = await Promise.all([
    api.getCustomAttributes('contact'),
    api.getCustomAttributes('conversation')
  ])
  customAttributes.value = [
    ...(contactAttrs.data?.data || []),
    ...(conversationAttrs.data?.data || [])
  ]
}

const onSubmit = form.handleSubmit(async (values) => {
  if (steps.value.some((s) => !s.id.trim() || !s.question.trim())) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('admin.guidedForms.stepsIncompleteError')
    })
    return
  }
  try {
    formLoading.value = true
    const payload = {
      name: values.name,
      inbox_id: Number(values.inbox_id),
      enabled: !!values.enabled,
      start_step_id: steps.value[0]?.id || '',
      steps: steps.value.map((s) => ({
        id: s.id.trim(),
        question: s.question,
        type: s.type,
        options: s.type === 'choice' ? s.options : [],
        save_as: s.id.trim(),
        custom_attribute_id: s.custom_attribute_id || 0,
        required: !!s.required,
        branches: s.branches.filter((b) => b.pattern.trim() && b.next_step_id),
        default_next_step_id: s.default_next_step_id || ''
      })),
      on_complete_action: values.on_complete_action,
      on_complete_team_id:
        values.on_complete_action === 'team' && values.on_complete_team_id
          ? Number(values.on_complete_team_id)
          : null,
      on_complete_assistant_id:
        values.on_complete_action === 'ai_assistant' && values.on_complete_assistant_id
          ? Number(values.on_complete_assistant_id)
          : null,
      completion_message: values.completion_message || ''
    }
    await props.submitForm(payload)
  } finally {
    formLoading.value = false
  }
})
</script>
