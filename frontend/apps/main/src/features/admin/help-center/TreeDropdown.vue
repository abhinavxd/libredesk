<template>
  <DropdownMenu :modal="false">
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="h-6 w-6 p-0" @click.stop>
        <span class="sr-only">{{ t('globals.terms.openMenu') }}</span>
        <MoreHorizontal class="h-3 w-3" />
      </Button>
    </DropdownMenuTrigger>

    <DropdownMenuContent align="end" class="w-48">
      <DropdownMenuItem @click="emit('edit', item)">
        <Pencil class="mr-2 h-4 w-4" />
        {{ t('globals.messages.edit') }}
      </DropdownMenuItem>

      <template v-if="item.type === 'collection'">
        <DropdownMenuSeparator />
        <DropdownMenuItem v-if="canCreateCollection" @click="emit('create-collection', item.id)">
          <FolderPlus class="mr-2 h-4 w-4" />
          {{ t('helpCenter.newCollection') }}
        </DropdownMenuItem>
        <DropdownMenuItem @click="emit('create-article', item)">
          <FilePlus class="mr-2 h-4 w-4" />
          {{ t('helpCenter.newArticle') }}
        </DropdownMenuItem>
      </template>

      <template v-if="!(isFirst && isLast)">
        <DropdownMenuSeparator />

        <DropdownMenuItem v-if="!isFirst" @click="emit('move', { item, direction: -1 })">
          <ArrowUp class="mr-2 h-4 w-4" />
          {{ t('globals.messages.moveUp') }}
        </DropdownMenuItem>
        <DropdownMenuItem v-if="!isLast" @click="emit('move', { item, direction: 1 })">
          <ArrowDown class="mr-2 h-4 w-4" />
          {{ t('globals.messages.moveDown') }}
        </DropdownMenuItem>
      </template>

      <DropdownMenuSeparator />

      <DropdownMenuItem @click="emit('toggle-status', item)">
        <template v-if="item.type === 'collection'">
          <Eye v-if="!item.is_published" class="mr-2 h-4 w-4" />
          <EyeOff v-else class="mr-2 h-4 w-4" />
          {{ item.is_published ? t('globals.messages.unpublish') : t('globals.messages.publish') }}
        </template>
        <template v-else>
          <Eye v-if="item.status === 'draft'" class="mr-2 h-4 w-4" />
          <EyeOff v-else class="mr-2 h-4 w-4" />
          {{
            item.status === 'published'
              ? t('globals.messages.unpublish')
              : t('globals.messages.publish')
          }}
        </template>
      </DropdownMenuItem>

      <DropdownMenuSeparator />

      <DropdownMenuItem
        @click="emit('delete', item)"
        class="text-destructive focus:text-destructive"
      >
        <Trash class="mr-2 h-4 w-4" />
        {{ t('globals.messages.delete') }}
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
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

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
