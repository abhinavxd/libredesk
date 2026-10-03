<template>
  <section :aria-label="$t('review.earlierMessages')">
    <h3 class="mb-2 text-sm font-medium text-muted-foreground">{{ $t('review.earlierMessages') }}</h3>
    <div v-if="loading" class="space-y-2">
      <div v-for="n in 2" :key="n" class="h-20 animate-pulse rounded-lg bg-muted" />
    </div>
    <p v-else-if="failed" class="text-sm text-muted-foreground">{{ $t('review.contextFailed') }}</p>
    <p v-else-if="!messages.length" class="text-sm text-muted-foreground">{{ $t('review.noEarlierMessages') }}</p>
    <template v-else>
      <Button v-if="hiddenCount > 0" variant="ghost" size="sm" class="mb-2" @click="showAll = true">
        {{ $t('review.showEarlier', hiddenCount) }}
      </Button>
      <ol class="space-y-3">
        <li v-for="message in visible" :key="message.uuid">
          <article
            class="rounded-lg border p-3 text-sm"
            :class="message.private ? 'border-warning/30 bg-private' : message.type === 'incoming' ? 'bg-card' : 'bg-muted/40'"
          >
            <header class="mb-2 flex flex-wrap items-baseline gap-x-2 text-xs text-muted-foreground">
              <span class="text-sm font-semibold text-foreground">{{ authorName(message) }}</span>
              <span v-if="message.private">{{ $t('globals.terms.privateNote') }}</span>
              <span v-else-if="recipients(message)">→ {{ recipients(message) }}</span>
              <time class="ml-auto" :datetime="message.created_at">{{ new Date(message.created_at).toLocaleString() }}</time>
            </header>
            <SafeMessageContent
              v-if="message.content_type !== 'text' && message.display"
              :message="message"
              :show-blocked-notice="false"
              :show-quoted-text="false"
              class="native-html break-words"
            />
            <p v-else class="whitespace-pre-wrap">{{ message.text_content || message.content }}</p>
          </article>
        </li>
      </ol>
    </template>
  </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import SafeMessageContent from '@shared-ui/components/SafeMessageContent.vue'
import api from '@main/api'

// Earlier messages of a conversation, read-only, so a reviewer can judge a
// reply against what it answers. Loaded directly so the inbox's open
// conversation and read state are untouched.
const props = defineProps({
  conversationUUID: { type: String, default: '' },
  // Re-fetch when this changes, e.g. after a live update.
  refreshKey: { type: [Number, String], default: 0 }
})

const RECENT = 4
const messages = ref([])
const loading = ref(false)
const failed = ref(false)
const showAll = ref(false)

const visible = computed(() => (showAll.value ? messages.value : messages.value.slice(-RECENT)))
const hiddenCount = computed(() => messages.value.length - visible.value.length)

const authorName = (message) => {
  const author = message.author || {}
  return [author.first_name, author.last_name].filter(Boolean).join(' ').trim() || author.email || ''
}
const recipients = (message) => (Array.isArray(message.meta?.to) ? message.meta.to.join(', ') : '')

async function load() {
  messages.value = []
  failed.value = false
  showAll.value = false
  if (!props.conversationUUID) return
  loading.value = true
  try {
    const response = await api.getConversationMessages(props.conversationUUID, { page: 1, page_size: 50 })
    const rows = response?.data?.data?.results || []
    messages.value = rows
      .filter((message) => message.type !== 'activity')
      .sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

watch(() => [props.conversationUUID, props.refreshKey], load, { immediate: true })
</script>
