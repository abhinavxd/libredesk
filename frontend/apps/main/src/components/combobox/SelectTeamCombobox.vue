<template>
  <SelectTag
    v-if="multiple"
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="teamStore.searchTeams"
  />
  <SelectComboBox
    v-else
    v-bind="$attrs"
    :model-value="modelValue"
    :items="items"
    :placeholder="placeholderText"
    :search="teamStore.searchTeams"
    type="team"
  >
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>
  </SelectComboBox>
</template>

<script setup>
import { computed, onMounted, watch } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { useTeamStore } from '@/stores/team'
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
  includeNone: {
    type: Boolean,
    default: false
  },
  prependItems: {
    type: Array,
    default: () => []
  }
})

const i18n = useI18n()
const teamStore = useTeamStore()

const placeholderText = computed(() => props.placeholder || i18n._('Select team'))

const items = computed(() => {
  const prepended = props.includeNone
    ? [{ value: 'none', label: i18n.pgettext('No assigned team', 'None') }, ...props.prependItems]
    : props.prependItems
  return [...prepended, ...teamStore.options]
})

onMounted(teamStore.fetchTeams)

watch(
  () => props.modelValue,
  (value) => {
    const ids = Array.isArray(value) ? value : [value]
    teamStore.ensureTeamIDs(ids)
  },
  { immediate: true }
)
</script>
