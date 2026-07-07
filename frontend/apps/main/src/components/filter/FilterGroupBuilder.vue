<template>
  <div class="space-y-2">
    <div class="max-h-[50vh] overflow-y-auto pr-1 pb-2 space-y-2">
      <template v-for="(grp, gi) in modelValue.rules" :key="grp.__id">
        <div v-if="gi > 0" class="flex justify-center">
          <ConnectorToggle :modelValue="modelValue.logic" @update:modelValue="setLogic" />
        </div>
        <FilterGroupCard
          :modelValue="grp"
          :fields="fields"
          :canRemove="modelValue.rules.length > 1"
          @update:modelValue="updateGroup(gi, $event)"
          @remove="removeGroup(gi)"
        />
      </template>
    </div>

    <Button
      type="button"
      variant="outline"
      size="sm"
      :disabled="modelValue.rules.length >= MAX_FILTER_GROUPS"
      @click.stop="addGroup"
    >
      <Plus class="w-3 h-3 mr-1" />
      {{ t('filter.addGroup') }}
    </Button>
  </div>
</template>

<script setup>
import { watch } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Plus } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import FilterGroupCard from '@/components/filter/FilterGroupCard.vue'
import ConnectorToggle from '@/components/filter/ConnectorToggle.vue'
import {
  createRoot,
  createGroup,
  normalizeToTwoLevel,
  isStrictTwoLevel
} from '@/components/filter/filterTree'
import { MAX_FILTER_GROUPS } from '@/constants/filterConfig'

// vee-validate's componentField carries onInput/onChange listeners; without this they fall
// through to the root div and bubbled keystrokes overwrite the whole filters value.
defineOptions({ inheritAttrs: false })
defineProps({
  fields: { type: Array, required: true }
})
const modelValue = defineModel('modelValue', { default: () => createRoot() })
const { t } = useI18n()

watch(
  modelValue,
  (v) => {
    if (!isStrictTwoLevel(v)) modelValue.value = normalizeToTwoLevel(v)
  },
  { immediate: true }
)

const setLogic = (logic) => {
  modelValue.value = { ...modelValue.value, logic }
}

const updateGroup = (index, group) => {
  modelValue.value = {
    ...modelValue.value,
    rules: modelValue.value.rules.map((g, i) => (i === index ? group : g))
  }
}

const addGroup = () => {
  if (modelValue.value.rules.length >= MAX_FILTER_GROUPS) return
  modelValue.value = { ...modelValue.value, rules: [...modelValue.value.rules, createGroup()] }
}

const removeGroup = (index) => {
  let rules = modelValue.value.rules.filter((_, i) => i !== index)
  if (rules.length === 0) rules = [createGroup()]
  modelValue.value = { ...modelValue.value, rules }
}
</script>
