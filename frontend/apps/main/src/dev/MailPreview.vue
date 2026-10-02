<template>
  <TooltipProvider>
    <div class="flex h-dvh flex-col bg-canvas text-foreground">
      <header
        class="flex flex-wrap items-center justify-between gap-3 border-b bg-background px-5 py-3"
      >
        <div class="flex items-center gap-3">
          <BrandLogo />
          <span class="rounded-md bg-success/10 px-2 py-1 text-xs font-medium text-success"
            >Mock mailbox</span
          >
        </div>
        <div class="flex items-center gap-2">
          <div class="lg:hidden"><BrowserNotifications /></div>
          <Button variant="outline" size="sm" @click="dark = !dark">{{
            dark ? 'Light mode' : 'Dark mode'
          }}</Button>
          <Button variant="outline" size="sm" @click="reset">Reset demo</Button>
          <Button size="sm" @click="receive"
            ><MailPlus class="mr-2 h-4 w-4" />Receive mock mail</Button
          >
        </div>
      </header>
      <div class="flex min-h-0 flex-1 gap-1.5 p-1.5">
        <aside class="hidden w-52 shrink-0 flex-col rounded-lg bg-background p-3 lg:flex">
          <div
            class="mb-6 px-2 pt-3 text-xs font-medium uppercase tracking-wider text-muted-foreground"
          >
            Addresses
          </div>
          <div class="flex items-center gap-3 rounded-md bg-accent px-3 py-2 text-sm font-medium">
            <Mail class="h-4 w-4" />contact@antimuonstudios.com<span class="ml-auto text-xs">{{ unreadCount }}</span>
          </div>
          <div class="mt-6 px-2 text-xs leading-relaxed text-muted-foreground">
            Green strips mark unread mail. Open a message to clear its strip, or right-click to mark
            it unread again.
          </div>
          <div class="mt-auto flex items-center justify-between border-t pt-3">
            <span class="text-sm font-medium">Demo account</span>
            <BrowserNotifications />
          </div>
        </aside>
        <section
          class="flex w-80 shrink-0 flex-col overflow-hidden rounded-lg bg-background max-md:w-64"
          aria-label="Mock inbox"
        >
          <div class="flex h-12 items-center justify-between border-b px-3">
            <h1 class="text-xl font-semibold">contact@antimuonstudios.com</h1>
            <span class="text-xs text-muted-foreground">{{ unreadCount }} unread</span>
          </div>
          <div
            class="flex items-center justify-between border-b px-3 py-3 text-xs text-muted-foreground"
          >
            <span>{{ store.conversations.data.length }} Open</span><span>Newest activity</span>
          </div>
          <div class="min-h-0 flex-1 divide-y overflow-y-auto">
            <ConversationListItem
              v-for="conversation in store.conversationsList"
              :key="conversation.uuid"
              :conversation="conversation"
              :current-conversation="store.current"
              :contact-full-name="store.getContactFullName(conversation.uuid)"
            />
          </div>
        </section>
        <main class="min-w-0 flex-1 overflow-hidden rounded-lg bg-background">
          <Conversation v-if="store.current.uuid" />
          <div v-else class="flex h-full items-center justify-center text-muted-foreground">
            Open a message to read it.
          </div>
        </main>
      </div>
      <div
        v-if="notificationPreview"
        class="fixed bottom-6 right-6 z-20 w-80 rounded-xl border bg-popover p-4 shadow-lg"
        aria-label="Notification preview"
      >
        <div class="mb-2 flex items-center justify-between text-xs text-muted-foreground">
          <span class="flex items-center gap-2"
            ><Bell class="h-3 w-3" />Notification preview · mocked</span
          >
          <Button
            variant="ghost"
            size="xs"
            aria-label="Dismiss preview"
            @click="notificationPreview = null"
            >×</Button
          >
        </div>
        <button type="button" class="w-full text-left" @click="openNotificationPreview">
          <strong class="text-sm">{{ notificationPreview.title }}</strong>
          <p class="mt-1 text-sm text-muted-foreground">{{ notificationPreview.body }}</p>
        </button>
      </div>
    </div>
  </TooltipProvider>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import BrandLogo from '@main/components/brand/BrandLogo.vue'
