<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="space-y-4">
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
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
