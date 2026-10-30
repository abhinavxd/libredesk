<template>
  <div class="space-y-3">
    <p class="text-xs text-muted-foreground">
      <TranslatedText :message="toolApprovalDescriptionMessage" scope="global">
        <template #tool
          ><code
            class="inline-flex items-center rounded-md border bg-muted px-1.5 py-0.5 font-mono text-xs font-medium text-foreground"
            >{{ approval.tool_name }}</code
          ></template
        >
      </TranslatedText>
    </p>
    <div class="space-y-1">
      <p class="text-xs font-medium text-foreground">{{ i18n._('Arguments') }}</p>
      <pre
        class="max-h-60 overflow-auto whitespace-pre-wrap rounded-md bg-muted p-3 text-xs text-foreground [overflow-wrap:anywhere]"
        >{{ formattedArguments }}</pre
      >
    </div>
  </div>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

const toolApprovalDescriptionMessage = msg({ message: "The AI wants to run {tool}. Review the arguments before allowing it." })

import { computed } from 'vue'

const props = defineProps({
  approval: {
    type: Object,
    required: true
  }
})

const formattedArguments = computed(() => {
  try {
    return JSON.stringify(JSON.parse(props.approval.arguments), null, 2)
  } catch {
    return props.approval.arguments || ''
  }
})
</script>
