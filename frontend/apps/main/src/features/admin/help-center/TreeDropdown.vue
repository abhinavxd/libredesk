<template>
  <DropdownMenu :modal="false">
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        class="h-6 w-6 p-0"
        @click.stop
      >
        <MoreHorizontalIcon class="h-3 w-3" />
      </Button>
    </DropdownMenuTrigger>
    
    <DropdownMenuContent align="end" class="w-48">
      <DropdownMenuItem @click="handleEdit">
        <PencilIcon class="mr-2 h-4 w-4" />
        Edit {{ item.type === 'collection' ? 'Collection' : 'Article' }}
      </DropdownMenuItem>
      
      <template v-if="item.type === 'collection'">
        <DropdownMenuSeparator />
        <DropdownMenuItem @click="handleCreateCollection">
          <FolderPlusIcon class="mr-2 h-4 w-4" />
          Add Collection
        </DropdownMenuItem>
        <DropdownMenuItem @click="handleCreateArticle">
          <DocumentPlusIcon class="mr-2 h-4 w-4" />
          Add Article
        </DropdownMenuItem>
      </template>
      
      <DropdownMenuSeparator />
      
      <DropdownMenuItem @click="handleToggleStatus">
        <template v-if="item.type === 'collection'">
          <EyeIcon v-if="!item.is_published" class="mr-2 h-4 w-4" />
          <EyeSlashIcon v-else class="mr-2 h-4 w-4" />
          {{ item.is_published ? 'Unpublish' : 'Publish' }}
        </template>
        <template v-else>
          <EyeIcon v-if="item.status === 'draft'" class="mr-2 h-4 w-4" />
          <EyeSlashIcon v-else class="mr-2 h-4 w-4" />
          {{ item.status === 'published' ? 'Unpublish' : 'Publish' }}
        </template>
      </DropdownMenuItem>
      
      <DropdownMenuSeparator />
      
      <DropdownMenuItem 
        @click="handleDelete"
        class="text-destructive focus:text-destructive"
      >
        <TrashIcon class="mr-2 h-4 w-4" />
        Delete
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
  DropdownMenuTrigger,
} from '@shared-ui/components/ui/dropdown-menu'
import {
  FilePlus as DocumentPlusIcon,
  Eye as EyeIcon,
  EyeOff as EyeSlashIcon,
  FolderPlus as FolderPlusIcon,
  MoreHorizontal as MoreHorizontalIcon,
  Pencil as PencilIcon,
  Trash as TrashIcon,
} from 'lucide-vue-next'

const props = defineProps({
  item: {
    type: Object,
    required: true
  }
})

const emit = defineEmits([
  'create-collection',
  'create-article',
  'edit',
  'delete',
  'toggle-status'
])

const handleEdit = () => {
  emit('edit', props.item)
}

const handleCreateCollection = () => {
  emit('create-collection', props.item.id)
}

const handleCreateArticle = () => {
  emit('create-article', props.item)
}

const handleDelete = () => {
  emit('delete', props.item)
}

const handleToggleStatus = () => {
  emit('toggle-status', props.item)
}
</script>