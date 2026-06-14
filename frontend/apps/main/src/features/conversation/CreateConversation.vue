<template>
  <div>
    <Dialog v-model:open="dialogOpen">
      <DialogContent class="max-w-5xl w-full h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>
            {{ $t('conversation.newConversation') }}
          </DialogTitle>
          <DialogDescription />
        </DialogHeader>

        <Tabs v-if="showChannelTabs" v-model="channel" class="flex flex-col flex-1 overflow-hidden">
          <TabsList class="w-max">
            <TabsTrigger value="email">{{ $t('globals.terms.email') }}</TabsTrigger>
            <TabsTrigger value="whatsapp">{{ $t('globals.terms.whatsapp') }}</TabsTrigger>
          </TabsList>
          <div class="flex flex-col flex-1 overflow-hidden mt-4">
            <EmailConversationForm v-if="channel === 'email'" @close="dialogOpen = false" />
            <WhatsAppConversationForm v-else @close="dialogOpen = false" />
          </div>
        </Tabs>

        <div v-else class="flex flex-col flex-1 overflow-hidden">
          <EmailConversationForm @close="dialogOpen = false" />
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useInboxStore } from '@main/stores/inbox'
import EmailConversationForm from './EmailConversationForm.vue'
import WhatsAppConversationForm from './WhatsAppConversationForm.vue'

const dialogOpen = defineModel({
  required: false,
  default: () => false
})

const inboxStore = useInboxStore()
const channel = ref('email')

const showChannelTabs = computed(() => inboxStore.whatsappOptions.length > 0)

onMounted(() => inboxStore.fetchInboxes())
</script>
