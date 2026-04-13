<template>
  <div v-if="integrationStore.shopifyEnabled">
    <!-- Loading skeleton -->
    <div v-if="integrationStore.shopifyLoading" class="space-y-3">
      <Skeleton class="h-4 w-3/4" />
      <Skeleton class="h-4 w-1/2" />
      <Skeleton class="h-4 w-2/3" />
    </div>

    <!-- Customer not found -->
    <div
      v-else-if="integrationStore.shopifyCustomer && !integrationStore.shopifyCustomer.found"
      class="text-sm text-muted-foreground"
    >
      No Shopify customer found for this email.
    </div>

    <!-- Error -->
    <div
      v-else-if="integrationStore.shopifyError"
      class="text-sm text-muted-foreground"
    >
      Could not load Shopify data.
    </div>

    <!-- Customer data -->
    <div v-else-if="customer" class="space-y-4 text-sm">
      <!-- Profile header -->
      <div class="space-y-1.5">
        <div class="flex items-center gap-2">
          <span class="font-medium">
            {{ customer.first_name }} {{ customer.last_name }}
          </span>
        </div>
        <Badge variant="secondary" class="text-xs">
          {{ isReturning ? 'Returning customer' : 'New customer' }}
        </Badge>
        <div class="text-muted-foreground text-xs">
          Customer for {{ customerTenure }}
        </div>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-2 gap-3">
        <div class="rounded-md bg-muted/50 p-2">
          <div class="text-xs text-muted-foreground">Lifetime spend</div>
          <div class="font-medium text-sm">
            {{ formatMoney(customer.total_spent, customer.currency_code) }}
          </div>
        </div>
        <div class="rounded-md bg-muted/50 p-2">
          <div class="text-xs text-muted-foreground">Orders</div>
          <div class="font-medium text-sm">{{ customer.number_of_orders }}</div>
        </div>
      </div>

      <!-- Location -->
      <div v-if="customer.default_address" class="flex items-start gap-2 text-muted-foreground">
        <MapPin class="h-4 w-4 shrink-0 mt-0.5" />
        <span>
          {{ locationString }}
        </span>
      </div>

      <!-- Recent orders -->
      <div v-if="orders.length > 0">
        <div class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
          Recent orders
        </div>
        <div class="space-y-2">
          <div
            v-for="order in orders"
            :key="order.id"
            class="rounded-md border p-2.5 space-y-1.5"
          >
            <div class="flex items-center justify-between">
              <span class="font-medium text-xs">{{ order.name }}</span>
              <span class="text-xs text-muted-foreground">
                {{ formatDate(order.created_at) }}
              </span>
            </div>
            <div class="flex items-center gap-1.5 flex-wrap">
              <Badge
                :variant="financialBadgeVariant(order.financial_status)"
                class="text-[10px] px-1.5 py-0"
              >
                {{ formatStatus(order.financial_status) }}
              </Badge>
              <Badge
                v-if="order.fulfillment_status"
                variant="outline"
                class="text-[10px] px-1.5 py-0"
              >
                {{ formatStatus(order.fulfillment_status) }}
              </Badge>
              <span class="ml-auto text-xs font-medium">
                {{ formatMoney(order.total_price, order.currency_code) }}
              </span>
            </div>

            <!-- Line items (collapsed by default) -->
            <details v-if="order.line_items && order.line_items.length > 0" class="mt-1">
              <summary class="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
                {{ order.line_items.length }} item{{ order.line_items.length !== 1 ? 's' : '' }}
              </summary>
              <div class="mt-1.5 space-y-1.5">
                <div
                  v-for="(item, idx) in order.line_items"
                  :key="idx"
                  class="flex items-center gap-2"
                >
                  <img
                    v-if="item.image_url"
                    :src="item.image_url"
                    :alt="item.title"
                    class="h-8 w-8 rounded object-cover shrink-0"
                  />
                  <div
                    v-else
                    class="h-8 w-8 rounded bg-muted flex items-center justify-center shrink-0"
                  >
                    <Package class="h-4 w-4 text-muted-foreground" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="text-xs truncate">{{ item.title }}</div>
                    <div class="text-[10px] text-muted-foreground">
                      Qty: {{ item.quantity }}
                      <span v-if="item.price"> &middot; {{ formatMoney(item.price, order.currency_code) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </details>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useConversationStore } from '@/stores/conversation'
import { useIntegrationStore } from '@/stores/integration'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { MapPin, Package } from 'lucide-vue-next'

const conversationStore = useConversationStore()
const integrationStore = useIntegrationStore()

const customer = computed(() => integrationStore.shopifyCustomer?.customer)
const orders = computed(() => integrationStore.shopifyCustomer?.orders || [])

const isReturning = computed(() => {
  const count = parseInt(customer.value?.number_of_orders || '0', 10)
  return count > 1
})

const customerTenure = computed(() => {
  if (!customer.value?.created_at) return ''
  const created = new Date(customer.value.created_at)
  const now = new Date()
  const diffMs = now - created
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  if (days < 1) return 'less than a day'
  if (days === 1) return '1 day'
  if (days < 30) return `${days} days`
  const months = Math.floor(days / 30)
  if (months < 12) return `about ${months} month${months !== 1 ? 's' : ''}`
  const years = Math.floor(months / 12)
  const rem = months % 12
  if (rem === 0) return `${years} year${years !== 1 ? 's' : ''}`
  return `${years} year${years !== 1 ? 's' : ''}, ${rem} month${rem !== 1 ? 's' : ''}`
})

const locationString = computed(() => {
  const a = customer.value?.default_address
  if (!a) return ''
  const parts = [a.city, a.province, a.country].filter(Boolean)
  return parts.join(', ')
})

const formatMoney = (amount, currency) => {
  if (!amount) return '$0.00'
  const num = parseFloat(amount)
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency || 'USD'
    }).format(num)
  } catch {
    return `$${num.toFixed(2)}`
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

const formatStatus = (status) => {
  if (!status) return ''
  return status.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

const financialBadgeVariant = (status) => {
  if (!status) return 'secondary'
  const s = status.toLowerCase()
  if (s === 'paid') return 'default'
  if (s === 'refunded' || s === 'voided') return 'destructive'
  return 'secondary'
}

watch(
  () => conversationStore.current?.contact?.email,
  (email) => {
    if (email && integrationStore.shopifyEnabled) {
      integrationStore.fetchShopifyCustomer(email)
    } else {
      integrationStore.clearShopifyCustomer()
    }
  },
  { immediate: true }
)
</script>
