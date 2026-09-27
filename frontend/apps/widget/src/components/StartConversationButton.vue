<template>
  <Button v-if="canStartNewConversation" type="button" @click="startNewConversation">
    {{ label }}
    <ArrowRight v-if="arrow" size="16" aria-hidden="true" />
  </Button>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowRight } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useStartConversation } from '@widget/composables/useStartConversation.js'

const props = defineProps({
  fallbackLabel: { type: String, default: 'globals.messages.startNewConversation' },
  arrow: { type: Boolean, default: false }
})

const { t } = useI18n()
const { configuredLabel, canStartNewConversation, startNewConversation } = useStartConversation()
const label = computed(() => configuredLabel.value || t(props.fallbackLabel))
</script>
