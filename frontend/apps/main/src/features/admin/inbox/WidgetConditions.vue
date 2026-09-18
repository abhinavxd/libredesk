<script setup>
import { ref, watch, onMounted } from 'vue'
import RuleBox from '@/features/admin/automation/RuleBox.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'

const props = defineProps({
  modelValue: { type: Object, default: () => ({ logical_op: 'AND', rules: [] }) }
})
const emit = defineEmits(['update:modelValue'])
const group = ref({ logical_op: 'AND', rules: [] })
watch(
  () => props.modelValue,
  (value) => {
    group.value = value ? JSON.parse(JSON.stringify(value)) : { logical_op: 'AND', rules: [] }
  },
  { immediate: true, deep: true }
)
const update = () => emit('update:modelValue', JSON.parse(JSON.stringify(group.value)))
const add = () => {
  group.value.rules.push({
    field: '',
    field_type: 'contact_custom_attribute',
    operator: '',
    value: '',
    case_sensitive_match: false
  })
  update()
}
const remove = (_, index) => {
  group.value.rules.splice(index, 1)
  update()
}
onMounted(() => useCustomAttributeStore().fetchCustomAttributes())
</script>

<template>
  <RuleBox
    :rule-group="group"
    :group-index="0"
    type="new_conversation"
    contacts-only
    @update-group="update"
    @add-condition="add"
    @remove-condition="remove"
  />
</template>
