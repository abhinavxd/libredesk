<template>
  <div class="flex min-h-screen bg-canvas p-1.5 text-foreground">
    <aside
      class="hidden w-72 shrink-0 flex-col rounded-lg border border-sidebar-border bg-sidebar md:flex"
    >
      <div class="border-b border-sidebar-border px-5 py-5">
        <p class="text-xs font-semibold uppercase tracking-[0.16em] text-primary">{{ siteName }}</p>
        <h1 class="mt-1 text-xl font-semibold text-sidebar-foreground">Support</h1>
      </div>
      <div class="flex-1 p-3">
        <Button class="mb-4 w-full justify-start gap-2" @click="createOpen = true">
          <Plus class="size-4" />
          New ticket
        </Button>
        <div
          class="flex items-center gap-3 rounded-md bg-sidebar-accent px-3 py-2.5 text-sm font-medium text-sidebar-accent-foreground"
        >
          <Ticket class="size-4" />
          My tickets
          <span class="ml-auto text-xs text-muted-foreground">{{ tickets.length }}</span>
        </div>
      </div>
      <div class="border-t border-sidebar-border p-4">
        <p class="truncate text-sm font-medium text-sidebar-foreground">{{ currentUser?.email }}</p>
        <a
          class="mt-2 inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
          href="/portal/logout"
        >
          <LogOut class="size-4" />
          Sign out
        </a>
      </div>
    </aside>

    <main
      class="ml-1.5 flex min-w-0 flex-1 flex-col overflow-hidden rounded-lg border border-border bg-background"
    >
      <header
        class="flex min-h-16 items-center justify-between gap-4 border-b border-border px-5 md:px-7"
      >
        <div>
          <h2 class="text-lg font-semibold">My tickets</h2>
          <p class="text-sm text-muted-foreground">
            Track requests and responses from the support team.
          </p>
        </div>
        <div class="flex items-center gap-2">
          <Button class="gap-2 md:hidden" size="sm" @click="createOpen = true">
            <Plus class="size-4" />
            New ticket
          </Button>
          <Button as-child class="md:hidden" size="icon" variant="ghost">
            <a aria-label="Sign out" href="/portal/logout"><LogOut class="size-4" /></a>
          </Button>
        </div>
      </header>

      <div
        v-if="loading"
        class="grid flex-1 place-items-center text-sm text-muted-foreground"
        aria-live="polite"
      >
        Loading your tickets…
      </div>
      <div v-else-if="error" class="grid flex-1 place-items-center p-6 text-center">
        <div>
          <CircleAlert class="mx-auto mb-3 size-8 text-destructive" />
          <h3 class="font-semibold">Couldn’t load your tickets</h3>
          <Button class="mt-4" variant="outline" @click="loadPortal">Try again</Button>
        </div>
      </div>
      <div v-else-if="!tickets.length" class="grid flex-1 place-items-center p-6 text-center">
        <div class="max-w-sm">
          <div
            class="mx-auto mb-4 grid size-12 place-items-center rounded-full bg-primary/10 text-primary"
          >
            <Ticket class="size-6" />
          </div>
          <h3 class="text-lg font-semibold">No tickets yet</h3>
          <p class="mt-1 text-sm text-muted-foreground">
            Create a ticket and the support conversation will appear here.
          </p>
          <Button class="mt-5 gap-2" @click="createOpen = true">
            <Plus class="size-4" />
            Create your first ticket
          </Button>
        </div>
      </div>

      <div v-else class="grid min-h-0 flex-1 md:grid-cols-[22rem_1fr]">
        <nav
          class="max-h-72 overflow-auto border-b border-border md:max-h-none md:border-b-0 md:border-r"
          aria-label="Your tickets"
        >
          <button
            v-for="ticket in tickets"
            :key="ticket.uuid"
            type="button"
            class="grid w-full gap-2 border-b border-border px-5 py-4 text-left transition-colors hover:bg-muted/60"
            :class="{ 'bg-muted': selected?.uuid === ticket.uuid }"
            @click="selectTicket(ticket)"
          >
            <span class="flex items-center justify-between gap-3">
              <span class="text-xs font-medium text-primary">#{{ ticket.reference_number }}</span>
              <Badge variant="secondary">{{ ticket.status }}</Badge>
            </span>
            <strong class="truncate text-sm">{{ ticket.subject || 'Support request' }}</strong>
            <span class="truncate text-sm text-muted-foreground">{{
              ticket.last_message || 'No messages yet'
            }}</span>
            <time class="text-xs text-muted-foreground">{{
              formatDate(ticket.last_message_at || ticket.created_at)
            }}</time>
          </button>
        </nav>

        <article class="min-h-0 overflow-auto">
          <div
            v-if="selected"
            class="sticky top-0 z-10 flex items-center justify-between gap-4 border-b border-border bg-background/95 px-5 py-4 backdrop-blur md:px-7"
          >
            <div class="min-w-0">
              <p class="text-xs font-medium text-primary">#{{ selected.reference_number }}</p>
              <h3 class="truncate font-semibold">{{ selected.subject || 'Support request' }}</h3>
            </div>
            <Badge variant="secondary">{{ selected.status }}</Badge>
          </div>
          <div v-if="messagesLoading" class="p-7 text-sm text-muted-foreground">
            Loading conversation…
          </div>
          <div v-else class="grid gap-4 p-5 md:p-7">
            <div
              v-for="message in messages"
              :key="message.uuid"
              class="max-w-[85%] rounded-lg border border-border bg-muted/60 px-4 py-3"
              :class="{ 'ml-auto border-primary/20 bg-primary/10': message.type === 'incoming' }"
            >
              <div class="mb-2 flex items-center justify-between gap-4 text-xs">
                <strong>{{
                  message.author_name || (message.type === 'incoming' ? 'You' : 'Support')
                }}</strong>
                <time class="text-muted-foreground">{{ formatDate(message.created_at) }}</time>
              </div>
              <div class="whitespace-pre-wrap text-sm leading-6">
                {{ message.text_content || message.content }}
              </div>
            </div>
          </div>
        </article>
      </div>
    </main>

    <Dialog v-model:open="createOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Create a new ticket</DialogTitle>
          <DialogDescription
            >Tell us what you need help with. You can follow the response from this
            portal.</DialogDescription
          >
        </DialogHeader>
        <form class="grid gap-4 pt-2" @submit.prevent="createTicket">
          <div v-if="portalInboxes.length > 1" class="grid gap-2">
            <Label for="ticket-inbox">Send to</Label>
            <select
              id="ticket-inbox"
              v-model.number="newTicket.inbox_id"
              class="h-10 rounded-md border border-input bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-ring"
              required
            >
              <option v-for="inbox in portalInboxes" :key="inbox.id" :value="inbox.id">
                {{ inbox.name }}
              </option>
            </select>
          </div>
          <div class="grid gap-2">
            <Label for="ticket-subject">Subject</Label>
            <Input id="ticket-subject" v-model.trim="newTicket.subject" maxlength="200" required />
          </div>
          <div class="grid gap-2">
            <Label for="ticket-message">Message</Label>
            <Textarea
              id="ticket-message"
              v-model.trim="newTicket.content"
              class="min-h-36"
              maxlength="10000"
              required
            />
          </div>
          <p v-if="createError" class="text-sm text-destructive">{{ createError }}</p>
          <div class="flex justify-end gap-2">
            <Button type="button" variant="outline" @click="createOpen = false">Cancel</Button>
            <Button type="submit" :disabled="creating || !newTicket.subject || !newTicket.content">
              {{ creating ? 'Creating…' : 'Create ticket' }}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { Badge } from '@shared-ui/components/ui/badge'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { CircleAlert, LogOut, Plus, Ticket } from 'lucide-vue-next'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

