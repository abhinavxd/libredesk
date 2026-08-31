<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <SidebarMenuButton
        v-if="!zendesk"
        size="default"
        class="p-0 !overflow-visible"
      >
        <div class="relative">
          <Avatar class="h-8 w-8 rounded-md">
            <AvatarImage :src="userStore.avatar" alt="U" class="rounded-md" />
            <AvatarFallback class="rounded-md">
              {{ userStore.getInitials }}
            </AvatarFallback>
          </Avatar>
          <StatusDot
            :status="userStore.user.availability_status"
            size="sm"
            class="absolute bottom-0 right-0 border border-background"
          />
        </div>
        <div class="grid flex-1 text-left text-sm leading-tight">
          <span class="truncate font-semibold">{{ userStore.getFullName }}</span>
          <span class="truncate text-xs">{{ userStore.email }}</span>
        </div>
        <ChevronsUpDown class="ml-auto size-4" />
      </SidebarMenuButton>
      <button
        v-else
        type="button"
        class="zendesk-nav-item w-full"
        :title="userStore.getFullName"
      >
        <div class="relative">
          <Avatar class="h-7 w-7 rounded">
            <AvatarImage :src="userStore.avatar" alt="U" class="rounded" />
            <AvatarFallback class="rounded text-xs">{{ userStore.getInitials }}</AvatarFallback>
          </Avatar>
          <StatusDot
            :status="userStore.user.availability_status"
            size="sm"
            class="absolute -bottom-0.5 -right-0.5 border border-[hsl(var(--zendesk-nav-bg))]"
          />
        </div>
      </button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      class="min-w-56"
      :side="zendesk ? menuSide : (isMobile ? 'bottom' : 'right')"
      :align="isMobile && !zendesk ? 'start' : 'end'"
      :side-offset="8"
      :align-offset="zendesk && menuSide === 'top' ? 8 : (isMobile ? 0 : 40)"
    >
      <DropdownMenuLabel class="font-normal space-y-2 px-2">
        <!-- User header -->
        <div class="flex items-center gap-2 py-1.5 text-left text-sm">
          <Avatar class="h-8 w-8 rounded-md">
            <AvatarImage :src="userStore.avatar" alt="U" />
            <AvatarFallback class="rounded-md">
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
            <span class="text-muted-foreground">{{ t('navigation.darkMode') }}</span>
            <Switch
              :checked="mode === 'dark'"
              @update:checked="(val) => (mode = val ? 'dark' : 'light')"
            />
          </div>

          <div class="flex items-center justify-between text-sm">
            <span class="text-muted-foreground">{{ t('navigation.interfaceLayout') }}</span>
            <Select :model-value="layout" @update:model-value="setLayout">
              <SelectTrigger class="h-8 w-[7.5rem]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="UI_LAYOUT_DEFAULT">{{ t('navigation.interfaceDefault') }}</SelectItem>
                <SelectItem :value="UI_LAYOUT_ZENDESK">{{ t('navigation.interfaceZendesk') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="border-t border-border pt-3 space-y-3">
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
      <DropdownMenuGroup>
        <DropdownMenuItem @click="showShortcuts = true">
          <Keyboard size="18" class="mr-2" />
          {{ t('navigation.keyboardShortcuts') }}
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <DropdownMenuItem @click="logout">
        <LogOut size="18" class="mr-2" />
        {{ t('navigation.logout') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <KeyboardShortcutsDialog v-model:open="showShortcuts" />
</template>

<script setup>
import { ref } from 'vue'
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
import StatusDot from '@shared-ui/components/StatusDot.vue'
import { Switch } from '@shared-ui/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { ChevronsUpDown, CircleUserRound, Keyboard, LogOut } from 'lucide-vue-next'
import { useUserStore } from '@main/stores/user'
import { useIsMobile } from '@shared-ui/composables'
import { useRouter } from 'vue-router'
import KeyboardShortcutsDialog from '@main/components/KeyboardShortcutsDialog.vue'
import { useColorMode } from '@vueuse/core'
import { useUiLayout, UI_LAYOUT_DEFAULT, UI_LAYOUT_ZENDESK } from '@main/composables/useUiLayout'

defineProps({
  zendesk: { type: Boolean, default: false },
  menuSide: { type: String, default: 'right' }
})

const isMobile = useIsMobile()
const mode = useColorMode()
const userStore = useUserStore()
const router = useRouter()
const { t } = useI18n()
const { layout, setLayout } = useUiLayout()
const showShortcuts = ref(false)

const logout = () => {
  window.location.href = '/logout'
}
</script>
