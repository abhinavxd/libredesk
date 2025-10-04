<template>
  <div class="placeholder-container">
    <Spinner v-if="isLoading" />
    <template v-else>
      <div v-if="showGettingStarted" class="getting-started-wrapper">
        <div class="text-center">
          <h2 class="text-2xl font-semibold text-foreground mb-6">
            {{ $t('setup.completeYourSetup') }}
          </h2>

          <div class="space-y-4 mb-6">
            <div class="checklist-item" :class="{ completed: hasInboxes }">
              <CheckCircle v-if="hasInboxes" class="check-icon completed" />
              <Circle v-else class="w-5 h-5 text-muted-foreground" />
              <span class="flex-1 text-left ml-3 text-foreground">
                {{ $t('setup.createFirstInbox') }}
              </span>
              <Button
                v-if="!hasInboxes"
                variant="ghost"
                size="sm"
                @click="router.push({ name: 'inbox-list' })"
                class="ml-auto"
              >
                {{ $t('globals.messages.setUp') }}
              </Button>
            </div>

            <div class="checklist-item" :class="{ completed: hasAgents, disabled: !hasInboxes }">
              <CheckCircle v-if="hasAgents" class="check-icon completed" />
              <Circle v-else class="w-5 h-5 text-muted-foreground" />
              <span class="flex-1 text-left ml-3 text-foreground">
                {{ $t('setup.inviteTeammates') }}
              </span>
              <Button
                v-if="!hasAgents && hasInboxes"
                variant="ghost"
                size="sm"
                @click="router.push({ name: 'agent-list' })"
                class="ml-auto"
              >
                {{ $t('globals.messages.invite') }}
              </Button>
            </div>
          </div>
        </div>
      </div>
      <div v-else>
        <p class="placeholder-text">{{ $t('conversation.placeholder') }}</p>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { CheckCircle, Circle } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'

const router = useRouter()
const inboxStore = useInboxStore()
const usersStore = useUsersStore()
const isLoading = ref(true)

onMounted(async () => {
  try {
    await Promise.all([inboxStore.fetchInboxes(), usersStore.fetchUsers()])
  } finally {
    isLoading.value = false
  }
})

const hasInboxes = computed(() => inboxStore.inboxes.length > 0)
const hasAgents = computed(() => usersStore.users.length > 0)
const showGettingStarted = computed(() => !hasInboxes.value || !hasAgents.value)
</script>

<style scoped>
.placeholder-container {
  @apply h-screen w-full flex items-center justify-center min-w-[400px] relative;
}

.getting-started-wrapper {
  @apply w-full max-w-md mx-auto px-4;
}

.checklist-item {
  @apply flex items-center justify-between py-3 px-4 rounded-lg border border-border;
}

.checklist-item.completed {
  @apply bg-muted/50;
}

.checklist-item.disabled {
  @apply opacity-50;
}

.check-icon.completed {
  @apply w-5 h-5 text-primary;
}
</style>
