<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <!-- The livechat form writes its live config here so the preview replaces the help rail. -->
      <div v-if="livechatPreview" class="space-y-2 sticky top-2">
        <h4 class="text-sm font-medium text-foreground">{{ $t('admin.inbox.livechat.preview') }}</h4>
        <LivechatWidgetPreview :config="livechatPreview" />
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

const livechatPreview = ref(null)
provide('livechatPreview', livechatPreview)
</script>
