<template>
  <AdminSplitLayout :fixed-help="!!livechatPreview">
    <template #content>
      <router-view />
    </template>

    <template #help>
      <!-- The livechat form writes its live config here so the preview replaces the help rail. -->
      <div v-if="livechatPreview" class="space-y-4 sticky top-2">
        <div class="space-y-2">
          <div class="flex items-center justify-end gap-2">
            <div class="flex items-center gap-2">
              <Tabs v-model="previewUserType">
                <TabsList class="w-max justify-start h-8 p-0.5">
                  <TabsTrigger value="visitors" class="text-xs">
                    {{ i18n._('Visitors') }}
                  </TabsTrigger>
                  <TabsTrigger value="users" class="text-xs">
                    {{ i18n._('Users') }}
                  </TabsTrigger>
                </TabsList>
              </Tabs>
              <PreviewThemeToggle v-model="previewTheme" />
            </div>
          </div>
          <LivechatWidgetPreview :config="livechatPreview" :user-type="previewUserType" />
        </div>
        <div class="space-y-1">
          <p class="ui-description">{{ i18n._('Create live chat widgets that can be embedded on your website for real-time customer support.') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/livechat"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ i18n._('Learn more') }}
          </a>
        </div>
      </div>
      <div v-else class="space-y-4">
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ i18n.ngettext('Email', 'Emails', 1, {}, 'Communication channel') }}</p>
          <p class="ui-description">{{ i18n._('Connect your Google or Microsoft email account, or configure IMAP and SMTP settings manually. Each added inbox creates a new email channel for receiving customer emails.') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/connecting-inboxes"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ i18n._('Learn more') }}
          </a>
        </div>
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ i18n._('Live Chat') }}</p>
          <p class="ui-description">{{ i18n._('Create live chat widgets that can be embedded on your website for real-time customer support.') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/livechat"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ i18n._('Learn more') }}
          </a>
        </div>
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">{{ i18n._('WhatsApp') }}</p>
          <p class="ui-description">{{ i18n._('Connect a WhatsApp number through the Meta Cloud API to receive and reply to contact messages on WhatsApp.') }}</p>
          <a
            href="https://docs.libredesk.io/configuration/whatsapp"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ i18n._('Learn more') }}
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
import PreviewThemeToggle from '@/components/PreviewThemeToggle.vue'

const previewUserType = ref('visitors')
const previewTheme = ref('light')
const livechatPreview = ref(null)
provide('livechatPreview', livechatPreview)
provide('livechatPreviewTheme', previewTheme)
</script>
