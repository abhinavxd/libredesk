<template>
  <div class="rounded-lg border border-border bg-muted/30 p-3">
    <div class="flex items-start gap-2">
      <div class="flex-1 space-y-2">
        <template v-for="(rule, index) in group.rules" :key="rule.__id">
          <ConnectorToggle v-if="index > 0" v-model:modelValue="group.logic" />
          <FilterRow v-model:modelValue="group.rules[index]" :fields="fields" @remove="removeRule(index)" />
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
      <CloseButton v-if="canRemove" type="button" :onClose="() => emit('remove')" />
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { Plus } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import FilterRow from '@/components/filter/FilterRow.vue'
import ConnectorToggle from '@/components/filter/ConnectorToggle.vue'
import CloseButton from '@/components/button/CloseButton.vue'
import { createLeaf } from '@/components/filter/filterTree'

defineProps({
  fields: { type: Array, required: true },
  canRemove: { type: Boolean, default: false }
})
const emit = defineEmits(['remove'])
const group = defineModel('modelValue', { required: true })
const { t } = useI18n()

const addCondition = () => group.value.rules.push(createLeaf())
const removeRule = (index) => {
  group.value.rules.splice(index, 1)
  if (group.value.rules.length === 0) emit('remove')
}
</script>
