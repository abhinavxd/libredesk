<template>
  <SelectTag
    v-if="multiple"
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="searchTags"
  />
  <SelectComboBox
    v-else
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="searchTags"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTagStore } from '@/stores/tag'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import { SelectTag } from '@shared-ui/components/ui/select'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: {
    type: [String, Number, Array],
    default: undefined
  },
  placeholder: {
    type: String,
    default: ''
  },
  multiple: {
    type: Boolean,
    default: false
  },
  // Conversations store tags as names; filters store them as ids.
  valueField: {
    type: String,
    default: 'name',
    validator: (value) => ['name', 'id'].includes(value)
  }
})

const { t } = useI18n()
const tagStore = useTagStore()

const placeholderText = computed(() => props.placeholder || t('placeholders.selectTags'))

const items = computed(() =>
  props.valueField === 'id'
    ? tagStore.tagOptions
    : tagStore.tagNames.map((name) => ({ label: name, value: name }))
)

const searchTags = (query) =>
  props.valueField === 'id'
    ? tagStore.searchTagOptions(query)
    : tagStore.searchTagNames(query)

onMounted(tagStore.fetchTags)

watch(
  () => props.modelValue,
  (value) => {
    if (props.valueField !== 'id') return
    const ids = Array.isArray(value) ? value : [value]
    tagStore.ensureTagIDs(ids)
  },
  { immediate: true }
)
</script>
