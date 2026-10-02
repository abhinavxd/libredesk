<template>
  <div class="placeholder-container">
    <Spinner v-if="addressStore.loading" />
    <div v-else class="text-center">
      <template v-if="addressStore.addresses.length">
        <p class="placeholder-text">Choose an address to see its conversations.</p>
      </template>
      <template v-else>
        <h2 class="mb-2 text-2xl font-semibold text-foreground">No addresses are available</h2>
        <p class="mb-5 text-sm text-muted-foreground">
          An administrator must connect a mail transport and configure an address before mail can appear here.
        </p>
        <Button v-if="userStore.can('inboxes:manage')" @click="router.push({ name: 'address-list' })">
          Configure addresses
        </Button>
      </template>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useAddressStore } from '@/stores/address'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const addressStore = useAddressStore()
const userStore = useUserStore()

onMounted(() => addressStore.fetchAddresses())
</script>

<style scoped>
.placeholder-container {
  @apply relative flex h-full min-w-[400px] w-full items-center justify-center;
}

.placeholder-text {
  @apply text-muted-foreground;
}
</style>
