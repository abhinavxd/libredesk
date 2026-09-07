<template>
  <div class="space-y-6" :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
    <Spinner v-if="isLoading" />
    <div>
      <p class="text-sm-muted">{{ helptext }}</p>
    </div>
    <div v-if="type === 'new_conversation'">
      <Select v-model="executionMode" v-if="rules.length > 0" @update:modelValue="updateExecutionMode">
        <SelectTrigger class="w-fit flex gap-2">
          <Settings size="16" />
          <SelectValue>{{
            executionMode === 'first_match'
              ? $t('admin.automation.executeFirstMatchingRule')
              : $t('admin.automation.executeAllMatchingRules')
          }}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="first_match">{{
            $t('admin.automation.executeFirstMatchingRule')
          }}</SelectItem>
          <SelectItem value="all">{{ $t('admin.automation.executeAllMatchingRules') }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <div
      v-if="!isLoading && rules.length === 0"
      class="flex flex-col items-center justify-center py-12 px-4"
    >
      <div class="text-center space-y-2">
        <p class="text-muted-foreground">
          {{ $t('admin.automation.noRulesFound') }}
        </p>
      </div>
    </div>

    <div class="space-y-4">
      <div v-if="type === 'new_conversation'">
        <draggable v-model="rules" class="space-y-6" item-key="id" @end="onDragEnd">
          <template #item="{ element }">
            <div class="draggable-item">
              <RuleList :rule="element" @delete-rule="deleteRule" @toggle-rule="toggleRule" />
            </div>
          </template>
        </draggable>
      </div>
      <div v-else class="space-y-6">
        <RuleList
          v-for="rule in rules"
          :key="rule.id"
          :rule="rule"
          @delete-rule="deleteRule"
          @toggle-rule="toggleRule"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import RuleList from './RuleList.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Settings } from 'lucide-vue-next'
import draggable from 'vuedraggable'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const { t } = useI18n()
const emitter = useEmitter()

const isLoading = ref(false)
const rules = ref([])
const executionMode = ref('all')
const props = defineProps({
  type: {
    type: String,
    required: true
  },
  helptext: {
    type: String,
    required: false
  }
})

onMounted(() => {
  fetchRules()
})

const fetchRules = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAutomationRules(props.type)
    rules.value = resp.data.data
    executionMode.value = resp.data.data[0]?.execution_mode || 'all'
  } finally {
    isLoading.value = false
  }
}

const deleteRule = async (id) => {
  await api.deleteAutomationRule(id)
  fetchRules()
}

const toggleRule = async (id) => {
  await api.toggleAutomationRule(id)
  fetchRules()
}

const onDragEnd = async () => {
  try {
    const weights = {}
    rules.value.forEach((rule, index) => {
      weights[rule.id] = index + 1
    })
    await api.updateAutomationRuleWeights(weights)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const updateExecutionMode = async () => {
  try {
    await api.updateAutomationRulesExecutionMode({
      mode: executionMode.value
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>

<style scoped>
.draggable-item {
  cursor: grab;
}

.draggable-item:active {
  cursor: grabbing;
}
</style>
