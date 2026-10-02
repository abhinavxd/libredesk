<template>
  <div class="rounded-lg border border-sidebar-border bg-sidebar-accent/30 p-2">
    <div class="flex min-w-0 items-center gap-2">
      <div
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-success/15 text-sm font-semibold text-success"
        aria-hidden="true"
      >
        {{ initials }}
      </div>
      <div class="min-w-0 flex-1">
        <span class="block truncate text-sm font-semibold text-foreground" :title="accountName">
          {{ accountName }}
        </span>
        <span
          v-if="userStore.email"
          class="block truncate text-xs text-muted-foreground"
          :title="userStore.email"
        >
          {{ userStore.email }}
        </span>
      </div>

      <BrowserNotifications
        button-class="h-8 w-8 rounded-md bg-sidebar-accent/50 hover:bg-sidebar-accent"
      />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 shrink-0 rounded-md bg-sidebar-accent/50 text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
            aria-label="Account options"
            title="Account options"
          >
            <MoreHorizontal class="h-4 w-4" aria-hidden="true" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="top" align="end" class="w-52">
          <DropdownMenuItem
            v-if="userStore.can('general_settings:manage')"
            @click="router.push('/admin/general')"
          >
            <Settings class="mr-2 h-4 w-4" aria-hidden="true" />
            Settings
          </DropdownMenuItem>
          <DropdownMenuItem @click="toggleColorMode">
            <Sun v-if="mode === 'dark'" class="mr-2 h-4 w-4" aria-hidden="true" />
            <Moon v-else class="mr-2 h-4 w-4" aria-hidden="true" />
            {{ appearanceLabel }}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem class="text-destructive focus:text-destructive" @click="logout">
            <LogOut class="mr-2 h-4 w-4" aria-hidden="true" />
            {{ t('navigation.logout') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useColorMode } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { Settings, LogOut, Sun, Moon, MoreHorizontal } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { useUserStore } from '@main/stores/user'
import { useLogout } from '@main/composables/useLogout'
import BrowserNotifications from './BrowserNotifications.vue'

const mode = useColorMode()
const userStore = useUserStore()
const router = useRouter()
const { t } = useI18n()
const logout = useLogout()
const accountName = computed(() => userStore.getFullName || userStore.email || 'Account')
const initials = computed(
  () => userStore.getInitials || accountName.value.slice(0, 1).toUpperCase()
)
const appearanceLabel = computed(() =>
  mode.value === 'dark' ? 'Use light appearance' : 'Use dark appearance'
)
const toggleColorMode = () => {
  mode.value = mode.value === 'dark' ? 'light' : 'dark'
}
</script>
