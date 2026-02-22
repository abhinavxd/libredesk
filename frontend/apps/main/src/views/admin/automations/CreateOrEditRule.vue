<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <div class="space-y-4">
    <p>{{ formTitle }}</p>
    <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
      <form @submit="onSubmit">
        <div class="space-y-5">
          <div class="space-y-5">
            <FormField
              v-slot="{ value, handleChange }"
              type="checkbox"
              name="enabled"
              v-if="!isNewForm"
            >
              <FormItem class="flex flex-row items-start gap-x-3 space-y-0">
                <FormControl>
                  <Checkbox :checked="value" @update:checked="handleChange" />
                </FormControl>
                <div class="space-y-1 leading-none">
                  <FormLabel> {{ $t('globals.terms.enabled') }} </FormLabel>
                  <FormMessage />
                </div>
              </FormItem>
            </FormField>

            <FormField v-slot="{ field }" name="name">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="" v-bind="field" />
                </FormControl>
                <FormDescription>{{ $t('admin.automation.name.description') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ field }" name="description">
              <FormItem>
                <FormLabel>{{ $t('globals.terms.description') }}</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="" v-bind="field" />
                </FormControl>
                <FormDescription>{{
                  $t('admin.automation.description.description')
                }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField, handleInput }" name="type">
              <FormItem>
                <FormLabel>Type</FormLabel>
                <FormControl>
                  <Select v-bind="componentField" @update:modelValue="handleInput">
                    <SelectTrigger>
                      <SelectValue
                        :placeholder="t('placeholders.selectType')"
                      />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="new_conversation">
                          {{ $t('conversation.newConversation') }}
                        </SelectItem>
                        <SelectItem value="conversation_update">
                          {{ $t('admin.automation.conversationUpdate') }}
                        </SelectItem>
                        <SelectItem value="time_trigger">
                          {{ $t('admin.automation.timeTriggers') }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>{{ $t('automation.typeOfRule') }}</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <div :class="{ hidden: form.values.type !== 'conversation_update' }">
              <FormField v-slot="{ componentField, handleChange }" name="events">
                <FormItem>
                  <FormLabel>{{ $t('globals.terms.event', 2) }}</FormLabel>
                  <FormControl>
                    <SelectTag
                      v-model="componentField.modelValue"
                      @update:modelValue="handleChange"
                      :items="conversationEventOptions"
                      :placeholder="t('placeholders.selectEvents')"
                    >
                    </SelectTag>
                  </FormControl>
                  <FormDescription>{{
                    $t('admin.automation.evaluateRuleOnTheseEvents')
                  }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </div>

          <p class="font-semibold">{{ $t('admin.automation.matchTheseRules') }}</p>

          <RuleBox
            v-if="form.values.type"
            :ruleGroup="firstRuleGroup"
            @update-group="handleUpdateGroup"
            @add-condition="handleAddCondition"
            @remove-condition="handleRemoveCondition"
            :type="form.values.type"
            :groupIndex="0"
          />

          <div class="flex justify-center">
            <div class="flex items-center space-x-2">
              <Button
                :class="[
                  groupOperator === 'AND'
                    ? 'bg-black text-white dark:bg-white dark:text-black'
                    : 'bg-gray-100 text-black dark:bg-secondary dark:text-white'
                ]"
                @click.prevent="toggleGroupOperator('AND')"
              >
                {{ $t('admin.automation.and') }}
              </Button>
              <Button
                :class="[
                  groupOperator === 'OR'
                    ? 'bg-black text-white dark:bg-white dark:text-black'
                    : 'bg-gray-100 text-black dark:bg-secondary dark:text-white'
                ]"
                @click.prevent="toggleGroupOperator('OR')"
              >
                {{ $t('admin.automation.or') }}
              </Button>
            </div>
          </div>

          <RuleBox
            v-if="form.values.type"
            :ruleGroup="secondRuleGroup"
            @update-group="handleUpdateGroup"
            @add-condition="handleAddCondition"
            @remove-condition="handleRemoveCondition"
            :type="form.values.type"
            :groupIndex="1"
          />
          <p class="font-semibold">{{ $t('admin.automation.performTheseActions') }}</p>

          <ActionBox
            :actions="getActions()"
            :update-actions="handleUpdateActions"
            @add-action="handleAddAction"
            @remove-action="handleRemoveAction"
          />
          <Button type="submit" :isLoading="isLoading">{{ $t('globals.messages.save') }}</Button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import RuleBox from '@/features/admin/automation/RuleBox.vue'
import ActionBox from '@/features/admin/automation/ActionBox.vue'
import api from '../../../api'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '../../../features/admin/automation/formSchema.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { SelectTag } from '@shared-ui/components/ui/select'
import { OPERATOR } from '../../../constants/filterConfig'
import { useI18n } from 'vue-i18n'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useRoute } from 'vue-router'
import { useRouter } from 'vue-router'

const isLoading = ref(false)
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const rule = ref({
  id: 0,
  name: '',
  description: '',
  type: 'new_conversation',
  rules: [
    {
      groups: [
        {
          rules: [],
          logical_op: 'OR'
        },
        {
          rules: [],
          logical_op: 'OR'
        }
      ],
      actions: [],
      group_operator: 'OR'
    }
  ]
})

