import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

export const orderAddresses = (addresses) =>
  [...addresses].sort((left, right) => {
    const kindOrder = (address) => (address.kind === 'mailbox' ? 0 : 1)
    const kindDifference = kindOrder(left) - kindOrder(right)
    if (kindDifference) return kindDifference

    const addressDifference = String(left.address).localeCompare(String(right.address))
    if (addressDifference) return addressDifference

    return Number(left.id) - Number(right.id)
  })

// Addresses are the product-facing mail endpoints. A transport mailbox may
// back several addresses, but agents should never need to reason about that in
// the main workspace.
export const useAddressStore = defineStore('address', () => {
  const addresses = ref([])
  const loading = ref(false)
  const emitter = useEmitter()

  const byID = computed(() => new Map(addresses.value.map((address) => [String(address.id), address])))
  const get = (id) => byID.value.get(String(id))

  async function fetchAddresses(force = false) {
    if (loading.value || (!force && addresses.value.length)) return
    loading.value = true
    try {
      const response = await api.getAddresses()
      addresses.value = orderAddresses(response?.data?.data || [])
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      loading.value = false
    }
  }

  return { addresses, loading, byID, get, fetchAddresses }
})
