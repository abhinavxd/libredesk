<template>
  <Card
    class="w-full cursor-pointer rounded-lg transition-colors hover:bg-accent"
    @click="emit('click', helpCenter)"
  >
    <CardHeader class="flex flex-row items-start gap-3 space-y-0">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
        <BookOpen class="h-5 w-5 text-muted-foreground" />
      </div>
      <div class="flex-1 min-w-0">
        <CardTitle class="text-base truncate">{{ helpCenter.name }}</CardTitle>
        <CardDescription class="truncate">{{ helpCenter.page_title }}</CardDescription>
      </div>
      <div @click.stop>
        <HelpCenterDropdown
          :help-center="helpCenter"
          @edit="emit('edit', $event)"
          @delete="emit('delete', $event)"
          @toggle="emit('toggle', $event)"
        />
      </div>
    </CardHeader>
    <CardContent class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 min-w-0">
        <Badge variant="secondary" class="font-normal truncate">/{{ helpCenter.slug }}</Badge>
        <Badge v-if="!helpCenter.is_active" variant="outline" class="font-normal shrink-0">
          {{ $t('helpCenter.paused') }}
        </Badge>
      </div>
      <span class="text-xs text-muted-foreground whitespace-nowrap">
        {{ $t('helpCenter.lastUpdated') }} {{ format(helpCenter.updated_at, 'PP') }}
      </span>
    </CardContent>
  </Card>
</template>

<script setup>
import { format } from 'date-fns'
import { BookOpen } from 'lucide-vue-next'
import { Badge } from '@shared-ui/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@shared-ui/components/ui/card'
import HelpCenterDropdown from './HelpCenterDropdown.vue'

defineProps({
  helpCenter: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['click', 'edit', 'delete'])
</script>
