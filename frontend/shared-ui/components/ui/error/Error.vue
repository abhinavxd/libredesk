<script setup lang="ts">
import { TriangleAlert } from 'lucide-vue-next'
import { Alert, AlertDescription, AlertTitle } from '@shared-ui/components/ui/alert'

defineProps({
  errorMessage: String,
  border: Boolean
})
</script>

<template>
  <Alert variant="destructive" v-if="errorMessage" :class="{ 'border-0': !border }">
    <TriangleAlert class="w-4 h-4" aria-hidden="true" />
    <AlertTitle>{{ $t('globals.terms.error') }}</AlertTitle>
    <AlertDescription>
      {{ errorMessage }}
    </AlertDescription>
  </Alert>
</template>
