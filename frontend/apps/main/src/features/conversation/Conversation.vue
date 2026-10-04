<template>
  <div class="flex flex-col h-full">
    <div v-if="isMobile" class="shrink-0 px-2 pt-1">
      <Button
        variant="ghost"
        size="icon"
        :aria-label="t('globals.messages.back')"
        @click="goBackToList"
      >
        <ChevronLeft class="w-4 h-4" />
      </Button>
    </div>

    <Transition
      enter-active-class="transition duration-200 ease-out motion-reduce:transition-none"
      enter-from-class="-translate-y-2 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
    >
      <section
        v-if="imagePermissionMessages.length"
        :key="conversationStore.current?.uuid"
        :aria-label="t('conversation.imagePermissions')"
        class="w-full shrink-0 max-h-[40%] overflow-y-auto border-b bg-muted divide-y"
      >
        <MessageImagePermissions
          v-for="message in imagePermissionMessages"
          :key="message.uuid"
          :message="message"
          :show-timestamp="imagePermissionMessages.length > 1"
          @updated="conversationStore.updateImagePermissions"
        />
      </section>
    </Transition>

    <!-- Messages & reply box -->
    <div v-if="canCompose" class="flex min-h-0 flex-grow flex-col overflow-hidden">
      <ResizablePanelGroup
        direction="vertical"
        class="min-h-0 flex-1"
        @layout="onConversationLayout"
      >
        <ResizablePanel :default-size="panelSizes[0]" :min-size="45">
          <div class="h-full min-h-0 overflow-hidden">
            <MessageList class="h-full overflow-y-auto" />
          </div>
        </ResizablePanel>
        <ResizableHandle
          withHandle
          class="h-2 border-0 bg-transparent transition-colors hover:bg-primary/10"
        />
        <ResizablePanel :default-size="panelSizes[1]" :min-size="18" :max-size="55">
          <ReplyBox />
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
    <MessageList v-else class="min-h-0 flex-1 overflow-y-auto" />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue'
import { useStorage } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { ChevronLeft } from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { useIsMobile } from '@shared-ui/composables'
import { Button } from '@shared-ui/components/ui/button'
import MessageList from '@/features/conversation/message/MessageList.vue'
import ReplyBox from './ReplyBox.vue'
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle
} from '@shared-ui/components/ui/resizable'
import MessageImagePermissions from './message/MessageImagePermissions.vue'
import { EMITTER_EVENTS, CONVERSATION_ACTIONS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { downloadBlobResponse, parseBlobError } from '@shared-ui/utils/file'
import api from '@main/api'
import { permissions as perms } from '@main/constants/permissions.js'
const conversationStore = useConversationStore()
const userStore = useUserStore()
const emitter = useEmitter()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const isMobile = useIsMobile()
const imagePermissionMessages = computed(() =>
  conversationStore.conversationMessages.filter(
    (message) => message.display?.blocked_images > 0 || message.display?.sender_trusted
  )
)
// Contributors compose too; their replies go to review instead of out.
const canCompose = computed(
  () =>
    userStore.can(perms.MESSAGES_WRITE) ||
    userStore.can(perms.MESSAGES_WRITE_PRIVATE) ||
    userStore.can(perms.REVIEWS_SUBMIT)
)
// Give email recipients, the editor, and the action bar room to coexist by default.
// The composer remains resizable down to a compact but usable 18% of the thread.
const panelSizes = useStorage('conversationComposerPanelSizesV2', [62, 38])

const onConversationLayout = (sizes) => {
  if (sizes.length === 2) panelSizes.value = sizes
}

// Each detail route is `<list route name>-conversation`.
const goBackToList = () => {
  const listName = String(route.name).replace(/-conversation$/, '')
  const { ...params } = route.params
  const target = router.resolve({ name: listName, params })
  if (window.history.state?.back?.split('?')[0] === target.path) router.back()
  else router.push(target)
}

const downloadTranscript = async () => {
  const conversation = conversationStore.current
  if (!conversation) return
  try {
    const response = await api.getConversationTranscript(conversation.uuid)
    downloadBlobResponse(response, `transcript-${conversation.reference_number}.txt`)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(await parseBlobError(error)).message
    })
  }
}

const paletteActions = {
  [CONVERSATION_ACTIONS.DOWNLOAD_TRANSCRIPT]: downloadTranscript
}
const onPaletteAction = (action) => paletteActions[action]?.()

onMounted(() => emitter.on(EMITTER_EVENTS.CONVERSATION_ACTION, onPaletteAction))
onUnmounted(() => emitter.off(EMITTER_EVENTS.CONVERSATION_ACTION, onPaletteAction))
</script>