const conversationEventOptions = [
  {
    label: t('conversation.agentAssigned'),
    value: 'conversation.user.assigned'
  },
  {
    label: t('conversation.teamAssigned'),
    value: 'conversation.team.assigned'
  },
  { label: t('admin.automation.event.priority.change'), value: 'conversation.priority.change' },
  { label: t('admin.automation.event.status.change'), value: 'conversation.status.change' },
  { label: t('admin.automation.event.message.outgoing'), value: 'conversation.message.outgoing' },
  { label: t('admin.automation.event.message.incoming'), value: 'conversation.message.incoming' }
]

const props = defineProps({
  id: {
    type: [String, Number],
    required: false
  }
})

const breadcrumbPageLabel = () => {
  if (props.id > 0) return t('automation.editRule')
  return t('automation.newRule')
}

const formTitle = computed(() => {
  return ''
  // if (props.id > 0) return 'Edit existing rule'
  // return 'Create new rule'
})

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: 'automations', label: t('globals.terms.automation') },
  { path: '', label: breadcrumbPageLabel() }
]

const firstRuleGroup = ref([])
const secondRuleGroup = ref([])
const groupOperator = ref('')

const getFirstGroup = () => {
  if (rule.value.rules?.[0]?.groups?.[0]) {
    return rule.value.rules[0].groups[0]
  }
  return []
}

const getSecondGroup = () => {
  if (rule.value.rules?.[0]?.groups?.[1]) {
    return rule.value.rules[0].groups[1]
  }
  return []
}

const getActions = () => {
  if (rule.value.rules?.[0]?.actions) {
    return rule.value.rules[0].actions
  }
  return []
}

const toggleGroupOperator = (value) => {
  if (rule.value.rules?.[0]) {
    rule.value.rules[0].group_operator = value
    groupOperator.value = value
  }
}

const getGroupOperator = () => {
  if (rule.value.rules?.[0]) {
    return rule.value.rules[0].group_operator
  }
  return ''
}

const handleUpdateGroup = (value, groupIndex) => {
  rule.value.rules[0].groups[groupIndex] = value.value
}

const handleAddCondition = (groupIndex) => {
  rule.value.rules[0].groups[groupIndex].rules.push({})
}

const handleRemoveCondition = (groupIndex, ruleIndex) => {
  rule.value.rules[0].groups[groupIndex].rules.splice(ruleIndex, 1)
}

const handleUpdateActions = (value, index) => {
  rule.value.rules[0].actions[index] = value
}

const handleAddAction = () => {
  rule.value.rules[0].actions.push({})
}

const handleRemoveAction = (index) => {
  rule.value.rules[0].actions.splice(index, 1)
}

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSubmit = form.handleSubmit(async (values) => {
  handleSave(values)
})

const handleSave = async (values) => {
  if (!areRulesValid()) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('admin.automation.invalid')
    })
    return
  }

  try {
    isLoading.value = true
    const updatedRule = { ...rule.value, ...values }
    // Delete fields not required.
    delete updatedRule.created_at
    delete updatedRule.updated_at
    if (props.id > 0) {
      await api.updateAutomationRule(props.id, updatedRule)
    } else {
      await api.createAutomationRule(updatedRule)
      router.push({ name: 'automations' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

// TODO: Maybe we can do some vee validate magic here.
const areRulesValid = () => {
  // Must have groups.
  if (rule.value.rules[0].groups.length == 0) {
    return false
  }

  // At least one group should have at least one rule
  const group1HasRules = rule.value.rules[0].groups[0].rules.length > 0
  const group2HasRules = rule.value.rules[0].groups[1].rules.length > 0
  if (!group1HasRules && !group2HasRules) {
    return false
  }

  // For both groups, each rule should have value, operator and field.
  for (const group of rule.value.rules[0].groups) {
    for (const rule of group.rules) {
      if (!rule.field || !rule.operator) {
        return false
      }
      // For 'set' and `not set` operator, value is not required.
      if (rule.operator !== OPERATOR.SET && rule.operator !== OPERATOR.NOT_SET && !rule.value) {
        return false
      }
    }
  }

  // Must have atleast one action.
  if (rule.value.rules[0].actions.length == 0) {
    return false
  }

  // Make sure each action has value.
  for (const action of rule.value.rules[0].actions) {
    // CSAT action does not require value, set dummy value.
    if (action.type === 'send_csat') {
      action.value = ['0']
    }

    // Empty array, no value selected.
    if (action.value.length === 0) {
      return false
    }

    // Check if all values are present.
    for (const key in action.value) {
      if (!action.value[key]) {
        return false
      }
    }
  }
  return true
}

onMounted(async () => {
  if (props.id > 0) {
    try {
      isLoading.value = true
      let resp = await api.getAutomationRule(props.id)
      rule.value = resp.data.data
      if (resp.data.data.type === 'conversation_update') {
        rule.value.rules.events = []
      }
      form.setValues(resp.data.data)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  }
  if (route.query.type) {
    form.setFieldValue('type', route.query.type)
  }
  firstRuleGroup.value = getFirstGroup()
  secondRuleGroup.value = getSecondGroup()
  groupOperator.value = getGroupOperator()
})
</script>
