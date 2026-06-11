<template>
  <div class="group flex items-center gap-3">
    <div class="flex gap-2 w-full">
      <div
        class="flex-1 rounded-md"
        :class="[shake && missingField && 'animate-shake', showInvalid && missingField && 'ring-1 ring-destructive']"
      >
        <Select :model-value="modelValue.field" @update:model-value="onFieldChange">
          <SelectTrigger>
            <SelectValue :placeholder="t('placeholders.selectField')" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem v-for="field in fields" :key="field.field" :value="field.field">
                {{ field.label }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div
        class="flex-1 rounded-md"
        :class="[shake && missingOperator && 'animate-shake', showInvalid && missingOperator && 'ring-1 ring-destructive']"
      >
        <Select
          v-if="modelValue.field"
          :model-value="modelValue.operator"
          @update:model-value="onOperatorChange"
        >
          <SelectTrigger>
            <SelectValue :placeholder="t('placeholders.selectOperator')" />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem v-for="op in fieldOperators" :key="op" :value="op">
                {{ opLabel(op) }}
              </SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div
        class="flex-1 rounded-md"
        :class="[shake && missingValue && 'animate-shake', showInvalid && missingValue && 'ring-1 ring-destructive']"
      >
        <div v-if="modelValue.field && modelValue.operator">
          <template v-if="modelValue.operator !== OPERATOR.SET && modelValue.operator !== OPERATOR.NOT_SET">
            <SelectTag
              v-if="fieldType === FIELD_TYPE.MULTI_SELECT"
              v-model="leafValue"
              :items="fieldOptions"
              :placeholder="t('placeholders.selectTags')"
            />

            <SelectComboBox
              v-else-if="fieldOptions.length > 0 && modelValue.field === 'assigned_user_id'"
              v-model="leafValue"
              :items="fieldOptions"
              :placeholder="t('placeholders.selectValue')"
              type="user"
            />

            <SelectComboBox
              v-else-if="fieldOptions.length > 0 && modelValue.field === 'assigned_team_id'"
              v-model="leafValue"
              :items="fieldOptions"
              :placeholder="t('placeholders.selectValue')"
              type="team"
            />

            <SelectComboBox
              v-else-if="fieldOptions.length > 0"
              v-model="leafValue"
              :items="fieldOptions"
              :placeholder="t('placeholders.selectValue')"
            />

            <DateFilterValue
              v-else-if="fieldType === FIELD_TYPE.DATE"
              v-model="leafValue"
              :range="modelValue.operator === OPERATOR.BETWEEN"
            />

            <Input v-else v-model="leafValue" :placeholder="t('globals.terms.value')" type="text" />
          </template>
        </div>
      </div>
    </div>
    <CloseButton type="button" :onClose="() => emit('remove')" />
  </div>
</template>

<script setup>
import { computed, inject, ref, watch, onUnmounted, nextTick } from 'vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Input } from '@shared-ui/components/ui/input'
import { useI18n } from 'vue-i18n'
import { FIELD_TYPE, OPERATOR, operatorLabel } from '@/constants/filterConfig'
import CloseButton from '@/components/button/CloseButton.vue'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import SelectTag from '@shared-ui/components/ui/select/SelectTag.vue'
import DateFilterValue from '@/components/filter/DateFilterValue.vue'

const props = defineProps({
  fields: {
    type: Array,
    required: true
  }
})
const emit = defineEmits(['remove'])
const modelValue = defineModel('modelValue', { required: true })
const { t } = useI18n()

const fieldConfig = computed(() => props.fields.find((f) => f.field === modelValue.value.field))
const fieldOptions = computed(() => fieldConfig.value?.options || [])
const fieldOperators = computed(() => fieldConfig.value?.operators || [])
const fieldType = computed(() => fieldConfig.value?.type || '')

// "contains any of" only applies to multi-value fields; single-value text stays "contains"
const opLabel = (op) => (fieldType.value === FIELD_TYPE.MULTI_SELECT ? operatorLabel(op, t) : op)

const isEmptyValue = (v) =>
  v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0)
const missingField = computed(() => !modelValue.value.field)
const missingOperator = computed(() => !!modelValue.value.field && !modelValue.value.operator)
const needsValue = computed(
  () =>
    !!modelValue.value.operator &&
    modelValue.value.operator !== OPERATOR.SET &&
    modelValue.value.operator !== OPERATOR.NOT_SET
)
const missingValue = computed(() => needsValue.value && isEmptyValue(modelValue.value.value))
const invalid = computed(() => missingField.value || missingOperator.value || missingValue.value)

const validateTick = inject('filterValidateTick', ref(0))
const showInvalid = computed(() => validateTick.value > 0)
const shake = ref(false)
let shakeTimer = null
watch(validateTick, async () => {
  if (!invalid.value) return
  shake.value = false
  await nextTick()
  shake.value = true
  clearTimeout(shakeTimer)
  shakeTimer = setTimeout(() => {
    shake.value = false
  }, 500)
})
onUnmounted(() => clearTimeout(shakeTimer))

// All edits emit a fresh leaf; the tree is never mutated in place.
const patch = (changes) => {
  modelValue.value = { ...modelValue.value, ...changes }
}

const leafValue = computed({
  get: () => modelValue.value.value,
  set: (value) => patch({ value })
})

const onFieldChange = (field) => {
  const config = props.fields.find((f) => f.field === field)
  patch({
    field,
    model: config?.model || '',
    operator: '',
    value: config?.type === FIELD_TYPE.MULTI_SELECT ? [] : ''
  })
}

const onOperatorChange = (operator) => {
  if (modelValue.value.operator === operator) return
  if (modelValue.value.operator === OPERATOR.BETWEEN || operator === OPERATOR.BETWEEN) {
    patch({ operator, value: '' })
    return
  }
  patch({ operator })
}
</script>
