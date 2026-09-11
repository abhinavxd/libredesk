<template>
  <form @submit="onSubmit" novalidate class="space-y-6 w-full">
    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem>
        <SwitchField
          :title="t('globals.terms.enabled')"
          :description="t('admin.guidedForms.enabledHint')"
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
        <FormDescription>{{ t('admin.guidedForms.nameHint') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="display_name">
      <FormItem>
        <FormLabel>{{ t('admin.guidedForms.displayName') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="form.values.name" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ t('admin.guidedForms.displayNameHint') }}</FormDescription>
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

      <Draggable v-model="steps" item-key="_key" :animation="200" handle=".step-drag-handle" class="space-y-3">
        <template #item="{ element: step, index }">
      <div class="border rounded-lg p-4 space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <div class="step-drag-handle cursor-move text-muted-foreground">
              <GripVertical class="w-4 h-4" />
            </div>
            <span class="text-sm font-medium">{{ t('admin.guidedForms.step') }} {{ index + 1 }}</span>
          </div>
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
          <label class="text-sm font-medium leading-none">
            {{ step.type === 'info' ? t('admin.guidedForms.statementText') : t('admin.guidedForms.question') }}
          </label>
          <Textarea v-model="step.question" rows="2" />
          <p v-if="step.type === 'info'" class="text-xs text-muted-foreground">
            {{ t('admin.guidedForms.statementHint') }}
          </p>
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
          <p class="text-xs text-muted-foreground">{{ t('admin.guidedForms.optionsHint') }}</p>
        </div>

        <div v-if="step.type !== 'info'" class="grid gap-4 md:grid-cols-2">
          <div class="space-y-2">
            <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.saveAsAttribute') }}</label>
            <Select
              :modelValue="saveTargetValue(step)"
              @update:modelValue="(v) => setSaveTargetValue(step, v)"
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">{{ t('globals.terms.none') }}</SelectItem>
                  <SelectItem value="contact_name">{{ t('admin.guidedForms.saveTargetName') }}</SelectItem>
                  <SelectItem value="contact_email">{{ t('admin.guidedForms.saveTargetEmail') }}</SelectItem>
                  <SelectItem
                    v-for="attr in customAttributes"
                    :key="attr.id"
                    :value="`attr_${attr.id}`"
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

        <!-- Branching, nested visually under the question it belongs to. Not shown for a
             statement step, which has no answer to branch on. -->
        <div class="border-t pt-3">
          <div v-if="step.type !== 'info'" class="flex items-center justify-between">
            <label class="text-sm font-medium leading-none">{{ t('admin.guidedForms.branches') }}</label>
            <Button type="button" variant="outline" size="sm" @click="addBranch(step)">
              <Plus class="mr-1 h-3 w-3" />
              {{ t('admin.guidedForms.addBranch') }}
            </Button>
          </div>
          <p v-if="step.type !== 'info'" class="text-xs text-muted-foreground mt-1">
            {{ t('admin.guidedForms.branchesHint') }}
          </p>

          <div class="mt-3 ml-2 pl-4 border-l-2 border-muted space-y-3">
            <template v-if="step.type !== 'info'">
              <!-- Quick-add a branch per choice option, prefilled to match that option exactly. -->
              <div v-if="step.type === 'choice' && step.options.length" class="flex flex-wrap items-center gap-2">
                <span class="text-xs text-muted-foreground">{{ t('admin.guidedForms.quickAddBranch') }}</span>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="h-6 px-2 text-xs"
                  v-for="option in step.options"
                  :key="option"
                  @click="addBranchFromOption(step, option)"
                >
                  <Plus class="mr-1 h-3 w-3" />
                  {{ option }}
                </Button>
              </div>

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
                <Select
                  :modelValue="targetValue(branch)"
                  @update:modelValue="(v) => setTargetValue(branch, v)"
                >
                  <SelectTrigger class="w-48">
                    <SelectValue :placeholder="t('admin.guidedForms.pickStep')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem v-for="s in otherSteps(step)" :key="s.id" :value="s.id">
                        {{ s.id || t('admin.guidedForms.untitledStep') }}
                      </SelectItem>
                    </SelectGroup>
                    <SelectGroup v-if="teams.length">
                      <SelectItem v-for="team in teams" :key="`team_${team.id}`" :value="`team_${team.id}`">
                        {{ t('admin.guidedForms.teamPrefix') }} {{ team.name }}
                      </SelectItem>
                    </SelectGroup>
                    <SelectGroup v-if="aiAssistantStore.assistants.length">
                      <SelectItem
                        v-for="assistant in aiAssistantStore.assistants"
                        :key="`assistant_${assistant.id}`"
                        :value="`assistant_${assistant.id}`"
                      >
                        {{ t('admin.guidedForms.assistantPrefix') }} {{ assistant.name }}
                      </SelectItem>
                    </SelectGroup>
                    <SelectGroup v-if="otherFormOptions.length">
                      <SelectItem
                        v-for="f in otherFormOptions"
                        :key="`form_${f.id}`"
                        :value="`form_${f.id}`"
                      >
                        {{ t('admin.guidedForms.formPrefix') }} {{ f.name }}
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
              <p v-if="!step.branches.length" class="text-xs text-muted-foreground italic">
                {{ t('admin.guidedForms.noBranches') }}
              </p>
            </template>

            <div class="flex items-center gap-2 pt-1">
              <span class="text-xs text-muted-foreground shrink-0">
                {{ step.type === 'info' ? t('admin.guidedForms.thenGoesTo') : t('admin.guidedForms.otherwiseGoesTo') }}
              </span>
              <Select
                :modelValue="otherwiseValue(step)"
                @update:modelValue="(v) => setOtherwiseValue(step, v)"
              >
                <SelectTrigger class="w-64">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="auto">{{ t('admin.guidedForms.continueToNext') }}</SelectItem>
                    <SelectItem value="end">{{ t('admin.guidedForms.endFlow') }}</SelectItem>
                    <SelectItem v-for="s in otherSteps(step)" :key="s.id" :value="s.id">
                      {{ t('admin.guidedForms.jumpTo') }} {{ s.id || t('admin.guidedForms.untitledStep') }}
                    </SelectItem>
                  </SelectGroup>
                  <SelectGroup v-if="teams.length">
                    <SelectItem v-for="team in teams" :key="`team_${team.id}`" :value="`team_${team.id}`">
                      {{ t('admin.guidedForms.teamPrefix') }} {{ team.name }}
                    </SelectItem>
                  </SelectGroup>
                  <SelectGroup v-if="aiAssistantStore.assistants.length">
                    <SelectItem
                      v-for="assistant in aiAssistantStore.assistants"
                      :key="`assistant_${assistant.id}`"
                      :value="`assistant_${assistant.id}`"
                    >
                      {{ t('admin.guidedForms.assistantPrefix') }} {{ assistant.name }}
                    </SelectItem>
                  </SelectGroup>
                  <SelectGroup v-if="otherFormOptions.length">
                    <SelectItem
                      v-for="f in otherFormOptions"
                      :key="`form_${f.id}`"
                      :value="`form_${f.id}`"
                    >
                      {{ t('admin.guidedForms.formPrefix') }} {{ f.name }}
                    </SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      </div>
        </template>
      </Draggable>

      <GuidedFormTester :steps="steps" />
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

      <FormField v-slot="{ componentField, handleChange }" name="allow_skip_to_human">
        <FormItem>
          <SwitchField
            :title="t('admin.guidedForms.allowSkipToHuman')"
            :description="t('admin.guidedForms.allowSkipToHumanHint')"
            :checked="componentField.modelValue"
            @update:checked="handleChange"
          />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="abandoned_timeout_minutes">
        <FormItem>
          <FormLabel>{{ t('admin.guidedForms.abandonedTimeout') }}</FormLabel>
          <FormControl>
            <Input type="number" min="0" step="1" class="max-w-[160px]" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ t('admin.guidedForms.abandonedTimeoutHint') }}</FormDescription>
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
import { Plus, X, GripVertical } from 'lucide-vue-next'
import Draggable from 'vuedraggable'
import GuidedFormTester from './GuidedFormTester.vue'
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

