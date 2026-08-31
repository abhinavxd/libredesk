<template>
  <Popover v-model:open="isOpen">
    <PopoverTrigger as-child>
      <SidebarMenuButton size="default" class="relative">
        <Bell class="h-5 w-5" />
        <span
          v-if="notificationStore.unreadCount > 0"
          class="absolute top-0.5 left-6 md:left-auto md:right-0.5 inline-flex size-3.5 items-center justify-center rounded-full bg-destructive text-[9px] font-medium text-destructive-foreground"
        >
          {{ notificationStore.unreadCount > 99 ? '99' : notificationStore.unreadCount }}
        </span>
        <span v-if="isMobile">{{ t('globals.terms.notification', 2) }}</span>
      </SidebarMenuButton>
    </PopoverTrigger>
    <PopoverContent
      :side="isMobile ? 'bottom' : 'right'"
      :side-offset="8"
      :align="isMobile ? 'start' : 'end'"
      class="w-[min(24rem,calc(100vw-2rem))] p-0"
    >
      <NotificationPanel @close="isOpen = false" />
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bell } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { SidebarMenuButton } from '@shared-ui/components/ui/sidebar'
import { useNotificationStore } from '@main/stores/notification'
import { useIsMobile } from '@shared-ui/composables'
import NotificationPanel from './NotificationPanel.vue'

const notificationStore = useNotificationStore()
const isMobile = useIsMobile()
const { t } = useI18n()
const isOpen = ref(false)

onMounted(() => {
  notificationStore.fetchStats()
})

watch(isOpen, (open) => {
  if (open) {
    notificationStore.fetchNotifications()
  }
})
</script>
