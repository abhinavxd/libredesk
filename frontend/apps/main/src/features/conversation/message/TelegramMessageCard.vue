<template>
  <div class="mb-2 flex min-w-0 items-start gap-3 rounded-md border border-border p-3">
    <ContactRound v-if="contact" aria-hidden="true" class="size-5 shrink-0 text-muted-foreground" />
    <MapPin v-else aria-hidden="true" class="size-5 shrink-0 text-muted-foreground" />
    <div class="min-w-0 flex-1 space-y-2">
      <template v-if="contact">
        <p class="font-medium break-words">
          {{ [contact.first_name, contact.last_name].filter(Boolean).join(' ') }}
        </p>
        <p class="break-words text-sm text-muted-foreground">{{ contact.phone_number }}</p>
        <Button
          v-if="userStore.can('contacts:write')"
          type="button"
          variant="outline"
          size="sm"
          class="max-md:min-h-11"
          :is-loading="saving"
          :disabled="saving"
          @click="saveContact"
        >
          {{ $t('globals.messages.saveContact') }}
        </Button>
      </template>
      <template v-else-if="location">
        <p class="font-medium break-words">{{ location.title || $t('globals.terms.location') }}</p>
        <p v-if="location.address" class="text-sm text-muted-foreground break-words">
          {{ location.address }}
        </p>
        <a
          :href="mapURL"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex min-h-8 items-center gap-1 text-sm text-link underline underline-offset-4 max-md:min-h-11 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-sm"
        >
          {{ $t('globals.messages.viewOnMap') }}
          <ExternalLink aria-hidden="true" class="size-3.5 shrink-0" />
        </a>
      </template>
    </div>
  </div>
</template>
<script setup>
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ContactRound, ExternalLink, MapPin } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http'
import api from '@main/api'
const props = defineProps({ message: { type: Object, required: true } })
const contact = computed(() => props.message.meta?.telegram_contact)
const location = computed(() => props.message.meta?.telegram_location)
const mapURL = computed(
  () =>
    `https://maps.google.com/?q=${location.value.location.latitude},${location.value.location.longitude}`
)
const userStore = useUserStore()
const router = useRouter()
const emitter = useEmitter()
const saving = ref(false)
const saveContact = async () => {
  if (saving.value) return
  saving.value = true
  try {
    const response = await api.saveTelegramContact(
      props.message.conversation_uuid,
      props.message.uuid
    )
    await router.push({ name: 'contact-detail', params: { id: response.data.data.id } })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    saving.value = false
  }
}
</script>