const stepTypes = ['text', 'choice', 'email', 'phone', 'number', 'info']

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
const allForms = ref([])
const steps = ref([])
let stepKeySeq = 0

const livechatInboxes = computed(() => inboxStore.livechatOptions)

// Every other guided form, selectable as a branch/otherwise "jump to another form" target.
const otherFormOptions = computed(() =>
  allForms.value.filter((f) => f.id !== Number(props.initialValues?.id))
)

const emptyTarget = () => ({
  next_step_id: '',
  action: '',
  team_id: 0,
  assistant_id: 0,
  form_id: 0
})

const newStep = () => ({
  _key: ++stepKeySeq,
  id: `step_${steps.value.length + 1}`,
  question: '',
  type: 'text',
  options: [],
  custom_attribute_id: 0,
  contact_field: '',
  required: false,
  branches: [],
  default_next_step_id: '',
  ends_form: false,
  default_action: '',
  default_team_id: 0,
  default_assistant_id: 0,
  default_form_id: 0
})

const addStep = () => {
  steps.value.push(newStep())
}

const removeStep = (index) => {
  steps.value.splice(index, 1)
}

const addBranch = (step) => {
  step.branches.push({ pattern: '', ...emptyTarget() })
}

// targetValue/setTargetValue encode a branch's routing target (continue to a step within this
// form, or route away to a team/AI assistant/different guided form) as a single select value.
const targetValue = (branch) => {
  if (branch.action === 'team') return `team_${branch.team_id}`
  if (branch.action === 'ai_assistant') return `assistant_${branch.assistant_id}`
  if (branch.action === 'form') return `form_${branch.form_id}`
  return branch.next_step_id || ''
}

