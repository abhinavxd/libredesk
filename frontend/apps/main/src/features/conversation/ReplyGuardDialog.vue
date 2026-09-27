<template>
  <!--
    radix focuses AlertDialogCancel when the dialog opens, so "Don't send" holds
    the keyboard: Enter dismisses and Escape dismisses. Sending needs a
    deliberate click on "Send anyway".
  -->
  <AlertDialog v-model:open="open">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('replyGuard.title') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ $t('replyGuard.description') }}</AlertDialogDescription>
      </AlertDialogHeader>

      <div class="max-h-64 space-y-3 overflow-y-auto">
        <div v-for="group in groups" :key="group.rule">
          <p class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {{ $t(group.label) }}
          </p>
          <ul class="mt-1.5 flex flex-wrap gap-1.5">
            <li
              v-for="excerpt in group.excerpts"
              :key="excerpt"
              class="max-w-full break-words rounded border bg-muted px-2 py-0.5 text-sm"
            >
              {{ excerpt }}
            </li>
          </ul>
        </div>
      </div>

      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('replyGuard.dontSend') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="emit('confirm')">
          {{ $t('replyBox.sendAnyway') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed } from 'vue'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { groupReplyGuardMatches } from '@/features/conversation/replyGuard'

const open = defineModel('open', { type: Boolean, default: false })

const props = defineProps({
  matches: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['confirm'])

const groups = computed(() => groupReplyGuardMatches(props.matches))
</script>
