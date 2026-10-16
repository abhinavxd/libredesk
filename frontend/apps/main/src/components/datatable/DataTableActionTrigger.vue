<template>
  <DropdownMenuTrigger as-child>
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="size-8 max-md:size-11"
      :aria-label="$t('globals.messages.moreActions')"
    >
      <MoreVertical class="size-4" aria-hidden="true" />
    </Button>
  </DropdownMenuTrigger>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { DropdownMenuTrigger } from '@shared-ui/components/ui/dropdown-menu'
import { MoreVertical } from 'lucide-vue-next'
</script>
