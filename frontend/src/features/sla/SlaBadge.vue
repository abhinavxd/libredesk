<template>
  <div v-if="dueAt" class="flex justify-start items-center space-x-2">
    <!-- Overdue-->
    <span v-if="sla?.status === 'overdue'" key="overdue" class="sla-badge box sla-overdue">
      <AlertCircle size="12" class="text-red-800" />
      <span class="sla-text text-red-800">
        <span v-if="!showExtra">{{ label }} {{ $t('sla.overdue') }}</span>
        <span v-else>{{ label }} {{ $t('sla.overdueBy') }} {{ sla.value }} </span>
      </span>
    </span>

    <!-- SLA Hit -->
    <span
      v-else-if="sla?.status === 'hit' && showExtra"
      key="sla-hit"
      class="sla-badge box sla-hit"
    >
      <CheckCircle size="12" />
      <span class="sla-text">{{ label }} {{ $t('sla.met') }}</span>
    </span>

    <!-- Remaining -->
    <span
      v-else-if="sla?.status === 'remaining'"
      key="remaining"
      class="sla-badge box sla-remaining"
    >
      <Clock size="12" />
      <span class="sla-text">{{ label }} {{ sla.value }}</span>
    </span>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useSla } from '@/composables/useSla'
import { AlertCircle, CheckCircle, Clock } from 'lucide-vue-next'
const props = defineProps({
  dueAt: String,
  actualAt: String,
  label: String,
  showExtra: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['status'])

let sla = null
if (props.dueAt) {
  sla = useSla(ref(props.dueAt), ref(props.actualAt))
}

// Watch for status change and emit
watch(
  sla,
  (newVal) => {
    if (newVal?.status) emit('status', newVal.status)
  },
  { immediate: true }
)
</script>

<style scoped>
.sla-badge {
  @apply inline-flex items-center justify-center p-1 text-xs space-x-1 w-full rounded-lg;
}

.sla-overdue {
  @apply bg-red-100 text-red-800;
}

.sla-hit {
  @apply bg-green-100 text-green-800;
}

.sla-remaining {
  @apply bg-yellow-100 text-yellow-800;
}

.sla-text {
  @apply text-[0.65rem];
}
</style>
