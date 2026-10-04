<template>
  <article class="rounded-lg border border-review/40 bg-review-soft p-4 text-foreground" :aria-label="$t('review.cardLabel', { name: review.author_name })">
    <header class="mb-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
      <span class="inline-flex items-center gap-1 rounded-full bg-review px-2 py-0.5 text-xs font-semibold text-review-foreground">
        <component :is="statusIcon" class="size-3.5" aria-hidden="true" />{{ statusLabel }}
      </span>
      <span class="font-medium">{{ review.author_name }}</span>
      <span class="text-muted-foreground">
        · {{ review.kind === 'new' ? $t('review.newEmail') : $t('review.reply') }} ·
        <time :datetime="review.updated_at" :title="new Date(review.updated_at).toLocaleString()">{{ when }}</time>
      </span>
      <span class="ml-auto flex flex-wrap items-center gap-2"><slot name="actions" /></span>
    </header>

    <dl class="mb-3 grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-xs">
      <dt class="text-muted-foreground">{{ $t('review.from') }}</dt>
      <dd class="break-words">{{ fromLabel }}</dd>
      <dt class="text-muted-foreground">{{ $t('review.to') }}</dt>
      <dd class="break-words">{{ (review.to || []).join(', ') || '—' }}</dd>
      <template v-if="review.cc?.length">
        <dt class="text-muted-foreground">{{ $t('review.cc') }}</dt>
        <dd class="break-words">{{ review.cc.join(', ') }}</dd>
      </template>
      <template v-if="review.bcc?.length">
        <dt class="text-muted-foreground">{{ $t('review.bcc') }}</dt>
        <dd class="break-words">{{ review.bcc.join(', ') }}</dd>
      </template>
      <template v-if="subject">
        <dt class="text-muted-foreground">{{ $t('review.subject') }}</dt>
        <dd class="break-words font-medium">{{ subject }}</dd>
      </template>
      <template v-if="review.activation_keys">
        <dt class="text-muted-foreground">{{ $t('keyDistribution.review.keys') }}</dt>
        <dd class="inline-flex items-center gap-1 font-medium">
          <KeyRound class="size-3.5 text-primary" aria-hidden="true" />
          {{ $t('keyDistribution.review.summary', { count: review.activation_keys.count, game: review.activation_keys.app_name }, review.activation_keys.count) }}
        </dd>
      </template>
    </dl>

    <div v-if="!compact" class="rounded-md border bg-background p-3 text-sm">
      <SafeMessageContent v-if="review.display" :message="asMessage" :show-blocked-notice="false" class="native-html break-words" />
      <p v-else class="whitespace-pre-wrap">{{ review.preview }}</p>
    </div>
    <p v-else class="line-clamp-3 text-sm text-foreground/80">{{ review.preview }}</p>

    <ul v-if="files.length" class="mt-3 flex flex-wrap gap-2" :aria-label="$t('review.attachments')">
      <li v-for="file in files" :key="file.id">
        <a :href="file.url" target="_blank" rel="noopener" class="inline-flex max-w-64 items-center gap-1.5 rounded-md border bg-background px-2 py-1 text-xs hover:bg-accent">
          <Paperclip class="size-3.5 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ file.filename }}</span>
        </a>
      </li>
    </ul>

    <p v-if="review.decision_note" class="mt-3 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-sm">
      <span class="font-medium">{{ $t('review.reason') }}</span> {{ review.decision_note }}
    </p>
    <slot />
  </article>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Clock, KeyRound, Paperclip, Undo2 } from 'lucide-vue-next'
import SafeMessageContent from '@shared-ui/components/SafeMessageContent.vue'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'

const props = defineProps({
  review: { type: Object, required: true },
  compact: { type: Boolean, default: false }
})
const { t } = useI18n()

const statusLabel = computed(() =>
  props.review.status === 'pending' ? t('review.awaiting') : t('review.returned')
)
const statusIcon = computed(() => (props.review.status === 'pending' ? Clock : Undo2))
const when = computed(() => getRelativeTime(props.review.updated_at, new Date()))
const subject = computed(() => props.review.subject || (props.review.kind === 'new' ? '' : props.review.conversation_subject))
const fromLabel = computed(() =>
  props.review.address_name ? `${props.review.address_name} <${props.review.address}>` : props.review.address
)
const files = computed(() => (props.review.attachments || []).filter((file) => !file.inline))
// SafeMessageContent renders the server-sanitized HTML in a sandboxed frame.
const asMessage = computed(() => ({
  uuid: props.review.uuid,
  content_type: 'html',
  content: props.review.content,
  display: props.review.display,
  attachments: []
}))
</script>
