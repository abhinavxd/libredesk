<template>
  <DropdownMenu :modal="false">
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="h-6 w-6 p-0" @click.stop>
        <span class="sr-only">{{ i18n._('Open menu') }}</span>
        <MoreHorizontal class="h-3 w-3" />
      </Button>
    </DropdownMenuTrigger>

    <DropdownMenuContent align="end" class="w-48">
      <DropdownMenuItem @click="emit('edit', item)">
        <Pencil class="mr-2 h-4 w-4" />
        {{ i18n._('Edit') }}
      </DropdownMenuItem>

      <template v-if="item.type === 'collection'">
        <DropdownMenuSeparator />
        <DropdownMenuItem v-if="canCreateCollection" @click="emit('create-collection', item.id)">
          <FolderPlus class="mr-2 h-4 w-4" />
          {{ i18n._('New collection') }}
        </DropdownMenuItem>
        <DropdownMenuItem @click="emit('create-article', item)">
          <FilePlus class="mr-2 h-4 w-4" />
          {{ i18n._('New article') }}
        </DropdownMenuItem>
      </template>

      <template v-if="!(isFirst && isLast)">
        <DropdownMenuSeparator />

        <DropdownMenuItem v-if="!isFirst" @click="emit('move', { item, direction: -1 })">
          <ArrowUp class="mr-2 h-4 w-4" />
          {{ i18n._('Move up') }}
        </DropdownMenuItem>
        <DropdownMenuItem v-if="!isLast" @click="emit('move', { item, direction: 1 })">
          <ArrowDown class="mr-2 h-4 w-4" />
          {{ i18n._('Move down') }}
        </DropdownMenuItem>
      </template>

      <DropdownMenuSeparator />

      <DropdownMenuItem @click="emit('toggle-status', item)">
        <template v-if="item.type === 'collection'">
          <Eye v-if="!item.is_published" class="mr-2 h-4 w-4" />
          <EyeOff v-else class="mr-2 h-4 w-4" />
          {{ item.is_published ? i18n._('Unpublish') : i18n._('Publish') }}
        </template>
        <template v-else>
          <Eye v-if="item.status === 'draft'" class="mr-2 h-4 w-4" />
          <EyeOff v-else class="mr-2 h-4 w-4" />
          {{
            item.status === 'published'
              ? i18n._('Unpublish')
              : i18n._('Publish')
          }}
        </template>
      </DropdownMenuItem>

      <DropdownMenuSeparator />

      <DropdownMenuItem
        @click="emit('delete', item)"
        class="text-destructive focus:text-destructive"
      >
        <Trash class="mr-2 h-4 w-4" />
        {{ i18n._('Delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import {
  ArrowDown,
  ArrowUp,
  Eye,
  EyeOff,
  FilePlus,
  FolderPlus,
  MoreHorizontal,
  Pencil,
  Trash
} from 'lucide-vue-next'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()

defineProps({
  item: {
    type: Object,
    required: true
  },
  canCreateCollection: {
    type: Boolean,
    default: true
  },
  isFirst: {
    type: Boolean,
    default: false
  },
  isLast: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits([
  'create-collection',
  'create-article',
  'edit',
  'delete',
  'toggle-status',
  'move'
])
</script>
