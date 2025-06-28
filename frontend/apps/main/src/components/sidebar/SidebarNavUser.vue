<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <SidebarMenuButton
        size="md"
        class="p-0"
      >
        <Avatar class="h-8 w-8 rounded relative overflow-visible">
          <AvatarImage :src="userStore.avatar" alt="U" class="rounded" />
          <AvatarFallback class="rounded">
            {{ userStore.getInitials }}
          </AvatarFallback>
          <div
            class="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border border-background"
            :class="{
              'bg-green-500': userStore.user.availability_status === 'online',
              'bg-amber-500':
                userStore.user.availability_status === 'away' ||
                userStore.user.availability_status === 'away_manual' ||
                userStore.user.availability_status === 'away_and_reassigning',
              'bg-gray-400': userStore.user.availability_status === 'offline'
            }"
          ></div>
        </Avatar>
        <div class="grid flex-1 text-left text-sm leading-tight">
          <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
          <span class="truncate text-xs">{{ userStore.email }}</span>
        </div>
        <ChevronsUpDown class="ml-auto size-4" />
      </SidebarMenuButton>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      class="w-[--radix-dropdown-menu-trigger-width] min-w-56"
      side="bottom"
      :side-offset="4"
    >
      <DropdownMenuLabel class="font-normal space-y-2 px-2">
        <!-- User header -->
        <div class="flex items-center gap-2 py-1.5 text-left text-sm">
          <Avatar class="h-8 w-8 rounded">
            <AvatarImage :src="userStore.avatar" alt="U" />
            <AvatarFallback class="rounded">
              {{ userStore.getInitials }}
            </AvatarFallback>
          </Avatar>
          <div class="flex-1 flex flex-col leading-tight">
            <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
            <span class="truncate text-xs text-muted-foreground">{{ userStore.email }}</span>
          </div>
        </div>

        <div class="space-y-2">
          <!-- Dark-mode toggle -->
          <div class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <Moon v-if="mode === 'dark'" size="16" class="text-muted-foreground" />
              <Sun v-else size="16" class="text-muted-foreground" />
              <span class="text-muted-foreground">{{ t('navigation.darkMode') }}</span>
            </div>
            <Switch
              :checked="mode === 'dark'"
              @update:checked="(val) => (mode = val ? 'dark' : 'light')"
            />
          </div>

          <div class="border-t border-gray-200 dark:border-gray-700 pt-3 space-y-3">
            <!-- Away toggle -->
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">{{ t('navigation.away') }}</span>
              <Switch
                :checked="
                  ['away_manual', 'away_and_reassigning'].includes(
                    userStore.user.availability_status
                  )
                "
                @update:checked="
                  (val) => userStore.updateUserAvailability(val ? 'away_manual' : 'online')
                "
              />
            </div>
            <!-- Reassign toggle -->
            <div class="flex items-center justify-between text-sm">
              <span class="text-muted-foreground">{{ t('navigation.reassignReplies') }}</span>
              <Switch
                :checked="userStore.user.availability_status === 'away_and_reassigning'"
                @update:checked="
                  (val) =>
                    userStore.updateUserAvailability(val ? 'away_and_reassigning' : 'away_manual')
                "
              />
            </div>
          </div>
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem @click.prevent="router.push({ name: 'account' })">
          <CircleUserRound size="18" class="mr-2" />
          {{ t('globals.terms.account') }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="logout">
        <LogOut size="18" class="mr-2" />
        {{ t('navigation.logout') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { SidebarMenuButton } from '@shared-ui/components/ui/sidebar'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Switch } from '@shared-ui/components/ui/switch'
import { ChevronsUpDown, CircleUserRound, LogOut, Moon, Sun } from 'lucide-vue-next'
import { useUserStore } from '../../stores/user'
import { useRouter } from 'vue-router'

import { useColorMode } from '@vueuse/core'

const mode = useColorMode()
const userStore = useUserStore()
const router = useRouter()
const { t } = useI18n()

const logout = () => {
  window.location.href = '/logout'
}
</script>
