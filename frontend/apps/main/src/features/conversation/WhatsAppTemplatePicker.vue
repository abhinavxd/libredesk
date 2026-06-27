<template>
  <div :class="{ 'flex flex-col min-h-0': fill }">
    <p v-if="isFetching" class="py-2 text-sm text-muted-foreground">
      {{ $t('globals.messages.loading') }}
    </p>
    <p v-else-if="!approvedTemplates.length" class="py-2 text-sm text-muted-foreground">
      {{ $t('conversation.whatsapp.noApprovedTemplates') }}
    </p>

    <div
      v-else-if="!selectedTemplate"
      class="overflow-y-auto space-y-2"
      :class="fill ? 'flex-1 min-h-0' : 'max-h-80'"
    >
      <button
        v-for="tmpl in approvedTemplates"
        :key="tmpl.id"
        type="button"
        class="w-full text-left box p-3 hover:bg-accent transition-colors"
        @click="$emit('pick', tmpl)"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="font-mono text-sm">{{ tmpl.name }}</div>
          <Badge variant="outline">{{ tmpl.language }}</Badge>
        </div>
        <div class="text-xs text-muted-foreground mt-1 line-clamp-2">{{ tmpl.body_content }}</div>
      </button>
    </div>

    <div v-else class="space-y-4" :class="{ 'flex-1 min-h-0 overflow-y-auto': fill }">
      <div class="box p-3">
        <div class="flex items-center justify-between gap-2 mb-2">
          <div class="font-mono text-sm">{{ selectedTemplate.name }}</div>
          <Button type="button" variant="ghost" size="sm" @click="$emit('back')">
            {{ $t('globals.messages.back') }}
          </Button>
        </div>
        <div class="text-sm whitespace-pre-wrap text-muted-foreground">{{ renderedPreview }}</div>
      </div>

      <div v-for="ph in placeholders" :key="ph.key" class="grid grid-cols-3 gap-3 items-center">
        <label class="text-sm font-mono flex items-center gap-2">
          {{ placeholderLabel(ph.name) }}
          <span class="text-xs text-muted-foreground font-sans">{{ ph.partLabel }}</span>
        </label>
        <Input
          :model-value="templateParams[ph.key]"
          @update:model-value="(v) => $emit('update:param', ph.key, v)"
          class="col-span-2"
        />
      </div>

      <div v-for="btn in urlButtonParams" :key="btn.key" class="grid grid-cols-3 gap-3 items-center">
        <label class="text-sm truncate" :title="btn.url">{{ btn.label }}</label>
        <Input
          :model-value="templateParams[btn.key]"
          @update:model-value="(v) => $emit('update:param', btn.key, v)"
          class="col-span-2"
          :placeholder="btn.url"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Input } from '@shared-ui/components/ui/input'
import { placeholderLabel } from './whatsappTemplate.js'

defineProps({
  approvedTemplates: { type: Array, default: () => [] },
  selectedTemplate: { type: Object, default: null },
  templateParams: { type: Object, required: true },
  placeholders: { type: Array, default: () => [] },
  urlButtonParams: { type: Array, default: () => [] },
  renderedPreview: { type: String, default: '' },
  isFetching: { type: Boolean, default: false },
  fill: { type: Boolean, default: false }
})

defineEmits(['pick', 'back', 'update:param'])
</script>
