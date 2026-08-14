<template>
  <main class="portal-shell">
    <header class="portal-header">
      <div>
        <p class="eyebrow">{{ siteName }}</p>
        <h1>Your support tickets</h1>
        <p v-if="currentUser" class="welcome">Signed in as {{ currentUser.email }}</p>
      </div>
      <a class="sign-out" href="/portal/logout">Sign out</a>
    </header>

    <div v-if="loading" class="state-card" aria-live="polite">Loading your tickets…</div>
    <div v-else-if="error" class="state-card error-card">
      <strong>We couldn’t load your tickets.</strong>
      <button type="button" @click="loadPortal">Try again</button>
    </div>
    <div v-else-if="!tickets.length" class="state-card">
      <span class="empty-mark">✓</span>
      <h2>No tickets yet</h2>
      <p>Messages sent from this email address will appear here.</p>
    </div>

    <section v-else class="ticket-grid">
      <nav class="ticket-list" aria-label="Your tickets">
        <button
          v-for="ticket in tickets"
          :key="ticket.uuid"
          type="button"
          class="ticket-row"
          :class="{ active: selected?.uuid === ticket.uuid }"
          @click="selectTicket(ticket)"
        >
          <span class="ticket-row-top">
            <span class="reference">#{{ ticket.reference_number }}</span>
            <span class="status">{{ ticket.status }}</span>
          </span>
          <strong>{{ ticket.subject || 'Support request' }}</strong>
          <span class="preview">{{ ticket.last_message || 'No messages yet' }}</span>
          <time>{{ formatDate(ticket.last_message_at || ticket.created_at) }}</time>
        </button>
      </nav>

      <article class="ticket-detail">
        <div v-if="selected" class="detail-heading">
          <div>
            <span class="reference">#{{ selected.reference_number }}</span>
            <h2>{{ selected.subject || 'Support request' }}</h2>
          </div>
          <span class="status">{{ selected.status }}</span>
        </div>
        <div v-if="messagesLoading" class="message-state">Loading conversation…</div>
        <div v-else class="messages">
          <div v-for="message in messages" :key="message.uuid" class="message" :class="message.type">
            <div class="message-meta">
              <strong>{{ message.author_name || (message.type === 'incoming' ? 'You' : 'Support') }}</strong>
              <time>{{ formatDate(message.created_at) }}</time>
            </div>
            <div class="message-body">{{ message.text_content || message.content }}</div>
          </div>
        </div>
      </article>
    </section>
  </main>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@main/api'

const currentUser = ref(null)
const siteName = ref('Support portal')
const tickets = ref([])
const selected = ref(null)
const messages = ref([])
const loading = ref(true)
const messagesLoading = ref(false)
const error = ref(false)

const formatDate = (value) => value
  ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
  : ''

async function selectTicket (ticket) {
  selected.value = ticket
  messagesLoading.value = true
  try {
    const response = await api.getPortalMessages(ticket.uuid, { page_size: 100 })
    messages.value = response.data.data.results || []
  } finally {
    messagesLoading.value = false
  }
}

async function loadPortal () {
  loading.value = true
  error.value = false
  try {
    const [meResponse, ticketsResponse, configResponse] = await Promise.all([
      api.getPortalMe(),
      api.getPortalConversations({ page_size: 100 }),
      api.getConfig()
    ])
    currentUser.value = meResponse.data.data
    siteName.value = configResponse.data.data?.['app.site_name'] || siteName.value
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

<style scoped>
.portal-shell { min-height: 100vh; padding: clamp(1.25rem, 4vw, 4rem); color: hsl(var(--foreground)); background: hsl(var(--background)); }
.portal-header { max-width: 1200px; margin: 0 auto 2rem; display: flex; align-items: end; justify-content: space-between; gap: 1rem; }
.eyebrow { margin: 0 0 .35rem; color: hsl(var(--primary)); font-weight: 800; letter-spacing: .08em; text-transform: uppercase; }
h1, h2, p { margin-top: 0; } h1 { margin-bottom: .25rem; color: hsl(var(--foreground)); font-size: clamp(2rem, 5vw, 3.25rem); line-height: 1; }
.welcome { margin-bottom: 0; color: hsl(var(--muted-foreground)); }.sign-out { color: hsl(var(--primary)); font-weight: 700; }
.ticket-grid { max-width: 1200px; min-height: 65vh; margin: auto; display: grid; grid-template-columns: minmax(260px, 380px) 1fr; overflow: hidden; border: 1px solid hsl(var(--border)); border-top: 3px solid hsl(var(--primary)); border-radius: 18px; background: hsl(var(--card)); box-shadow: 0 22px 60px -38px hsl(var(--foreground)); }
.ticket-list { border-right: 1px solid hsl(var(--border)); overflow: auto; }.ticket-row { width: 100%; padding: 1.15rem; display: grid; gap: .45rem; border: 0; border-bottom: 1px solid hsl(var(--border)); text-align: left; color: inherit; background: transparent; cursor: pointer; }.ticket-row:hover, .ticket-row.active { background: hsl(var(--accent)); }.ticket-row.active { box-shadow: inset 4px 0 hsl(var(--primary)); }
.ticket-row-top, .message-meta, .detail-heading { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }.reference { color: hsl(var(--primary)); font-size: .78rem; font-weight: 800; letter-spacing: .05em; }.status { padding: .25rem .55rem; border-radius: 999px; color: hsl(var(--primary)); background: hsl(var(--accent)); font-size: .75rem; font-weight: 700; }.preview { overflow: hidden; color: hsl(var(--muted-foreground)); font-size: .9rem; text-overflow: ellipsis; white-space: nowrap; }.ticket-row time { color: hsl(var(--muted-foreground)); font-size: .75rem; }
.ticket-detail { padding: clamp(1.25rem, 3vw, 2.5rem); overflow: auto; }.detail-heading { padding-bottom: 1.5rem; border-bottom: 1px solid hsl(var(--border)); }.detail-heading h2 { margin: .35rem 0 0; color: hsl(var(--foreground)); }.messages { display: grid; gap: 1rem; padding-top: 1.5rem; }.message { max-width: 84%; padding: 1rem 1.1rem; border-radius: 4px 16px 16px; background: hsl(var(--muted)); }.message.incoming { margin-left: auto; border-radius: 16px 4px 16px 16px; background: hsl(var(--accent)); }.message-meta { margin-bottom: .65rem; font-size: .78rem; }.message-meta time { color: hsl(var(--muted-foreground)); }.message-body { white-space: pre-wrap; }.state-card, .message-state { max-width: 1200px; margin: 4rem auto; text-align: center; }.state-card button { margin-left: 1rem; color: hsl(var(--primary)); font-weight: 700; }.empty-mark { display: inline-grid; width: 3rem; height: 3rem; margin-bottom: 1rem; place-items: center; border-radius: 50%; color: hsl(var(--primary-foreground)); background: hsl(var(--primary)); }
@media (max-width: 760px) { .portal-shell { padding: 1rem; }.portal-header { align-items: start; }.ticket-grid { display: block; }.ticket-list { max-height: 40vh; border-right: 0; border-bottom: 1px solid hsl(var(--border)); }.ticket-detail { min-height: 50vh; }.message { max-width: 94%; } }
</style>
