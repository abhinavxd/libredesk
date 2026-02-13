<template>
  <Popover v-model:open="isOpen">
    <PopoverTrigger as-child>
      <SidebarMenuButton size="md" class="relative" @click="handleOpen">
        <Bell class="h-5 w-5" />
        <span
          v-if="notificationStore.unreadCount > 0"
          class="absolute top-0.5 right-0.5 inline-flex size-3.5 items-center justify-center rounded-full bg-destructive text-[9px] font-medium text-destructive-foreground"
        >
          {{ notificationStore.unreadCount > 99 ? '99' : notificationStore.unreadCount }}
        </span>
      </SidebarMenuButton>
    </PopoverTrigger>
    <PopoverContent side="right" :side-offset="8" align="end" class="w-96 p-0">
      <NotificationPanel @close="isOpen = false" />
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Bell } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { SidebarMenuButton } from '@shared-ui/components/ui/sidebar'
import { useNotificationStore } from '@main/stores/notification'
import NotificationPanel from './NotificationPanel.vue'

const notificationStore = useNotificationStore()
const isOpen = ref(false)

onMounted(() => {
  notificationStore.fetchStats()
})

const handleOpen = () => {
  if (!isOpen.value) {
    notificationStore.fetchNotifications()
  }
}
</script>
