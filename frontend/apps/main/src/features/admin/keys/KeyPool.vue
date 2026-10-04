<template>
  <section class="box flex min-w-0 flex-col bg-card" :aria-labelledby="headingId">
    <header class="flex items-center justify-between gap-2 border-b px-3 py-2">
      <h2 :id="headingId" class="text-sm font-semibold">{{ title }}</h2>
      <span class="text-sm tabular-nums text-muted-foreground">{{ total }}</span>
    </header>
    <div class="border-b p-2">
      <Input
        v-model="search"
        type="search"
        class="h-8 text-sm"
        autocomplete="off"
        spellcheck="false"
        :placeholder="isActivated ? $t('keyDistribution.pool.searchActivated') : $t('keyDistribution.pool.searchRedeemable')"
        :aria-label="$t('keyDistribution.pool.search', { pool: title })"
      />
    </div>

    <!-- Fixed height; the list scrolls inside. -->
    <div class="h-80 overflow-y-auto">
      <p v-if="error" role="alert" class="p-3 text-sm text-destructive">{{ error }}</p>
      <div v-else-if="loading && !keys.length" class="space-y-2 p-3">
        <div v-for="n in 4" :key="n" class="h-8 animate-pulse rounded-md bg-muted" />
      </div>
      <p v-else-if="!keys.length" class="p-6 text-center text-sm text-muted-foreground">
        {{ search ? $t('keyDistribution.pool.noMatches') : isActivated ? $t('keyDistribution.pool.emptyActivated') : $t('keyDistribution.pool.emptyRedeemable') }}
      </p>
      <ul v-else>
        <li
          v-for="key in keys"
          :key="key.id"
          class="group/row flex items-start gap-1 border-b px-2 py-1.5 last:border-b-0"
        >
          <div class="min-w-0 flex-1">
            <button
              type="button"
              class="max-w-full truncate rounded-sm px-1 text-left font-mono text-sm tracking-wide hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed"
              :class="{ 'text-muted-foreground': revealedId !== key.id }"
              :aria-label="revealedId === key.id ? $t('keyDistribution.pool.hide') : $t('keyDistribution.pool.reveal')"
              :aria-pressed="revealedId === key.id"
              :disabled="!enabled"
              @click="$emit('reveal', key)"
            >
              {{ revealedId === key.id ? revealedKey : MASK }}
            </button>
            <p class="truncate px-1 text-xs text-muted-foreground">
              <template v-if="isActivated">
                <span>{{ (key.recipients || []).join(', ') || $t('keyDistribution.pool.noRecipient') }}</span>
                <template v-if="key.conversation_ref">
                  ·
                  <router-link
                    v-if="key.conversation_uuid && key.address_id"
                    :to="{ name: 'address-inbox-conversation', params: { addressID: key.address_id, uuid: key.conversation_uuid } }"
                    class="link-style"
                  >#{{ key.conversation_ref }}</router-link>
                  <span v-else>#{{ key.conversation_ref }}</span>
                </template>
                <template v-if="key.sent_by_name"> · {{ key.sent_by_name }}</template>
                <template v-if="key.approved_by_name"> · {{ $t('keyDistribution.pool.approvedBy', { name: key.approved_by_name }) }}</template>
                · <time :datetime="key.activated_at" :title="new Date(key.activated_at).toLocaleString()">{{ relative(key.activated_at) }}</time>
                <span v-if="key.delivery_status === 'failed'" class="text-destructive"> · {{ $t('keyDistribution.pool.deliveryFailed') }}</span>
                <span v-else-if="key.delivery_status === 'pending'"> · {{ $t('keyDistribution.pool.deliveryPending') }}</span>
              </template>
              <template v-else>
                {{ key.added_by_name ? $t('keyDistribution.pool.addedBy', { name: key.added_by_name }) : $t('keyDistribution.pool.added') }}
                · <time :datetime="key.created_at" :title="new Date(key.created_at).toLocaleString()">{{ relative(key.created_at) }}</time>
              </template>
            </p>
          </div>
          <Button
            v-if="!isActivated"
            type="button"
            variant="ghost"
            size="icon"
            class="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive [@media(hover:hover)]:opacity-0 group-hover/row:opacity-100 focus-visible:!opacity-100"
            :aria-label="$t('keyDistribution.pool.void')"
            :disabled="!enabled"
            @click="$emit('void', key)"
          >
            <Ban class="size-4" />
          </Button>
        </li>
      </ul>
      <div v-if="keys.length < total" class="p-2">
        <Button type="button" variant="ghost" size="sm" class="w-full" :disabled="loading" @click="loadMore">
          {{ $t('keyDistribution.pool.loadMore') }}
        </Button>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { Ban } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import api from '@main/api'

// Rows never carry the key. Clicking a row asks the page to reveal that one
// key; the page shows at most one at a time across both pools.
const MASK = '••••••••••••••••'
const PAGE_SIZE = 50

const props = defineProps({
  appId: { type: Number, required: true },
  status: { type: String, required: true },
  title: { type: String, required: true },
  enabled: { type: Boolean, default: true },
  revealedId: { type: Number, default: 0 },
  revealedKey: { type: String, default: '' },
  // Bumped by the page after imports, voids and sends to reload the list.
  refreshKey: { type: Number, default: 0 }
})

defineEmits(['reveal', 'void'])

const { t } = useI18n()
const headingId = `key-pool-${props.status}`
const isActivated = computed(() => props.status === 'activated')
const keys = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const error = ref('')
const search = ref('')
let generation = 0

const relative = (value) => (value ? getRelativeTime(new Date(value)) : '')

const load = async (nextPage = 1) => {
  if (!props.appId) return
  const current = ++generation
  loading.value = true
  error.value = ''
  try {
    const response = await api.getActivationKeys(props.appId, {
      status: props.status,
      search: search.value.trim(),
      page: nextPage,
      page_size: PAGE_SIZE
    })
    if (current !== generation) return
    const data = response.data.data
    keys.value = nextPage === 1 ? data.results : [...keys.value, ...data.results]
    total.value = data.total
    page.value = nextPage
  } catch (err) {
    if (current === generation) error.value = handleHTTPError(err).message || t('keyDistribution.pool.loadFailed')
  } finally {
    if (current === generation) loading.value = false
  }
}

const loadMore = () => load(page.value + 1)
const searchSoon = useDebounceFn(() => load(1), 300)

watch(search, searchSoon)
watch(
  () => [props.appId, props.refreshKey],
  () => load(1),
  { immediate: true }
)
</script>
