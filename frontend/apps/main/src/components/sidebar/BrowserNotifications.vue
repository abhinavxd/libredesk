<template>
  <Popover @update:open="notifications.refreshPermission()">
    <PopoverTrigger as-child>
      <Button
        variant="ghost"
        size="icon"
        :class="[
          'shrink-0',
          buttonClass,
          notifications.active ? 'text-success hover:text-success' : 'text-muted-foreground'
        ]"
        :aria-label="t('mailNotifications.title')"
        :title="t('mailNotifications.title')"
      >
        <Bell v-if="notifications.active" class="h-4 w-4" aria-hidden="true" />
        <BellOff v-else class="h-4 w-4" aria-hidden="true" />
      </Button>
    </PopoverTrigger>
    <PopoverContent side="top" align="start" class="w-80 space-y-3">
      <p class="text-sm font-semibold">{{ t('mailNotifications.title') }}</p>
      <p class="text-sm text-muted-foreground">{{ t('mailNotifications.description') }}</p>
      <p v-if="!notifications.supported" role="status" class="text-sm text-muted-foreground">
        {{ t('mailNotifications.unsupported') }}
      </p>
      <p
        v-else-if="notifications.permission === 'denied'"
        role="status"
        class="text-sm text-muted-foreground"
      >
        {{ t('mailNotifications.blocked') }}
      </p>
      <p v-if="notifications.error" role="status" class="text-sm text-destructive">
        {{ t('mailNotifications.failed') }}
      </p>
      <Button
        v-if="notifications.enabled"
        variant="outline"
        size="sm"
        @click="notifications.disable()"
      >
        {{ t('mailNotifications.disable') }}
      </Button>
      <Button
        v-if="
          !notifications.active && notifications.supported && notifications.permission !== 'denied'
        "
        size="sm"
        :disabled="notifications.requesting"
        @click="notifications.enable()"
      >
        {{ t('mailNotifications.enable') }}
      </Button>
      <p v-if="notifications.active" role="status" class="text-xs text-success">
        {{ t('mailNotifications.enabled') }}
      </p>
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { Bell, BellOff } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useEventListener } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useBrowserNotificationsStore } from '@main/stores/browserNotifications'

defineProps({
  buttonClass: { type: String, default: '' }
})

const { t } = useI18n()
const notifications = useBrowserNotificationsStore()
useEventListener(window, 'focus', notifications.refreshPermission)
</script>
