<template>
  <Button :variant="variant" :size="size" type="button" @click="handleCopy" :class="buttonClass">
    <Copy v-if="!copied" class="w-4 h-4" />
    <Check v-else class="w-4 h-4 text-green-500" />
    <span v-if="showText">
      {{ copied ? copiedText : copyText }}
    </span>
  </Button>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Copy, Check } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  textToCopy: {
    type: String,
    required: true
  },
  variant: {
    type: String,
    default: 'secondary'
  },
  size: {
    type: String,
    default: 'sm'
  },
  showText: {
    type: Boolean,
    default: true
  },
  copyText: {
    type: String,
    default: null
  },
  copiedText: {
    type: String,
    default: null
  },
  resetDelay: {
    type: Number,
    default: 2000
  },
  class: {
    type: String,
    default: ''
  }
})

const { t } = useI18n()
const copied = ref(false)

const buttonClass = computed(() => props.class)
const copyText = computed(() => props.copyText || t('globals.terms.copy'))
const copiedText = computed(() => props.copiedText || t('globals.terms.copied'))

const handleCopy = async () => {
  try {
    await navigator.clipboard.writeText(props.textToCopy)
    copied.value = true

    if (props.resetDelay > 0) {
      setTimeout(() => {
        copied.value = false
      }, props.resetDelay)
    }
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}
</script>
