<template>
  <div class="space-y-2">
    <div v-for="(header, index) in rows" :key="header.id" class="flex items-start gap-2">
      <Input
        :model-value="header.key"
        type="text"
        :placeholder="t('admin.ai.tool.headerKeyPlaceholder')"
        class="flex-1"
        @update:model-value="updateHeader(index, 'key', $event)"
      />
      <Input
        :model-value="header.value"
        type="password"
        autocomplete="new-password"
        :placeholder="t('admin.ai.tool.headerValuePlaceholder')"
        class="flex-1"
        @update:model-value="updateHeader(index, 'value', $event)"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        :aria-label="t('globals.terms.remove')"
        class="text-muted-foreground hover:text-foreground"
        @click="removeHeader(index)"
      >
        <X class="w-4 h-4" />
      </Button>
    </div>

    <Button type="button" variant="ghost" size="sm" class="text-foreground" @click="addHeader">
      <Plus class="w-4 h-4" />
      {{ t('admin.ai.tool.addHeader') }}
    </Button>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Plus, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'

const model = defineModel({ type: Array, required: true })
const { t } = useI18n()

// Local rows carry a stable id so v-for keys survive removals; the emitted model stays {key,value}.
let seq = 0
const rows = ref([])

const emit = () => {
  model.value = rows.value.map(({ key, value }) => ({ key, value }))
}

watch(
  model,
  (val) => {
    const incoming = val || []
    const same =
      incoming.length === rows.value.length &&
      incoming.every((h, i) => h.key === rows.value[i].key && h.value === rows.value[i].value)
    if (!same) rows.value = incoming.map((h) => ({ id: ++seq, key: h.key, value: h.value }))
  },
  { immediate: true, deep: true }
)

const addHeader = () => {
  rows.value.push({ id: ++seq, key: '', value: '' })
  emit()
}

const removeHeader = (index) => {
  rows.value.splice(index, 1)
  emit()
}

const updateHeader = (index, field, value) => {
  rows.value[index][field] = value
  emit()
}
</script>
