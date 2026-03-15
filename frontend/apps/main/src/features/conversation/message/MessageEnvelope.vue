<template>
  <div class="mb-1 space-y-0.5 text-xs text-muted-foreground">
    <p v-if="message.meta.from?.length">
      <span class="font-medium text-foreground">From:</span> {{ message.meta.from.join(', ') }}
    </p>
    <p v-if="message.meta.to?.length">
      <span class="font-medium text-foreground">To:</span> {{ message.meta.to.join(', ') }}
    </p>
    <p v-if="message.meta.cc?.length">
      <span class="font-medium text-foreground">CC:</span> {{ message.meta.cc.join(', ') }}
    </p>
    <p v-if="message.meta.bcc?.length">
      <span class="font-medium text-foreground">BCC:</span> {{ message.meta.bcc.join(', ') }}
    </p>
    <p v-if="message.meta.subject">
      <span class="font-medium text-foreground">Subject:</span> {{ message.meta.subject }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  message: {
    type: Object,
    required: true
  }
})
</script>