const setTargetValue = (branch, value) => {
  Object.assign(branch, emptyTarget())
  if (value.startsWith('team_')) {
    branch.action = 'team'
    branch.team_id = Number(value.slice(5))
  } else if (value.startsWith('assistant_')) {
    branch.action = 'ai_assistant'
    branch.assistant_id = Number(value.slice(10))
  } else if (value.startsWith('form_')) {
    branch.action = 'form'
    branch.form_id = Number(value.slice(5))
  } else {
    branch.next_step_id = value
  }
}

// escapeRegex lets an admin type/click a plain option label without knowing regex syntax -
// branch patterns are matched as case-insensitive regex on the backend.
const escapeRegex = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

// addBranchFromOption adds a branch pre-filled to match a choice option exactly, so wiring up
// branching for a choice step is a couple of clicks instead of hand-writing regex.
const addBranchFromOption = (step, option) => {
  if (step.branches.some((b) => b.pattern === `^${escapeRegex(option)}$`)) return
  step.branches.push({ pattern: `^${escapeRegex(option)}$`, ...emptyTarget() })
}

const otherSteps = (current) => steps.value.filter((s) => s !== current)

// otherwiseValue/setOtherwiseValue collapse ends_form + default_next_step_id/default_action into
// a single select: continue automatically (the default), end the form here, jump to a specific
// step, or route away to a team/AI assistant/different guided form.
const otherwiseValue = (step) => {
  if (step.ends_form) return 'end'
  if (step.default_action === 'team') return `team_${step.default_team_id}`
  if (step.default_action === 'ai_assistant') return `assistant_${step.default_assistant_id}`
  if (step.default_action === 'form') return `form_${step.default_form_id}`
  return step.default_next_step_id || 'auto'
}

const setOtherwiseValue = (step, value) => {
  step.ends_form = false
  step.default_next_step_id = ''
  step.default_action = ''
  step.default_team_id = 0
  step.default_assistant_id = 0
  step.default_form_id = 0
  if (value === 'auto') {
    return
  } else if (value === 'end') {
    step.ends_form = true
  } else if (value.startsWith('team_')) {
    step.default_action = 'team'
    step.default_team_id = Number(value.slice(5))
  } else if (value.startsWith('assistant_')) {
    step.default_action = 'ai_assistant'
    step.default_assistant_id = Number(value.slice(10))
  } else if (value.startsWith('form_')) {
    step.default_action = 'form'
    step.default_form_id = Number(value.slice(5))
  } else {
    step.default_next_step_id = value
  }
}

// saveTargetValue/setSaveTargetValue collapse contact_field + custom_attribute_id into the
// single "save answer to" select: none, a core contact field, or a custom attribute.
const saveTargetValue = (step) => {
  if (step.contact_field === 'name') return 'contact_name'
  if (step.contact_field === 'email') return 'contact_email'
  if (step.custom_attribute_id) return `attr_${step.custom_attribute_id}`
  return 'none'
}

const setSaveTargetValue = (step, value) => {
  step.contact_field = ''
  step.custom_attribute_id = 0
  if (value === 'contact_name') {
    step.contact_field = 'name'
  } else if (value === 'contact_email') {
    step.contact_field = 'email'
  } else if (value.startsWith('attr_')) {
    step.custom_attribute_id = Number(value.slice(5))
  }
}

