<template>
  <div class="p-4">
    <!-- Logo -->
    <img
      v-if="config.logo_url"
      :src="config.logo_url"
      :alt="config.brand_name"
      class="max-h-8 max-w-full"
    />
    <!-- Greeting and introduction -->
    <div class="mt-24 font-medium text-4xl">
      <h2>{{ config.greeting_message }}</h2>
      <p class="text-muted-foreground mt-2">
        {{ config.introduction_message }}
      </p>
    </div>
  </div>
</template>

<script setup>
defineProps({
  config: {
    type: Object,
    required: true
  }
})
</script>
