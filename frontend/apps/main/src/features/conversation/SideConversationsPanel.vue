<template>
  <div class="space-y-3">
    <p class="text-xs font-medium text-muted-foreground">{{ t('sideConversation.title') }}</p>
    <form class="space-y-2" @submit.prevent="onCreate">
      <Input v-model="to" type="text" :placeholder="t('sideConversation.toPlaceholder')" />
      <Input v-model="subject" type="text" :placeholder="t('globals.terms.subject')" />
      <textarea
        v-model="body"
        class="w-full rounded-md border bg-background px-2 py-1.5 text-sm min-h-20"
        :placeholder="t('sideConversation.bodyPlaceholder')"
      />
      <Button type="submit" size="sm" :disabled="saving">{{ t('sideConversation.send') }}</Button>
    </form>
    <div v-for="thread in threads" :key="thread.id" class="rounded-md border p-2 space-y-2">
      <p class="text-sm font-medium truncate">{{ thread.subject }}</p>
      <p class="text-xs text-muted-foreground truncate">{{ (thread.recipients || []).join(', ') }}</p>
      <div
        v-for="msg in thread.messages || []"
        :key="msg.id"
        class="text-xs border-l pl-2"
      >
        <span class="text-muted-foreground">{{ msg.author_first_name }} · {{ msg.direction }}</span>
        <div v-html="msg.content"></div>
      </div>
      <div class="flex gap-1">
        <Input v-model="replies[thread.uuid]" type="text" :placeholder="t('sideConversation.replyPlaceholder')" />
        <Button size="sm" variant="outline" @click="onReply(thread)">{{ t('globals.messages.send') }}</Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { useConversationStore } from '@/stores/conversation'
import api from '@/api'

const { t } = useI18n()
const conversationStore = useConversationStore()
const threads = ref([])
const to = ref('')
const subject = ref('')
const body = ref('')
const replies = ref({})
const saving = ref(false)

async function load() {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return
  try {
    const resp = await api.getSideConversations(uuid)
    threads.value = resp.data.data || []
  } catch {
    threads.value = []
  }
}

async function onCreate() {
  const uuid = conversationStore.current?.uuid
  if (!uuid) return
  saving.value = true
  try {
    await api.createSideConversation(uuid, {
      to: to.value.split(/[,;\s]+/).filter(Boolean),
      subject: subject.value,
      content: body.value
    })
    to.value = ''
    subject.value = ''
    body.value = ''
    await load()
  } finally {
    saving.value = false
  }
}

async function onReply(thread) {
  const uuid = conversationStore.current?.uuid
  const content = replies.value[thread.uuid]
  if (!uuid || !content) return
  await api.replySideConversation(uuid, thread.uuid, { content })
  replies.value[thread.uuid] = ''
  await load()
}

onMounted(load)
watch(() => conversationStore.current?.uuid, load)
</script>
