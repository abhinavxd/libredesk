<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading">
    <div class="space-y-4">
      <form @submit="onSubmit">
        <div class="space-y-6">
          <div class="space-y-6">
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
                  <FormLabel class="text-foreground"> {{ i18n._('Enabled') }} </FormLabel>
                  <FormMessage />
                </div>
              </FormItem>
            </FormField>

            <FormGrid>
              <FormField v-slot="{ field }" name="name">
                <FormItem>
                  <FormLabel>{{ i18n.ngettext('Name', 'Names', 1) }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="" v-bind="field" @keydown.enter.prevent />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>

              <FormField v-slot="{ field }" name="description">
                <FormItem>
                  <FormLabel>{{ i18n.ngettext('Description', 'Descriptions', 1) }}</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="" v-bind="field" @keydown.enter.prevent />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </FormGrid>

            <FormField v-slot="{ componentField, handleInput }" name="type">
              <FormItem class="max-w-xl">
                <FormLabel>{{ i18n.ngettext('Type', 'Types', 1) }}</FormLabel>
                <FormControl>
                  <Select v-bind="componentField" @update:modelValue="handleInput">
                    <SelectTrigger>
                      <SelectValue :placeholder="i18n._('Select type')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="new_conversation">
                          {{ i18n._('New conversation') }}
                        </SelectItem>
                        <SelectItem value="conversation_update">
                          {{ i18n._('Conversation update') }}
                        </SelectItem>
                        <SelectItem value="time_trigger">
                          {{ i18n._('Time triggers') }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <div :class="{ hidden: form.values.type !== 'conversation_update' }">
              <FormField v-slot="{ componentField, handleChange }" name="events">
                <FormItem>
                  <FormLabel>{{ i18n.ngettext('Event', 'Events', 2) }}</FormLabel>
                  <FormControl>
                    <SelectTag
                      v-model="componentField.modelValue"
                      @update:modelValue="handleChange"
                      :items="conversationEventOptions"
                      :placeholder="i18n._('Select events')"
                    >
                    </SelectTag>
                  </FormControl>
                  <FormDescription>{{
                    i18n._('Evaluate rule on these events.')
                  }}</FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </div>

          <h4 class="ui-heading">
            {{ i18n._('Match these rules') }}
          </h4>

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
                type="button"
                :variant="groupOperator === 'AND' ? 'default' : 'outline'"
                @click.prevent="toggleGroupOperator('AND')"
              >
                {{ i18n._('AND') }}
              </Button>
              <Button
                type="button"
                :variant="groupOperator === 'OR' ? 'default' : 'outline'"
                @click.prevent="toggleGroupOperator('OR')"
              >
                {{ i18n._('OR') }}
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
          <h4 class="ui-heading mt-2">
            {{ i18n._('Perform these actions') }}
          </h4>

          <ActionBox
            :actions="getActions()"
            :update-actions="handleUpdateActions"
            @add-action="handleAddAction"
            @remove-action="handleRemoveAction"
          />
          <Button type="submit" :isLoading="isLoading">{{
            isNewForm ? i18n._('Create') : i18n._('Save')
          }}</Button>
        </div>
      </form>
    </div>
  </LoadingOverlay>
</template>

<script setup>
import FormGrid from '@shared-ui/components/FormGrid.vue'
import { onMounted, ref, computed } from 'vue'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'
import RuleBox from '@/features/admin/automation/RuleBox.vue'
import ActionBox from '@/features/admin/automation/ActionBox.vue'
import api from '@main/api'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '@main/features/admin/automation/formSchema.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { SelectTag } from '@shared-ui/components/ui/select'
import { OPERATOR } from '@main/constants/filterConfig'
import { useI18n } from '@shared-ui/i18n'
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
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useRoute } from 'vue-router'
import { useRouter } from 'vue-router'

const isLoading = ref(false)
const i18n = useI18n()
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
    label: i18n._('Agent assigned'),
    value: 'conversation.user.assigned'
  },
  {
    label: i18n._('Team assigned'),
    value: 'conversation.team.assigned'
  },
  { label: i18n._('Priority change'), value: 'conversation.priority.change' },
  { label: i18n._('Status change'), value: 'conversation.status.change' },
  { label: i18n._('Outgoing message'), value: 'conversation.message.outgoing' },
  { label: i18n._('Incoming message'), value: 'conversation.message.incoming' }
]

const props = defineProps({
  id: {
    type: [String, Number],
    required: false
  }
})

const breadcrumbPageLabel = () => {
  if (props.id > 0) return i18n._('Edit rule')
  return i18n._('New rule')
}

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: 'automation-list', label: i18n._('{count, plural, one {Automation} other {Automations}}') },
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
  validationSchema: toTypedSchema(createFormSchema(i18n))
})

const onSubmit = form.handleSubmit(async (values) => {
  handleSave(values)
})

const handleSave = async (values) => {
  const validationError = getRulesValidationError()
  if (validationError) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'warning',
      description: validationError
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
      router.push({ name: 'automation-list' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
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

// Returns a specific validation error message, or empty string if valid.
const getRulesValidationError = () => {
  // Must have groups.
  if (rule.value.rules[0].groups.length == 0) {
    return i18n._('Please add at least one condition.')
  }

  // At least one group should have at least one rule.
  const group1HasRules = rule.value.rules[0].groups[0].rules.length > 0
  const group2HasRules = rule.value.rules[0].groups[1].rules.length > 0
  if (!group1HasRules && !group2HasRules) {
    return i18n._('Please add at least one condition.')
  }

  // For both groups, each rule should have field, operator, and value.
  for (const group of rule.value.rules[0].groups) {
    for (const rule of group.rules) {
      if (!rule.field) {
        return i18n._('Please select a field for all conditions.')
      }
      if (!rule.operator) {
        return i18n._('Please select an operator for all conditions.')
      }
      // For 'set' and 'not set' operator, value is not required.
      if (rule.operator !== OPERATOR.SET && rule.operator !== OPERATOR.NOT_SET && !rule.value) {
        return i18n._('Please set a value for all conditions.')
      }
    }
  }

  // Must have at least one action.
  if (rule.value.rules[0].actions.length == 0) {
    return i18n._('Please add at least one action.')
  }

  // Make sure each action has a type and value.
  for (const action of rule.value.rules[0].actions) {
    if (!action.type) {
      return i18n._('Please select a type for all actions.')
    }

    // CSAT action does not require value, set dummy value.
    if (action.type === 'send_csat') {
      action.value = ['0']
    }

    // Notify uses its own fields instead of value.
    if (action.type === 'notify') {
      if (!action.subject?.trim() || !action.message?.trim() || !action.recipients?.length) {
        return i18n._('Please set a value for all actions.')
      }
      continue
    }

    // Empty array, no value selected.
    if (action.value.length === 0) {
      return i18n._('Please set a value for all actions.')
    }

    // Check if all values are present.
    for (const key in action.value) {
      if (!action.value[key]) {
        return i18n._('Please set a value for all actions.')
      }
    }
  }
  return ''
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
      form.setValues(resp.data.data, false)
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
