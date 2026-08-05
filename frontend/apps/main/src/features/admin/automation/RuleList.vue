<template>
  <div class="flex flex-col box px-5 justify-center py-3">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-3">
        <router-link
          :to="{ name: 'edit-automation', params: { id: rule.id } }"
          class="text-base text-foreground font-medium hover:underline"
        >
          {{ rule.name }}
        </router-link>
        <Badge v-if="rule.enabled">{{ $t('globals.terms.enabled') }}</Badge>
        <Badge v-else variant="secondary">{{ $t('globals.terms.disabled') }}</Badge>
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button
            variant="ghost"
            size="sm"
            class="h-8 w-8 p-0"
            :aria-label="$t('globals.messages.options')"
          >
            <EllipsisVertical size="18" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="navigateToEditRule(rule.id)">
            <span>{{ $t('globals.messages.edit') }}</span>
          </DropdownMenuItem>
          <DropdownMenuItem @click="$emit('toggle-rule', rule.id)" v-if="rule.enabled">
            <span>{{ $t('globals.messages.disable') }}</span>
          </DropdownMenuItem>
          <DropdownMenuItem @click="$emit('toggle-rule', rule.id)" v-else>
            <span>{{ $t('globals.messages.enable') }}</span>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem class="text-destructive" @click="() => (alertOpen = true)">
            <span>{{ $t('globals.messages.delete') }}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
    <p class="text-sm-muted">{{ rule.description }}</p>
  </div>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('automation.deletionConfirmation')
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">{{
          $t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref } from 'vue'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
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
import { EllipsisVertical } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Badge } from '@shared-ui/components/ui/badge'
import { Button } from '@shared-ui/components/ui/button'

const router = useRouter()
const alertOpen = ref(false)
const emit = defineEmits(['delete-rule', 'toggle-rule'])

const props = defineProps({
  rule: {
    type: Object,
    required: true
  }
})

const navigateToEditRule = (id) => {
  router.push({ name: 'edit-automation', params: { id } })
}

const handleDelete = () => {
  emit('delete-rule', props.rule.id)
  alertOpen.value = false
}
</script>
