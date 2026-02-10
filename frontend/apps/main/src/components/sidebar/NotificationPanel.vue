<template>
  <div class="flex flex-col max-h-[32rem]">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b">
      <h3 class="font-semibold text-sm">{{ t('globals.terms.notification', 2) }}</h3>
      <div class="flex items-center gap-2">
        <Button
          v-if="notificationStore.unreadCount > 0"
          variant="ghost"
          size="sm"
          class="h-7 px-2"
          :title="t('globals.messages.markAllAsRead')"
          @click="handleMarkAllAsRead"
        >
          <CheckCheck class="h-3.5 w-3.5" />
        </Button>
        <Button
          v-if="notificationStore.notifications.length > 0"
          variant="ghost"
          size="sm"
          class="h-7 px-2"
          :title="t('globals.messages.deleteAll')"
          @click="handleDeleteAll"
        >
          <Trash2 class="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>

    <!-- Notification List -->
    <div class="flex-1 overflow-y-auto">
      <!-- Loading State -->
      <div v-if="notificationStore.isLoading && notificationStore.notifications.length === 0" class="p-4">
        <div class="space-y-3">
          <div v-for="i in 3" :key="i" class="flex gap-3">
            <Skeleton class="h-8 w-8 rounded-full" />
            <div class="flex-1 space-y-2">
              <Skeleton class="h-3 w-3/4" />
              <Skeleton class="h-3 w-1/2" />
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-else-if="notificationStore.notifications.length === 0"
        class="flex flex-col items-center justify-center py-8 text-muted-foreground"
      >
        <BellOff class="h-8 w-8 mb-2" />
        <p class="text-sm">{{ t('globals.messages.noResults', { name: t('globals.terms.notification', 2) }) }}</p>
      </div>

      <!-- Notifications -->
      <div v-else class="divide-y">
        <div
          v-for="notification in notificationStore.notifications"
          :key="notification.id"
          class="group relative px-4 py-3 hover:bg-muted/50 cursor-pointer transition-colors"
          :class="{ 'opacity-60': notification.is_read }"
          @click="handleNotificationClick(notification)"
        >
          <div class="flex gap-3">
            <!-- Icon based on notification type -->
            <div
              class="flex-shrink-0 h-8 w-8 rounded-full flex items-center justify-center"
              :class="getNotificationIconClass(notification.notification_type)"
            >
              <component :is="getNotificationIcon(notification.notification_type)" class="h-4 w-4" />
            </div>

            <!-- Content -->
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium" :class="{ 'font-semibold': !notification.is_read }">
                {{ notification.title }}
              </p>
              <p v-if="notification.body" class="text-xs text-muted-foreground mt-0.5">
                {{ notification.body }}
              </p>
              <p class="text-xs text-muted-foreground mt-1">
                {{ getRelativeTime(new Date(notification.created_at)) }}
              </p>
            </div>

            <!-- Action buttons (visible on hover) -->
            <div class="flex items-start gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <Button
                v-if="!notification.is_read"
                variant="ghost"
                size="sm"
                class="h-6 w-6 p-0"
                @click.stop="handleMarkAsRead(notification)"
              >
                <Check class="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                class="h-6 w-6 p-0 hover:text-destructive"
                @click.stop="handleDelete(notification)"
              >
                <X class="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <!-- Load More -->
      <div v-if="notificationStore.hasMore && notificationStore.notifications.length > 0" class="p-2">
        <Button
          variant="ghost"
          size="sm"
          class="w-full"
          :disabled="notificationStore.isLoading"
          @click="notificationStore.loadMore"
        >
          {{ notificationStore.isLoading ? t('globals.messages.loading') : t('globals.terms.loadMore') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Bell,
  BellOff,
  Check,
  CheckCheck,
  X,
  Trash2,
  AtSign,
  UserPlus,
  AlertTriangle,
  AlertCircle
} from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Skeleton } from '@shared-ui/components/ui/skeleton'
import { useNotificationStore } from '@main/stores/notification'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'

const emit = defineEmits(['close'])

const router = useRouter()
const { t } = useI18n()
const notificationStore = useNotificationStore()

onMounted(() => {
  notificationStore.fetchNotifications()
})

const getNotificationIcon = (type) => {
  const icons = {
    mention: AtSign,
    assignment: UserPlus,
    sla_warning: AlertTriangle,
    sla_breach: AlertCircle
  }
  return icons[type] || Bell
}

const getNotificationIconClass = (type) => {
  const classes = {
    mention: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400',
    assignment: 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400',
    sla_warning: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400',
    sla_breach: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400'
  }
  return classes[type] || 'bg-muted text-muted-foreground'
}

const handleNotificationClick = async (notification) => {
  // Mark as read if unread
  if (!notification.is_read) {
    await notificationStore.markAsRead(notification.id)
  }

  // Navigate to conversation if available
  if (notification.conversation_uuid) {
    emit('close')
    router.push({
      name: 'inbox-conversation',
      params: {
        type: notification.notification_type === 'mention' ? 'mentioned' : 'assigned',
        uuid: notification.conversation_uuid
      },
      query: notification.message_uuid ? { scrollTo: notification.message_uuid } : {}
    })
  }
}

const handleMarkAsRead = async (notification) => {
  await notificationStore.markAsRead(notification.id)
}

const handleMarkAllAsRead = async () => {
  await notificationStore.markAllAsRead()
}

const handleDelete = async (notification) => {
  await notificationStore.deleteNotification(notification.id)
}

const handleDeleteAll = async () => {
  await notificationStore.deleteAll()
}
</script>
