<template>
  <div v-if="dueAt" class="flex justify-start items-center space-x-2">
    <span v-if="sla?.status === 'overdue'" key="overdue" :class="compact ? 'sla-compact text-destructive' : 'sla-badge sla-overdue'">
      <AlertCircle size="12" class="size-3 shrink-0 text-destructive" stroke-width="2" />
      <span class="sla-text">
        <span v-if="compact">{{ label ? `${label} ${sla.value}` : sla.value }}</span>
        <span v-else-if="!showExtra">{{ label }}</span>
        <span v-else>{{ label }} {{ i18n._('Overdue by') }} {{ sla.value }} </span>
      </span>
    </span>

    <Tooltip v-else-if="sla?.status === 'hit' && showExtra" key="sla-hit">
      <TooltipTrigger as-child>
        <span class="inline-flex items-center text-success rounded-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" tabindex="0">
          <CheckCircle size="14" aria-hidden="true" stroke-width="2" />
          <span class="sr-only">{{ label }} {{ i18n._('SLA met') }}</span>
        </span>
      </TooltipTrigger>
      <TooltipContent>{{ label }} {{ i18n._('SLA met') }}</TooltipContent>
    </Tooltip>

    <span v-else-if="sla?.status === 'remaining'" key="remaining" :class="compact ? 'sla-compact text-warning-600' : 'sla-badge sla-remaining'">
      <component :is="compact ? Timer : Clock" size="12" class="size-3 shrink-0 text-warning-600" stroke-width="2" />
      <span class="sla-text">{{ label ? `${label} ${sla.value}` : sla.value }}</span>
    </span>
  </div>
</template>

<script setup>
import { toRef, watch } from 'vue'
import { useSla } from '@main/composables/useSla'
import { AlertCircle, CheckCircle, Clock, Timer } from 'lucide-vue-next'
import { Tooltip, TooltipTrigger, TooltipContent } from '@shared-ui/components/ui/tooltip'
const props = defineProps({
  dueAt: String,
  actualAt: String,
  label: String,
  compact: { type: Boolean, default: false },
  showExtra: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['status'])
let sla = useSla(toRef(props, 'dueAt'), toRef(props, 'actualAt'))

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
  @apply inline-flex items-center px-1.5 py-0.5 rounded-md border transition-colors
         text-xs font-medium tracking-tight space-x-1 hover:shadow-sm;
}

.sla-compact {
  @apply inline-flex items-center gap-0.5 text-xs tabular-nums;
}

.sla-overdue {
  background-color: hsl(var(--destructive) / 0.1);
  border-color: hsl(var(--destructive) / 0.2);
  color: hsl(var(--destructive));
}

.sla-remaining {
  background-color: hsl(var(--warning) / 0.1);
  border-color: hsl(var(--warning) / 0.2);
  color: hsl(var(--warning));
}

.sla-text {
  @apply whitespace-nowrap;
}
</style>
