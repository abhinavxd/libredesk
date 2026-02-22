<template>
    <div v-if="errorMessage" class="p-3 bg-destructive/10 border-t border-destructive/20 flex-shrink-0">
        <p class="text-sm text-destructive text-center">{{ errorMessage }}</p>
    </div>
</template>

<script setup>
defineProps({
    errorMessage: {
        type: String,
        default: ''
    }
})
</script>
