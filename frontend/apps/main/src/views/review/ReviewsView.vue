<template>
  <div class="flex h-full min-h-0 flex-col">
    <header class="flex h-12 shrink-0 items-center gap-2 border-b px-2">
      <SidebarTrigger class="cursor-pointer" />
      <Button v-if="isMobile && uuid" variant="ghost" size="icon" :aria-label="$t('review.backToList')" @click="router.push({ name: 'reviews' })">
        <ArrowLeft class="size-4" />
      </Button>
      <ShieldCheck class="size-5 text-review" aria-hidden="true" />
      <h1 class="min-w-0 flex-1 truncate text-xl font-semibold">{{ reviewStore.isReviewer ? $t('review.queueTitle') : $t('review.mySubmissions') }}</h1>
    </header>

    <div class="flex min-h-0 flex-1">
      <!-- Queue -->
      <nav
        v-show="!isMobile || !uuid"
        class="flex min-h-0 w-full shrink-0 flex-col border-r md:w-80 lg:w-96"
        :aria-label="$t('review.title')"
      >
        <div v-if="reviewStore.loading && !reviewStore.loaded" class="space-y-2 p-3">
          <div v-for="n in 3" :key="n" class="h-20 animate-pulse rounded-md bg-muted" />
        </div>
        <div v-else-if="!reviewStore.items.length" class="flex flex-1 flex-col items-center justify-center gap-2 p-8 text-center text-sm text-muted-foreground">
          <CheckCheck class="size-8 text-review/60" aria-hidden="true" />
          {{ reviewStore.isReviewer ? $t('review.empty') : $t('review.emptyMine') }}
        </div>
        <ul v-else class="min-h-0 flex-1 overflow-y-auto">
          <li v-for="item in reviewStore.items" :key="item.uuid">
            <router-link
              :to="{ name: 'reviews', params: { uuid: item.uuid } }"
              class="block border-b border-l-4 px-4 py-3 transition-colors hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
              :class="item.uuid === uuid ? 'border-l-review bg-review-soft' : 'border-l-transparent'"
              :aria-current="item.uuid === uuid ? 'page' : undefined"
            >
              <div class="flex items-center gap-2">
                <span class="min-w-0 flex-1 truncate text-sm font-semibold">{{ item.author_name }}</span>
                <time class="shrink-0 text-xs text-muted-foreground" :datetime="item.updated_at">{{ relative(item.updated_at) }}</time>
              </div>
              <div class="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                <component :is="item.kind === 'new' ? SquarePen : Reply" class="size-3.5 shrink-0" aria-hidden="true" />
                <span class="truncate">{{ item.kind === 'new' ? $t('review.newEmail') : $t('review.reply') }} · {{ item.address_name || item.address }}</span>
                <Badge v-if="item.status !== 'pending'" variant="warning" class="ml-auto shrink-0 px-1.5 py-0 font-medium">{{ $t('review.returned') }}</Badge>
              </div>
              <div class="mt-1 truncate text-sm font-medium">{{ item.subject || item.conversation_subject || $t('review.noSubject') }}</div>
              <p class="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{{ item.preview }}</p>
            </router-link>
          </li>
        </ul>
      </nav>

      <!-- Detail -->
      <main v-show="!isMobile || uuid" class="min-h-0 min-w-0 flex-1 overflow-y-auto">
        <div v-if="!uuid" class="flex h-full items-center justify-center p-8 text-center text-sm text-muted-foreground">
          {{ reviewStore.items.length ? $t('review.select') : '' }}
        </div>
        <div v-else-if="loadingDetail && !current" class="space-y-3 p-6">
          <div class="h-6 w-1/2 animate-pulse rounded-md bg-muted" />
          <div class="h-40 animate-pulse rounded-lg bg-muted" />
        </div>
        <div v-else-if="current" class="mx-auto max-w-3xl space-y-6 p-4 md:p-6">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0">
              <h2 class="break-words text-lg font-semibold">{{ current.subject || current.conversation_subject || $t('review.noSubject') }}</h2>
              <p class="text-sm text-muted-foreground">{{ $t('review.submittedBy', { name: current.author_name, address: current.address_name || current.address }) }}</p>
            </div>
            <router-link
              v-if="current.conversation_uuid"
              :to="{ name: 'address-inbox-conversation', params: { addressID: current.address_id, uuid: current.conversation_uuid } }"
              class="inline-flex items-center gap-1 text-sm text-link hover:underline"
            >
              {{ $t('review.openConversation') }}<ExternalLink class="size-3.5" aria-hidden="true" />
            </router-link>
          </div>

          <ReviewContext v-if="current.kind === 'reply'" :conversationUUID="current.conversation_uuid" :refreshKey="contextKey" />
          <p v-else class="rounded-md border border-dashed px-3 py-2 text-sm text-muted-foreground">{{ $t('review.noEarlierMessages') }}</p>

          <ReviewSubmission :review="current">
            <template #actions>
              <template v-if="canDecide">
                <Button variant="outline" size="sm" :disabled="busy" @click="openDeny">
                  <Undo2 class="size-4" />{{ $t('review.deny') }}
                </Button>
                <Button size="sm" class="bg-review text-review-foreground hover:bg-review/90" :disabled="busy" @click="confirmApprove = true">
                  <Send class="size-4" />{{ $t('review.approve') }}
                </Button>
              </template>
              <template v-else-if="isMine && current.status === 'pending'">
                <Button variant="outline" size="sm" :disabled="busy" @click="confirmWithdraw = true">{{ $t('review.withdraw') }}</Button>
              </template>
              <template v-else-if="isMine && current.kind === 'new'">
                <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" :disabled="busy" @click="confirmDiscard = true">
                  {{ $t('review.discard') }}
                </Button>
                <Button size="sm" :disabled="busy" @click="editReturned"><Pencil class="size-4" />{{ $t('review.editAndResubmit') }}</Button>
              </template>
            </template>
          </ReviewSubmission>
        </div>
      </main>
    </div>

    <!-- Approve -->
    <AlertDialog :open="confirmApprove" @update:open="confirmApprove = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('review.approveTitle') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ $t('review.approveDescription', { name: current?.author_name, address: current?.address, recipients: (current?.to || []).join(', ') }) }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction class="bg-review text-review-foreground hover:bg-review/90" @click="approve">{{ $t('review.approve') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Deny -->
    <AlertDialog :open="confirmDeny" @update:open="confirmDeny = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('review.denyTitle', { name: current?.author_name }) }}</AlertDialogTitle>
          <AlertDialogDescription>{{ $t('review.denyDescription') }}</AlertDialogDescription>
        </AlertDialogHeader>
        <label class="block space-y-1.5 text-sm">
          <span class="font-medium">{{ $t('review.denyReasonLabel') }}</span>
          <textarea
            v-model="denyNote"
            rows="3"
            maxlength="2000"
            class="w-full rounded-md border bg-background px-3 py-2"
            :placeholder="$t('review.denyReasonPlaceholder')"
          />
        </label>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" @click="deny">{{ $t('review.denyConfirm') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Withdraw -->
    <AlertDialog :open="confirmWithdraw" @update:open="confirmWithdraw = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('review.withdrawTitle') }}</AlertDialogTitle>
          <AlertDialogDescription>
            {{ current?.kind === 'new' ? $t('review.withdrawDescriptionNew') : $t('review.withdrawDescriptionReply') }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction @click="withdraw">{{ $t('review.withdraw') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Discard -->
    <AlertDialog :open="confirmDiscard" @update:open="confirmDiscard = $event">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ $t('review.discardTitle') }}</AlertDialogTitle>
          <AlertDialogDescription>{{ $t('review.discardDescription') }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" @click="discard">{{ $t('review.discard') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, CheckCheck, ExternalLink, Pencil, Reply, Send, ShieldCheck, SquarePen, Undo2 } from 'lucide-vue-next'
import { Badge } from '@shared-ui/components/ui/badge'
import { Button } from '@shared-ui/components/ui/button'
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { useIsMobile } from '@shared-ui/composables'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import ReviewContext from '@main/features/review/ReviewContext.vue'
import ReviewSubmission from '@main/features/review/ReviewSubmission.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { useComposeStore } from '@main/stores/compose'
import { useReviewStore } from '@main/stores/review'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'

const props = defineProps({ uuid: { type: String, default: '' } })

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const isMobile = useIsMobile()
const reviewStore = useReviewStore()
const composeStore = useComposeStore()
const userStore = useUserStore()

const current = ref(null)
const loadingDetail = ref(false)
const busy = ref(false)
const contextKey = ref(0)
const confirmApprove = ref(false)
const confirmDeny = ref(false)
const confirmWithdraw = ref(false)
const confirmDiscard = ref(false)
const denyNote = ref('')

const isMine = computed(() => current.value?.author_id === userStore.userID)
const canDecide = computed(() => reviewStore.isReviewer && !isMine.value && current.value?.status === 'pending')
const relative = (date) => getRelativeTime(date, new Date())

// The next item to open once the current one leaves the queue.
function nextAfter(uuid) {
  const items = reviewStore.items
  const index = items.findIndex((item) => item.uuid === uuid)
  return items[index + 1]?.uuid || items[index - 1]?.uuid || items.find((item) => item.uuid !== uuid)?.uuid || ''
}

function openNext(next) {
  if (next && !isMobile.value) router.replace({ name: 'reviews', params: { uuid: next } })
  else router.replace({ name: 'reviews' })
}

async function loadDetail(uuid, { quiet = false } = {}) {
  if (!uuid) {
    current.value = null
    return
  }
  if (!quiet) loadingDetail.value = true
  try {
    const review = (await api.getReview(uuid)).data?.data
    // A reviewer's queue only holds pending mail; someone else got there first.
    if (reviewStore.isReviewer && review.author_id !== userStore.userID && review.status !== 'pending') {
      throw new Error('decided')
    }
    current.value = review
  } catch {
    if (uuid !== props.uuid) return
    if (current.value?.uuid === uuid || quiet) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'info', description: t('review.alreadyDecided') })
    }
    current.value = null
    openNext(nextAfter(uuid))
  } finally {
    loadingDetail.value = false
  }
}

