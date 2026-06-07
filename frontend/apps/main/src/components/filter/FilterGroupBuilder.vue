<template>
  <div class="space-y-2">
    <template v-for="(grp, gi) in modelValue.rules" :key="grp.__id">
      <div v-if="gi > 0" class="flex justify-center">
        <ConnectorToggle v-model:modelValue="modelValue.logic" />
      </div>
      <FilterGroupCard
        v-model:modelValue="modelValue.rules[gi]"
        :fields="fields"
        :canRemove="modelValue.rules.length > 1"
        @remove="removeGroup(gi)"
      />
    </template>

    <Button type="button" variant="outline" size="sm" @click.stop="addGroup">
      <Plus class="w-3 h-3 mr-1" />
      {{ t('filter.addGroup') }}
    </Button>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Plus } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import FilterGroupCard from '@/components/filter/FilterGroupCard.vue'
import ConnectorToggle from '@/components/filter/ConnectorToggle.vue'
import { createRoot, createGroup, normalizeToTwoLevel, isGroupNode } from '@/components/filter/filterTree'

defineProps({
  fields: { type: Array, required: true }
})
const modelValue = defineModel('modelValue', { default: () => createRoot() })
const { t } = useI18n()

onMounted(() => {
  const v = modelValue.value
  const isStrictTwoLevel =
    v &&
    typeof v === 'object' &&
    !Array.isArray(v) &&
    Array.isArray(v.rules) &&
    v.rules.length > 0 &&
    v.rules.every((g) => isGroupNode(g) && Array.isArray(g.rules) && g.rules.every((r) => !isGroupNode(r)))
  if (!isStrictTwoLevel) modelValue.value = normalizeToTwoLevel(v)
})

const addGroup = () => modelValue.value.rules.push(createGroup())
const removeGroup = (index) => {
  modelValue.value.rules.splice(index, 1)
  if (modelValue.value.rules.length === 0) modelValue.value.rules.push(createGroup())
}
</script>