import { useRoute, useRouter } from 'vue-router'
import { Bell, Mail, MailPlus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import ConversationListItem from '@main/features/conversation/list/ConversationListItem.vue'
import Conversation from '@main/features/conversation/Conversation.vue'
import BrowserNotifications from '@main/components/sidebar/BrowserNotifications.vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { mailNotificationContent } from '@main/stores/browserNotifications'
import { WebSocketClient } from '@main/websocket'
import api from '@main/api'

const route = useRoute()
const router = useRouter()
const user = useUserStore()
user.setCurrentUser({
  id: 'mock-preview',
  first_name: 'Demo',
  last_name: 'Account',
  permissions: [],
  roles: []
})
const store = useConversationStore()
const client = new WebSocketClient()
// Feed the real event handler locally; no socket is opened and no mail is sent.
client.socket = {}
const mockedMessages = new Map()
api.getConversationMessage = async (_conversationUUID, uuid) => ({
  data: { data: mockedMessages.get(uuid) }
})
api.markConversationAsUnread = async () => ({})
api.updateAssigneeLastSeen = async () => ({})
api.getSidebarCounts = async () => ({
  data: {
    data: {
      all: store.conversations.data.length,
      unread: store.conversations.data.reduce(
        (sum, row) => sum + (row.unread_message_count || 0),
        0
      ),
      addresses: { 1: store.conversations.data.reduce((sum, row) => sum + (row.unread_message_count || 0), 0) }
    }
  }
})
api.getConversationTranscript = async () => ({
  data: new Blob(['Mock email transcript']),
  headers: {}
})
const dark = ref(true)
watch(dark, (value) => document.documentElement.classList.toggle('dark', value), {
  immediate: true
})
const unreadCount = computed(() =>
  store.conversations.data.reduce((sum, c) => sum + (c.unread_message_count || 0), 0)
)
const notificationPreview = ref(null)

function makeMail(uuid, firstName, lastName, subject, body, minutes, unread) {
  const author = {
    first_name: firstName,
    last_name: lastName,
    email: `${firstName.toLowerCase()}@example.test`
  }
  const createdAt = new Date(Date.now() - minutes * 60000).toISOString()
  const conversation = {
    uuid,
    correspondent: author,
    subject,
    status: 'Open',
    inbox_channel: 'email',
    inbox_name: 'Studio mail',
    inbox_mail: 'contact@antiumstudios.com',
    address_id: 1,
    last_message: body,
    last_message_at: createdAt,
    last_message_sender: 'contact',
    unread_message_count: unread
  }
  const message = {
    uuid: `${uuid}-message`,
    conversation_uuid: uuid,
    author,
    type: 'incoming',
    sender_type: 'contact',
    content_type: 'html',
    content: body,
    text_content: body,
    display: { html: `<p>${body}</p>` },
    created_at: createdAt,
    attachments: [],
    meta: { subject, from: [author.email], to: [conversation.inbox_mail] }
  }
  mockedMessages.set(message.uuid, message)
  store.messages.data.addMessages(uuid, [message], 1, 1)
  return conversation
}

function showConversation(uuid) {
  const row = store.conversations.data.find((c) => c.uuid === uuid)
  store.conversation.data = row ? { ...row } : null
  if (row) store.fetchMessages(uuid)
}

async function reset() {
  notificationPreview.value = null
  store.conversations.status = 'Open'
  store.conversations.listType = 'address'
  store.conversations.addressID = 1
  store.conversations.sortField = 'newest'
  store.conversations.data = [
    makeMail(
      'maya',
      'Maya',
      'Chen',
      'Press kit for launch week',
      'The updated screenshots are ready. Can you share the final launch date?',
      2,
      1
    ),
    makeMail(
      'james',
      'James',
      'Wilson',
      'A quick question about the playtest',
      'Hi! Is there room for two more testers this weekend?',
      8,
      2
    ),
    makeMail(
      'sofia',
      'Sofia',
      'Rossi',
      'Re: Trailer feedback',
      'The new cut looks great. I have attached a few notes for the final pass.',
      19,
      1
    ),
    makeMail(
      'albert',
      'Albert',
      'Aubachirov',
      'Steam request: Bering Tonnage',
      'Good day from KZ (Kazakhstan) Gaming Group o/\n\nWe are a community of enthusiastic videogamers and would love to try Bering Tonnage. Could you share a review key?\n\nHope all is well and have a great day~',
      35,
      0
    ),
    makeMail(
      'noah',
      'Noah',
      'Williams',
      'Thanks for the update',
      'Everything is working now. Thanks for your help!',
      62,
      0
    )
  ]
  const albert = mockedMessages.get('albert-message')
  albert.author.email = 'kzgamingcontact@gmail.com'
  albert.meta.from = [albert.author.email]
  albert.display.html =
    '<p>Good day from KZ (Kazakhstan) Gaming Group o/</p><p>We are a community of enthusiastic videogamers and would love to try Bering Tonnage. Could you share a review key?</p><p>Hope all is well and have a great day~</p>'
  store.messages.data.updateMessage('albert', 'albert-message', albert)
  store.messages.version++
  store.conversations.total = store.conversations.data.length
  store.sidebarCounts.unread = store.conversations.data.reduce(
    (sum, row) => sum + row.unread_message_count,
    0
  )
  store.sidebarCounts.addresses = { 1: store.sidebarCounts.unread }
  await router.push('/addresses/1/conversation/albert')
  showConversation('albert')
}

function receive() {
  const uuid = crypto.randomUUID()
  const conversation = makeMail(
    uuid,
    'Maya',
    'Chen',
    'Launch assets are ready',
    'Hi! The final screenshots and press kit are ready for your review.',
    0,
    0
  )
  const message = mockedMessages.get(`${uuid}-message`)
  const event = { ...message, sender: message.author, preview: message.text_content, conversation }
  client.handleMessage({
    target: client.socket,
    data: JSON.stringify({ type: 'new_message', data: event })
  })
  notificationPreview.value = { ...mailNotificationContent(event, 'New mail', 'New message'), uuid }
}

function openNotificationPreview() {
  router.push({
    name: 'address-inbox-conversation',
    params: { addressID: 1, uuid: notificationPreview.value.uuid }
  })
  notificationPreview.value = null
}

watch(() => route.params.uuid, showConversation)
reset()
</script>
