<template>
  <div class="relative group w-28 h-28 cursor-pointer" @click="triggerFileInput">
    <Avatar class="size-28">
      <AvatarImage :src="src || ''" />
      <AvatarFallback>{{ initials }}</AvatarFallback>
    </Avatar>

    <!-- Hover Overlay -->
    <div
      class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer rounded-full"
    >
      <span class="text-white font-semibold">{{ label }}</span>
    </div>

    <!-- Delete Icon -->
    <X
      class="absolute top-1 right-1 rounded-full p-1 shadow-md z-10 opacity-0 group-hover:opacity-100 transition-opacity"
      size="20"
      @click.stop="emit('remove')"
      v-if="src"
    />

    <!-- File Input -->
    <input
      ref="fileInput"
      type="file"
      class="hidden"
      accept="image/png,image/jpeg,image/jpg"
      @change="handleChange"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Avatar, AvatarImage, AvatarFallback } from '.'
import { X } from 'lucide-vue-next'

defineProps({
  src: String,
  initials: String,
  label: {
    type: String,
    default: 'Upload'
  }
})

const emit = defineEmits(['upload', 'remove'])
const fileInput = ref(null)

function triggerFileInput() {
  fileInput.value?.click()
}

function handleChange(e) {
  const file = e.target.files[0]
  if (file) emit('upload', file)
}
</script>
