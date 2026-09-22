<script setup>
const MUTED_TEXT_CLASS = 'text-sm text-muted-foreground'

import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { buttonVariants } from '@shared-ui/components/ui/button'
import RuleBox from '@/features/admin/automation/RuleBox.vue'
import { useCustomAttributeStore } from '@/stores/customAttributes'

const props = defineProps({
  modelValue: { type: Object, default: () => ({ logical_op: 'AND', rules: [] }) }
})
const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()
const customAttributeStore = useCustomAttributeStore()
const group = ref({ logical_op: 'AND', rules: [] })
const isLoading = ref(true)
const hasContactCustomAttributes = computed(
  () => customAttributeStore.contactAttributeOptions.length > 0
)
watch(
  () => props.modelValue,
  (value) => {
    const next = value
      ? JSON.parse(JSON.stringify(value))
      : { logical_op: 'AND', rules: [] }
    if (JSON.stringify(next) !== JSON.stringify(group.value)) group.value = next
  },
  { immediate: true, deep: true }
)
const update = () => {
  emit('update:modelValue', JSON.parse(JSON.stringify(group.value)))
}
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
onMounted(async () => {
  await customAttributeStore.refreshCustomAttributes()
  isLoading.value = false
})
</script>

<template>
  <RuleBox
    v-if="hasContactCustomAttributes"
    :rule-group="group"
    :group-index="0"
    type="new_conversation"
    contacts-only
    @update-group="update"
    @add-condition="add"
    @remove-condition="remove"
  />
  <p v-else-if="isLoading" role="status" :class="MUTED_TEXT_CLASS">
    {{ t('globals.terms.loading') }}
  </p>
  <div v-else class="space-y-3 rounded-md border border-dashed p-4">
    <p :class="MUTED_TEXT_CLASS">
      {{ t('widget.campaignContactAttributesEmpty') }}
    </p>
    <RouterLink
      :to="{ name: 'custom-attributes' }"
      :class="buttonVariants({ variant: 'link', size: 'sm' })"
    >
      {{ t('globals.messages.addContactCustomAttribute') }}
    </RouterLink>
  </div>
</template>