const currentUser = ref(null)
const siteName = ref('Support portal')
const portalInboxes = ref([])
const tickets = ref([])
const selected = ref(null)
const messages = ref([])
const loading = ref(true)
const messagesLoading = ref(false)
const error = ref(false)
const createOpen = ref(false)
const creating = ref(false)
const createError = ref('')
const newTicket = reactive({ inbox_id: 0, subject: '', content: '' })

const formatDate = (value) =>
  value
    ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(
        new Date(value)
      )
    : ''

async function selectTicket(ticket) {
  selected.value = ticket
  messagesLoading.value = true
  try {
    const response = await api.getPortalMessages(ticket.uuid, { page_size: 100 })
    messages.value = response.data.data.results || []
  } finally {
    messagesLoading.value = false
  }
}

async function createTicket() {
  creating.value = true
  createError.value = ''
  try {
    const response = await api.createPortalConversation(newTicket)
    const ticket = response.data.data
    tickets.value.unshift(ticket)
    newTicket.subject = ''
    newTicket.content = ''
    createOpen.value = false
    await selectTicket(ticket)
  } catch (requestError) {
    createError.value = handleHTTPError(requestError).message
  } finally {
    creating.value = false
  }
}

async function loadPortal() {
  loading.value = true
  error.value = false
  try {
    const [meResponse, inboxesResponse, ticketsResponse, configResponse] = await Promise.all([
      api.getPortalMe(),
      api.getPortalInboxes(),
      api.getPortalConversations({ page_size: 100 }),
      api.getConfig()
    ])
    currentUser.value = meResponse.data.data
    siteName.value = configResponse.data.data?.['app.site_name'] || siteName.value
    portalInboxes.value = inboxesResponse.data.data || []
    newTicket.inbox_id = portalInboxes.value[0]?.id || 0
    tickets.value = ticketsResponse.data.data.results || []
    if (tickets.value.length) await selectTicket(tickets.value[0])
  } catch (requestError) {
    if (requestError.response?.status === 401 || requestError.response?.status === 403) {
      window.location.assign('/')
      return
    }
    error.value = true
  } finally {
    loading.value = false
  }
}

onMounted(loadPortal)
</script>