watch(() => props.uuid, (uuid) => loadDetail(uuid), { immediate: true })

// Live updates: keep the open item honest and its conversation context fresh.
watch(
  () => reviewStore.version,
  () => {
    if (!props.uuid) return
    contextKey.value++
    loadDetail(props.uuid, { quiet: true })
  }
)

// On desktop, open the first item rather than an empty pane.
watch(
  () => reviewStore.items,
  (items) => {
    if (!props.uuid && !isMobile.value && items.length) router.replace({ name: 'reviews', params: { uuid: items[0].uuid } })
  }
)

function openDeny() {
  denyNote.value = ''
  confirmDeny.value = true
}

async function act(action) {
  const uuid = current.value?.uuid
  if (!uuid || busy.value) return
  busy.value = true
  const next = nextAfter(uuid)
  const result = await action(uuid)
  busy.value = false
  if (result) {
    current.value = null
    openNext(next)
  } else {
    loadDetail(uuid, { quiet: true })
  }
}

const approve = () => act((uuid) => reviewStore.approve(uuid))
const deny = () => act((uuid) => reviewStore.deny(uuid, denyNote.value.trim()))
const discard = () => act((uuid) => reviewStore.discard(uuid))

// A withdrawn reply goes back to its draft and leaves the list; a withdrawn
// new email stays here, returned, ready to edit.
async function withdraw() {
  const review = current.value
  if (!review || busy.value) return
  if (review.kind === 'reply') return act((uuid) => reviewStore.withdraw(uuid, 'reply'))
  busy.value = true
  const result = await reviewStore.withdraw(review.uuid, 'new')
  busy.value = false
  await reviewStore.fetchReviews()
  if (result) loadDetail(review.uuid)
}

function editReturned() {
  const review = current.value
  if (!review) return
  composeStore.open({
    review_uuid: review.uuid,
    address_id: review.address_id,
    subject: review.subject,
    content: review.display?.html ?? review.content,
    to: review.to,
    cc: review.cc,
    bcc: review.bcc,
    attachments: review.attachments
  })
}

onMounted(() => {
  reviewStore.fetchReviews()
  reviewStore.fetchCounts()
})
</script>