const form = useForm({
  validationSchema: toTypedSchema(
    z.object({
      name: z
        .string({ required_error: t('globals.messages.required') })
        .min(1, { message: t('globals.messages.required') }),
      display_name: z.string().optional(),
      inbox_id: z.string({ required_error: t('globals.messages.required') }).min(1, {
        message: t('globals.messages.required')
      }),
      on_complete_action: z.string().default('team'),
      on_complete_team_id: z.string().optional(),
      on_complete_assistant_id: z.string().optional(),
      completion_message: z.string().optional(),
      enabled: z.boolean().optional(),
      allow_skip_to_human: z.boolean().optional(),
      abandoned_timeout_minutes: z.coerce.number().int().min(0).optional()
    })
  ),
  initialValues: {
    name: '',
    display_name: '',
    inbox_id: '',
    on_complete_action: 'team',
    on_complete_team_id: '',
    on_complete_assistant_id: '',
    completion_message: '',
    enabled: true,
    allow_skip_to_human: true,
    abandoned_timeout_minutes: 0
  }
})

watch(
  () => props.initialValues,
  (values) => {
    form.setValues(
      {
        name: values.name || '',
        display_name: values.display_name || '',
        inbox_id: values.inbox_id ? String(values.inbox_id) : '',
        on_complete_action: values.on_complete_action || 'team',
        on_complete_team_id: values.on_complete_team_id ? String(values.on_complete_team_id) : '',
        on_complete_assistant_id: values.on_complete_assistant_id
          ? String(values.on_complete_assistant_id)
          : '',
        completion_message: values.completion_message || '',
        enabled: values.enabled ?? true,
        allow_skip_to_human: values.allow_skip_to_human ?? true,
        abandoned_timeout_minutes: values.abandoned_timeout_minutes ?? 0
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
      contact_field: s.contact_field || '',
      required: !!s.required,
      branches: (s.branches || []).map((b) => ({
        pattern: b.pattern || '',
        next_step_id: b.next_step_id || '',
        action: b.action || '',
        team_id: b.team_id || 0,
        assistant_id: b.assistant_id || 0,
        form_id: b.form_id || 0
      })),
      default_next_step_id: s.default_next_step_id || '',
      ends_form: !!s.ends_form,
      default_action: s.default_action || '',
      default_team_id: s.default_team_id || 0,
      default_assistant_id: s.default_assistant_id || 0,
      default_form_id: s.default_form_id || 0
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
    const [teamsResp, formsResp] = await Promise.all([
      api.getTeamsCompact(),
      api.getGuidedForms(),
      inboxStore.fetchInboxes(),
      aiAssistantStore.loadAssistants(),
      fetchCustomAttributes()
    ])
    teams.value = teamsResp.data.data || []
    allForms.value = formsResp.data.data || []
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
      display_name: values.display_name || '',
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
        contact_field: s.contact_field || '',
        required: !!s.required,
        branches: s.branches
          .filter((b) => b.pattern.trim() && (b.action ? true : b.next_step_id))
          .map((b) => ({
            pattern: b.pattern,
            next_step_id: b.action ? '' : b.next_step_id,
            action: b.action || '',
            team_id: b.action === 'team' ? Number(b.team_id) || 0 : 0,
            assistant_id: b.action === 'ai_assistant' ? Number(b.assistant_id) || 0 : 0,
            form_id: b.action === 'form' ? Number(b.form_id) || 0 : 0
          })),
        default_next_step_id: s.default_action ? '' : s.default_next_step_id || '',
        ends_form: !!s.ends_form,
        default_action: s.default_action || '',
        default_team_id: s.default_action === 'team' ? Number(s.default_team_id) || 0 : 0,
        default_assistant_id:
          s.default_action === 'ai_assistant' ? Number(s.default_assistant_id) || 0 : 0,
        default_form_id: s.default_action === 'form' ? Number(s.default_form_id) || 0 : 0
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
      completion_message: values.completion_message || '',
      allow_skip_to_human: !!values.allow_skip_to_human,
      abandoned_timeout_minutes: Number(values.abandoned_timeout_minutes) || 0
    }
    await props.submitForm(payload)
  } finally {
    formLoading.value = false
  }
})
</script>
