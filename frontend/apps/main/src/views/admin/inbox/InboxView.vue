<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <!-- The livechat form writes its live config here so the preview replaces the help rail. -->
      <div v-if="livechatPreview" class="space-y-4 sticky top-2">
        <div class="space-y-2">
          <div class="flex items-center justify-between gap-2">
            <h4 class="text-sm font-medium text-foreground">
              {{ $t('admin.inbox.livechat.preview') }}
            </h4>
            <Tabs v-model="previewUserType">
              <TabsList class="h-8 p-0.5">
                <TabsTrigger value="visitors" class="text-xs">
                  {{ $t('admin.inbox.livechat.userSettings.visitors') }}
                </TabsTrigger>
                <TabsTrigger value="users" class="text-xs">
                  {{ $t('globals.terms.users') }}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
          <LivechatWidgetPreview :config="livechatPreview" :user-type="previewUserType" />
        </div>
        <div class="space-y-1">
          <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.help.livechat') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/livechat"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ $t('globals.terms.learnMore') }}
          </a>
        </div>
      </div>
      <div v-else class="space-y-4">
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ $t('globals.terms.email') }}</p>
          <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.help.email') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/connecting-inboxes"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ $t('globals.terms.learnMore') }}
          </a>
        </div>
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ $t('globals.terms.liveChat') }}</p>
          <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.help.livechat') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/livechat"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ $t('globals.terms.learnMore') }}
          </a>
        </div>
      </div>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, provide } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LivechatWidgetPreview from '@/features/admin/inbox/LivechatWidgetPreview.vue'
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'

const previewUserType = ref('visitors')
const livechatPreview = ref(null)
provide('livechatPreview', livechatPreview)
</script>
