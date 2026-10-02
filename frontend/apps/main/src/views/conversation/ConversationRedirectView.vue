<template>
  <div class="flex h-full items-center justify-center text-sm text-muted-foreground">Opening conversation…</div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '@main/api'

const route = useRoute()
const router = useRouter()

onMounted(async () => {
  try {
    const response = await api.getConversation(route.params.uuid)
    const addressID = response?.data?.data?.address_id
    if (addressID) {
      await router.replace({
        name: 'address-inbox-conversation',
        params: { addressID, uuid: route.params.uuid },
        query: route.query
      })
      return
    }
  } catch {
    // The normal address landing page gives the user a useful next action.
  }
  router.replace({ name: 'address-inbox' })
})
</script>
