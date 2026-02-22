<template>
  <div class="space-y-6">
    <!-- Empty State -->
    <div
      v-if="!model.length"
      class="text-center py-12 px-6 border-2 border-dashed border-muted rounded-lg"
    >
      <div class="mx-auto w-12 h-12 bg-muted rounded-full flex items-center justify-center mb-3">
        <Plus class="w-6 h-6 text-muted-foreground" />
      </div>
      <h3 class="text-sm font-medium text-foreground mb-2">
        {{ $t('actions.noActions') }}
      </h3>
      <Button
        @click.prevent="add"
        variant="outline"
        size="sm"
        class="inline-flex items-center gap-2"
      >
        <Plus class="w-4 h-4" />
        {{ config.addButtonText }}
      </Button>
    </div>

    <!-- Actions List -->
    <div v-else class="space-y-6">
      <div v-for="(action, index) in model" :key="index" class="relative">
        <!-- Action Card -->
        <div class="border rounded p-6 shadow-sm hover:shadow-md transition-shadow">
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1 space-y-4">
              <!-- Action Type Selection -->
              <div class="flex flex-col sm:flex-row gap-4">
                <div class="flex-1 max-w-xs">
                  <label class="block text-sm font-medium mb-2">{{
                    $t('macro.actionType')
                  }}</label>
                  <Select
                    v-model="action.type"
                    @update:modelValue="(value) => updateField(value, index)"
                  >
                    <SelectTrigger class="w-full">
                      <SelectValue :placeholder="config.typePlaceholder" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem
                          v-for="(actionConfig, key) in config.actions"
                          :key="key"
                          :value="key"
                        >
                          {{ actionConfig.label }}
                        </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </div>

                <!-- Value Selection -->
                <div
                  v-if="action.type && config.actions[action.type]?.type === 'select'"
                  class="flex-1 max-w-xs"
                >
                  <label class="block text-sm font-medium mb-2">{{ $t('globals.terms.value', 1) }}</label>

                  <SelectComboBox
                    v-if="action.type === 'assign_user'"
                    v-model="action.value[0]"
                    :items="config.actions[action.type].options"
                    :placeholder="config.valuePlaceholder"
                    @update:modelValue="(value) => updateValue(value, index)"
                    type="user"
                  />

                  <SelectComboBox
                    v-else-if="action.type === 'assign_team'"
                    v-model="action.value[0]"
                    :items="config.actions[action.type].options"
                    :placeholder="config.valuePlaceholder"
                    @update:modelValue="(value) => updateValue(value, index)"
                    type="team"
                  />
                  <SelectComboBox
                    v-else
                    v-model="action.value[0]"
                    :items="config.actions[action.type].options"
                    :placeholder="config.valuePlaceholder"
                    @update:modelValue="(value) => updateValue(value, index)"
                  />
                </div>
              </div>

              <!-- Tag Selection -->
              <div
                v-if="action.type && config.actions[action.type]?.type === 'tag'"
                class="max-w-md"
              >
                <label class="block text-sm font-medium mb-2">{{ $t('globals.terms.tag') }}</label>
                <SelectTag
                  v-model="action.value"
                  :items="tagsStore.tagNames.map((tag) => ({ label: tag, value: tag }))"
                  :placeholder="$t('placeholders.selectTags')"
                />
              </div>
            </div>

            <!-- Remove Button -->
            <CloseButton :onClose="() => remove(index)" />
          </div>
        </div>
      </div>

      <!-- Add Action Button -->
      <div class="flex justify-center pt-2">
        <Button
          type="button"
          variant="outline"
          @click="add"
          class="inline-flex items-center gap-2 border-dashed hover:border-solid"
        >
          <Plus class="w-4 h-4" />
          {{ config.addButtonText }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { Plus } from 'lucide-vue-next'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import CloseButton from '@main/components/button/CloseButton.vue'
import { SelectTag } from '@shared-ui/components/ui/select'
import { useTagStore } from '../../../stores/tag'
import SelectComboBox from '@main/components/combobox/SelectCombobox.vue'

const model = defineModel('actions', {
  type: Array,
  required: true,
  default: () => []
})

defineProps({
  config: {
    type: Object,
    required: true
  }
})

const tagsStore = useTagStore()

const updateField = (value, index) => {
  const newModel = [...model.value]
  newModel[index] = { type: value, value: [] }
  model.value = newModel
}

const updateValue = (value, index) => {
  const newModel = [...model.value]
  newModel[index] = {
    ...newModel[index],
    value: [value?.value ?? value]
  }
  model.value = newModel
}

const remove = (index) => {
  model.value = model.value.filter((_, i) => i !== index)
}

const add = () => {
  model.value = [...model.value, { type: '', value: [] }]
}
</script>
