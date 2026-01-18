<template>
  <div class="text-center" v-if="introText">
    <p class="text-muted-foreground text-sm">{{ introText }}</p>
  </div>
</template>

<script setup>
defineProps({
  introText: {
    type: String,
    default: ''
  }
})
</script>
