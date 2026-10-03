<template>
  <div class="flex min-w-0 items-center gap-1">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <SidebarMenuButton
          size="lg"
          class="min-w-0 flex-1 gap-2.5 px-1.5 data-[state=open]:bg-sidebar-accent"
        >
          <Avatar class="h-8 w-8 text-xs font-medium">
            <AvatarImage v-if="userStore.avatar" :src="userStore.avatar" alt="" />
            <AvatarFallback>{{ initials }}</AvatarFallback>
          </Avatar>
          <span class="min-w-0 flex-1 leading-tight">
            <span class="block truncate text-sm font-medium text-foreground">{{ accountName }}</span>
            <span v-if="userStore.email" class="block truncate text-xs text-muted-foreground">
              {{ userStore.email }}
            </span>
          </span>
          <ChevronsUpDown class="text-muted-foreground" aria-hidden="true" />
        </SidebarMenuButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" class="w-[--reka-dropdown-menu-trigger-width] min-w-52">
        <DropdownMenuItem
          v-if="userStore.can('general_settings:manage')"
          @click="router.push('/admin/general')"
        >
          <Settings class="mr-2 h-4 w-4" aria-hidden="true" />
          {{ t('navigation.settings') }}
        </DropdownMenuItem>
        <DropdownMenuItem @click="toggleColorMode">
          <Sun v-if="mode === 'dark'" class="mr-2 h-4 w-4" aria-hidden="true" />
          <Moon v-else class="mr-2 h-4 w-4" aria-hidden="true" />
          {{ mode === 'dark' ? t('navigation.useLightAppearance') : t('navigation.useDarkAppearance') }}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem class="text-destructive focus:text-destructive" @click="logout">
          <LogOut class="mr-2 h-4 w-4" aria-hidden="true" />
          {{ t('navigation.logout') }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>

    <BrowserNotifications button-class="h-8 w-8" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useColorMode } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { ChevronsUpDown, LogOut, Moon, Settings, Sun } from 'lucide-vue-next'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { SidebarMenuButton } from '@shared-ui/components/ui/sidebar'
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
const toggleColorMode = () => {
  mode.value = mode.value === 'dark' ? 'light' : 'dark'
}
</script>
