<template>
  <div class="rounded-lg border border-border bg-muted/30 p-3 space-y-2">
    <div v-if="canRemove" class="flex justify-end">
      <Button
        type="button"
        variant="ghost"
        size="sm"
        class="text-muted-foreground hover:text-foreground h-7 px-2"
        @click.stop="emit('remove')"
      >
        {{ t('filter.removeGroup') }}
      </Button>
    </div>

    <template v-for="(rule, index) in group.rules" :key="rule.__id">
      <ConnectorToggle
        v-if="index > 0"
        :modelValue="group.logic"
        @update:modelValue="setLogic"
      />
      <FilterRow
        :modelValue="rule"
        :fields="fields"
        @update:modelValue="updateRule(index, $event)"
        @remove="removeRule(index)"
      />
    </template>

    <Button
      type="button"
      variant="ghost"
      size="sm"
      class="text-muted-foreground"
      @click.stop="addCondition"
    >
      <Plus class="w-3 h-3 mr-1" />
      {{ t('actions.addCondition') }}
    </Button>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { Plus } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import FilterRow from '@/components/filter/FilterRow.vue'
import ConnectorToggle from '@/components/filter/ConnectorToggle.vue'
import { createLeaf } from '@/components/filter/filterTree'

defineProps({
  fields: { type: Array, required: true },
  canRemove: { type: Boolean, default: false }
})
const emit = defineEmits(['remove'])
const group = defineModel('modelValue', { required: true })
const { t } = useI18n()

const setLogic = (logic) => {
  group.value = { ...group.value, logic }
}

const updateRule = (index, rule) => {
  group.value = { ...group.value, rules: group.value.rules.map((r, i) => (i === index ? rule : r)) }
}

const addCondition = () => {
  group.value = { ...group.value, rules: [...group.value.rules, createLeaf()] }
}

const removeRule = (index) => {
  const rules = group.value.rules.filter((_, i) => i !== index)
  if (rules.length === 0) {
    emit('remove')
    return
  }
  group.value = { ...group.value, rules }
}
</script>
