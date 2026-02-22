<template>
  <div v-if="dueAt" class="flex justify-start items-center space-x-2">
    <!-- Overdue-->
    <span v-if="sla?.status === 'overdue'" key="overdue" class="sla-badge sla-overdue">
      <AlertCircle size="12" class="shrink-0 text-red-600 dark:text-red-300" stroke-width="2" />
      <span class="sla-text">
        <span v-if="!showExtra">{{ label }} {{ $t('globals.terms.overdue') }}</span>
        <span v-else>{{ label }} {{ $t('sla.overdueBy') }} {{ sla.value }} </span>
      </span>
    </span>

    <!-- SLA Hit -->
    <span v-else-if="sla?.status === 'hit' && showExtra" key="sla-hit" class="sla-badge sla-hit">
      <CheckCircle size="12" class="shrink-0 text-green-600 dark:text-green-300" stroke-width="2" />
      <span class="sla-text">{{ label }} {{ $t('sla.met') }}</span>
    </span>

    <!-- Remaining -->
    <span v-else-if="sla?.status === 'remaining'" key="remaining" class="sla-badge sla-remaining">
      <Clock size="12" class="shrink-0 text-amber-600 dark:text-amber-300" stroke-width="2" />
      <span class="sla-text">{{ label }} {{ sla.value }}</span>
    </span>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useSla } from '../../composables/useSla'
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
let sla = useSla(ref(props.dueAt), ref(props.actualAt))

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
  @apply inline-flex items-center px-1.5 py-0.5 rounded border transition-all
         text-xs font-medium tracking-tight space-x-1 hover:shadow-sm;
}

.sla-overdue {
  @apply bg-red-50/80 border-red-100 text-red-600 
         dark:bg-red-900/40 dark:border-red-800/20 dark:text-red-300;
}

.sla-hit {
  @apply bg-green-50/80 border-green-100 text-green-600
         dark:bg-green-900/40 dark:border-green-800/20 dark:text-green-300;
}

.sla-remaining {
  @apply bg-amber-50/80 border-amber-100 text-amber-600
         dark:bg-amber-900/40 dark:border-amber-800/20 dark:text-amber-300;
}

.sla-text {
  @apply whitespace-nowrap;
}
</style>
