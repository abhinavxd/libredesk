<template>
  <div
    ref="editorRef"
    class="my-2 min-h-0 shrink overflow-y-auto rounded-md border border-border bg-muted/30 p-3 max-h-60 space-y-3"
  >
    <div class="space-y-1">
      <p class="text-sm font-medium">{{ $t('globals.terms.button', 2) }}</p>
      <p class="text-xs text-muted-foreground">
        {{ $t('conversation.telegram.buttons.description') }}
      </p>
    </div>
    <div v-for="(button, index) in buttons" :key="index" class="flex items-start gap-2">
      <div
        class="grid min-w-0 flex-1 grid-cols-[repeat(auto-fit,minmax(min(100%,12rem),1fr))] gap-2"
      >
        <div :class="FIELD_CLASS">
          <Label :for="`${id}-text-${index}`">{{ $t('globals.terms.buttonText') }}</Label>
          <Input
            :ref="(el) => (textInputRefs[index] = el)"
            :id="`${id}-text-${index}`"
            :model-value="button.text"
            :aria-invalid="Boolean(errors[index]?.text)"
            :aria-describedby="errors[index]?.text ? `${id}-text-error-${index}` : undefined"
            :class="{ 'border-destructive': errors[index]?.text }"
            @update:model-value="update(index, 'text', $event)"
          />
          <p
            v-if="errors[index]?.text"
            :id="`${id}-text-error-${index}`"
            role="alert"
            class="text-xs text-destructive"
          >
            {{
              errors[index].text === 'required'
                ? $t('globals.messages.required', { name: $t('globals.terms.buttonText') })
                : $t('conversation.telegram.error.buttonTextTooLong')
            }}
          </p>
        </div>
        <div :class="FIELD_CLASS">
          <Label :for="`${id}-url-${index}`">{{ $t('globals.messages.optionalLink') }}</Label>
          <Input
            :id="`${id}-url-${index}`"
            :model-value="button.url"
            placeholder="https://"
            inputmode="url"
            autocapitalize="none"
            :spellcheck="false"
            :aria-invalid="Boolean(errors[index]?.url)"
            :aria-describedby="errors[index]?.url ? `${id}-url-error-${index}` : undefined"
            :class="{ 'border-destructive': errors[index]?.url }"
            @update:model-value="update(index, 'url', $event)"
          />
          <p
            v-if="errors[index]?.url"
            :id="`${id}-url-error-${index}`"
            role="alert"
            class="text-xs text-destructive"
          >
            {{ $t('conversation.telegram.error.buttonLink') }}
          </p>
        </div>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="mt-6 shrink-0 text-muted-foreground hover:text-foreground max-md:size-11"
        :aria-label="$t('globals.terms.remove')"
        @click="remove(index)"
        ><X class="size-4" aria-hidden="true"
      /></Button>
    </div>
    <p v-if="validated && buttons.length > 10" role="alert" class="text-sm text-destructive">
      {{ $t('conversation.telegram.error.buttons') }}
    </p>
  </div>
</template>
<script setup>
const FIELD_CLASS = 'min-w-0 space-y-1'

import { computed, nextTick, ref, useId } from 'vue'
import { X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { telegramButtonFieldErrors } from './telegramReply'
const id = useId()
const editorRef = ref(null)
const textInputRefs = ref([])
const buttons = defineModel({ type: Array, default: () => [] })
const props = defineProps({ validated: Boolean })
const emit = defineEmits(['empty'])
const errors = computed(() => (props.validated ? buttons.value.map(telegramButtonFieldErrors) : []))
const update = (index, key, value) => {
  buttons.value = buttons.value.map((button, i) =>
    i === index ? { ...button, [key]: value } : button
  )
}
const remove = async (index) => {
  const remaining = buttons.value.filter((_, i) => i !== index)
  buttons.value = remaining
  if (!remaining.length) {
    emit('empty')
    return
  }
  await nextTick()
  focus(Math.min(index, buttons.value.length - 1))
}
const focus = (index = buttons.value.length - 1) => textInputRefs.value[index]?.$el?.focus()
const focusError = () => editorRef.value?.querySelector('[aria-invalid="true"]')?.focus()
defineExpose({ focus, focusError })
</script>
