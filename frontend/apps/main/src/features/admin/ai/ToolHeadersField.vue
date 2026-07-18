<template>
  <div class="space-y-2">
    <div v-for="(header, index) in headers" :key="index" class="flex items-start gap-2">
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
      <Plus class="w-3 h-3 mr-1" />
      {{ t('admin.ai.tool.addHeader') }}
    </Button>
  </div>
</template>

<script setup>
import { Plus, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'

const headers = defineModel({ type: Array, required: true })
const { t } = useI18n()

const addHeader = () => {
  headers.value = [...headers.value, { key: '', value: '' }]
}

const removeHeader = (index) => {
  headers.value = headers.value.filter((_, i) => i !== index)
}

const updateHeader = (index, field, value) => {
  headers.value = headers.value.map((h, i) => (i === index ? { ...h, [field]: value } : h))
}
</script>
