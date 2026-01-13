<template>
  <Button
    variant="ghost"
    size="icon"
    class="h-7 w-7 text-zinc-400 hover:text-white hover:bg-zinc-700"
    @click="copy"
  >
    <Check v-if="copied" class="h-4 w-4 text-green-500" />
    <Copy v-else class="h-4 w-4" />
  </Button>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Copy, Check } from 'lucide-vue-next'

const props = defineProps({
  text: { type: String, required: true }
})

const copied = ref(false)

const copy = async () => {
  await navigator.clipboard.writeText(props.text)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 1500)
}
</script>
