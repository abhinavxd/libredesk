import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export const useIntegrationStore = defineStore('integration', () => {
  const integrations = ref([])
  const shopifyCustomer = ref(null)
  const shopifyLoading = ref(false)
  const shopifyError = ref(null)
  const customerCache = ref({})

  const shopifyEnabled = computed(() => {
    const shopify = integrations.value.find((i) => i.provider === 'shopify')
    return shopify?.enabled ?? false
  })

  const shopifyConfigured = computed(() => {
    return integrations.value.some((i) => i.provider === 'shopify')
  })

  async function fetchIntegrations() {
    try {
      const resp = await api.getIntegrations()
      integrations.value = resp.data.data || []
    } catch {
      integrations.value = []
    }
  }

  async function fetchShopifyCustomer(email) {
    if (!email || !shopifyEnabled.value) {
      shopifyCustomer.value = null
      return
    }

    if (customerCache.value[email]) {
      shopifyCustomer.value = customerCache.value[email]
      return
    }

    shopifyLoading.value = true
    shopifyError.value = null
    try {
      const resp = await api.getShopifyCustomer(email)
      const data = resp.data.data
      shopifyCustomer.value = data
      customerCache.value[email] = data
    } catch (err) {
      shopifyError.value = err
      shopifyCustomer.value = null
    } finally {
      shopifyLoading.value = false
    }
  }

  function clearShopifyCustomer() {
    shopifyCustomer.value = null
    shopifyError.value = null
  }

  return {
    integrations,
    shopifyEnabled,
    shopifyConfigured,
    shopifyCustomer,
    shopifyLoading,
    shopifyError,
    fetchIntegrations,
    fetchShopifyCustomer,
    clearShopifyCustomer
  }
})
