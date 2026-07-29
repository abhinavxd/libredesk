<template>
  <div class="flex flex-col min-h-[20rem] max-h-[32rem]" role="region" :aria-label="t('globals.terms.notification', 2)">
    <!-- Header -->
    <div class="flex items-center justify-between px-3 py-2.5 border-b border-border">
      <h3 class="font-semibold text-sm">{{ t('globals.terms.notification', 2) }}</h3>
      <div class="flex items-center gap-1">
        <Button
          v-if="notificationStore.unreadCount > 0"
          variant="ghost"
          size="sm"
          class="h-6 px-1.5"
          :title="t('globals.messages.markAllAsRead')"
          :aria-label="t('globals.messages.markAllAsRead')"
          @click="handleMarkAllAsRead"
        >
          <CheckCheck class="h-3.5 w-3.5" />
        </Button>
        <Button
          v-if="notificationStore.notifications.length > 0"
          variant="ghost"
          size="sm"
          class="h-6 px-1.5"
          :title="t('globals.messages.deleteAll')"
          :aria-label="t('globals.messages.deleteAll')"
          @click="handleDeleteAll"
        >
          <Trash2 class="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>

    <!-- Notification List -->
    <div class="flex-1 flex flex-col overflow-y-auto">
      <!-- Loading State -->
      <div v-if="notificationStore.isLoading && notificationStore.notifications.length === 0" class="divide-y divide-border">
        <div v-for="i in 4" :key="i" class="flex gap-2.5 px-3 py-2.5">
          <Skeleton class="h-7 w-7 rounded-full shrink-0" />
          <div class="flex-1 space-y-1.5">
            <Skeleton class="h-3" :class="i % 2 === 0 ? 'w-3/4' : 'w-4/5'" />
            <Skeleton class="h-3 w-1/2" />
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-else-if="notificationStore.notifications.length === 0"
        class="flex flex-col items-center justify-center flex-1 text-muted-foreground"
      >
        <BellOff class="h-7 w-7 mb-2" />
        <p class="text-xs">{{ t('toast.noNotificationsFound') }}</p>
      </div>

      <!-- Notifications -->
      <div v-else class="divide-y divide-border">
        <div
          v-for="notification in notificationStore.notifications"
          :key="notification.id"
          class="group relative px-3 py-2.5 hover:bg-muted/50 cursor-pointer transition-colors"
          @click="handleNotificationClick(notification)"
        >
          <div class="flex gap-2.5">
            <!-- Icon based on notification type -->
            <component
              :is="getNotificationStyle(notification.notification_type).icon"
              class="flex-shrink-0 h-4 w-4 mt-0.5"
              :class="getNotificationStyle(notification.notification_type).class"
            />

            <!-- Content -->
            <div class="flex-1 min-w-0">
              <p class="text-xs leading-snug" :class="notification.is_read ? 'text-muted-foreground' : 'font-medium text-foreground'">
                {{ notification.title }}
              </p>
              <p v-if="notification.body" class="text-xs leading-snug text-muted-foreground mt-0.5 line-clamp-2">
                {{ notification.body }}
              </p>
              <p class="text-xs text-muted-foreground/70 mt-0.5">
                {{ getRelativeTime(new Date(notification.created_at)) }}
              </p>
            </div>

            <!-- Action buttons (visible on hover) -->
            <div class="flex items-start gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
              <Button
                v-if="!notification.is_read"
                variant="ghost"
                size="sm"
                class="h-5 w-5 p-0"
                :aria-label="t('globals.messages.markAsRead')"
                @click.stop="handleMarkAsRead(notification)"
              >
                <Check class="h-2.5 w-2.5" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                class="h-5 w-5 p-0 hover:text-destructive"
                :aria-label="t('globals.messages.delete')"
                @click.stop="handleDelete(notification)"
              >
                <X class="h-2.5 w-2.5" />
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
          {{ notificationStore.isLoading ? t('globals.terms.loading') : t('globals.terms.loadMore') }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
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
  MessageSquare,
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


const notificationStyles = {
  mention: { icon: AtSign, class: 'text-primary' },
  assignment: { icon: UserPlus, class: 'text-accent-foreground' },
  new_reply: { icon: MessageSquare, class: 'text-primary' },
  new_reply_participating: { icon: MessageSquare, class: 'text-primary' }
}

const getNotificationStyle = (type) => {
  if (type?.startsWith('sla_')) {
    return {
      icon: type.endsWith('_breach') ? AlertCircle : AlertTriangle,
      class: 'text-destructive'
    }
  }
  return notificationStyles[type] || { icon: Bell, class: 'text-muted-foreground' }
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
