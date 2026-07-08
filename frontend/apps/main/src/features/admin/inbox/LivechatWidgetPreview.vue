<template>
  <div class="relative w-full h-[750px] rounded-xl border border-border bg-muted overflow-hidden">
    <!-- Widget window, themed independently of the admin app. -->
    <transition name="ld-preview-window">
      <div
        v-if="open"
        class="absolute top-4 left-3 right-3 mx-auto max-w-[360px]"
        :style="{ bottom: windowBottom + 'px' }"
      >
        <div
          class="libredesk-widget-preview flex flex-col h-full  bg-background text-foreground rounded-2xl overflow-hidden shadow-2xl border border-border"
          :class="isDark ? 'dark' : 'light'"
          :style="primaryStyle"
        >
          <!-- Chat view -->
          <template v-if="view === 'chat'">
            <!-- Chat header -->
            <div class="flex items-center p-2 border-b border-border gap-3 shrink-0">
              <button
                type="button"
                class="flex items-center justify-center size-8 rounded-md hover:bg-accent text-foreground"
                @click="view = 'messages'"
              >
                <ArrowLeft :size="18" />
              </button>
              <div class="flex items-center gap-2">
                <div
                  class="size-10 rounded-full bg-secondary text-secondary-foreground flex items-center justify-center overflow-hidden shrink-0"
                >
                  <img :src="launcherLogo" alt="" class="w-full h-full object-cover" />
                </div>
                <div class="flex flex-col">
                  <h3 class="text-base font-bold leading-tight">
                    {{ config.brand_name || $t('globals.terms.name') }}
                  </h3>
                  <p class="text-xs text-muted-foreground flex items-center gap-1">
                    <template v-if="replyExpectation">{{ replyExpectation }}</template>
                    <template v-else>
                      <span class="inline-block w-2 h-2 rounded-full bg-green-500"></span>
                      {{ $t('globals.terms.online', 1) }}
                    </template>
                  </p>
                </div>
              </div>
              <div
                class="ml-auto flex items-center justify-center size-8 rounded-md text-muted-foreground"
              >
                <Maximize2 :size="16" />
              </div>
            </div>

            <!-- Pre-chat form -->
            <template v-if="showPrechat && !openedExisting">
              <div class="flex-1 min-h-0 overflow-y-auto p-4 space-y-4">
                <div
                  v-if="config.prechat_form?.title"
                  class="text-xl text-foreground mb-2 text-center"
                >
                  {{ config.prechat_form.title }}
                </div>
                <div v-for="field in prechatFields" :key="field.key" class="space-y-2">
                  <div v-if="field.type === 'checkbox'" class="flex items-start gap-3">
                    <div class="size-4 mt-0.5 rounded border border-input shrink-0"></div>
                    <label class="text-sm font-medium">
                      {{ field.label
                      }}<span v-if="field.required" class="text-destructive"> *</span>
                    </label>
                  </div>
                  <template v-else-if="field.type === 'phone'">
                    <label class="text-sm font-medium">
                      {{ field.label
                      }}<span v-if="field.required" class="text-destructive"> *</span>
                    </label>
                    <div class="flex items-stretch">
                      <div
                        class="flex items-center gap-2 px-3 rounded-l-md border border-r-0 border-input text-sm text-muted-foreground"
                      >
                        <span>{{ $t('globals.terms.select') }}</span>
                        <ChevronDown :size="14" class="opacity-50" />
                      </div>
                      <Input
                        type="tel"
                        :placeholder="field.placeholder || ''"
                        readonly
                        class="rounded-l-none"
                      />
                    </div>
                  </template>
                  <template v-else>
                    <label class="text-sm font-medium">
                      {{ field.label
                      }}<span v-if="field.required" class="text-destructive"> *</span>
                    </label>
                    <Input
                      :type="field.type === 'number' ? 'number' : 'text'"
                      :placeholder="field.placeholder || ''"
                      readonly
                    />
                  </template>
                </div>
                <div class="space-y-2">
                  <label class="text-sm font-medium">
                    {{ $t('globals.terms.message', 1) }}<span class="text-destructive"> *</span>
                  </label>
                  <Textarea
                    :placeholder="$t('globals.terms.typeMessage')"
                    rows="3"
                    readonly
                    class="resize-none"
                  />
                </div>
              </div>
              <div class="p-4 border-t border-border shrink-0">
                <Button type="button" class="w-full">{{
                  $t('widget.prechatForm.startChat')
                }}</Button>
              </div>
            </template>

            <!-- Message thread -->
            <template v-else>
              <div class="flex-1 min-h-0 overflow-y-auto p-4 flex flex-col gap-4">
                <div v-if="config.chat_introduction" class="text-center">
                  <p class="text-muted-foreground text-sm">{{ config.chat_introduction }}</p>
                </div>
                <div
                  v-if="config.notice_banner?.enabled && config.notice_banner?.text"
                  class="bg-secondary/30 border border-secondary rounded-lg p-4"
                >
                  <div class="flex items-center justify-center gap-3">
                    <AlertTriangle class="shrink-0 text-secondary-foreground" :size="16" />
                    <p class="text-sm text-secondary-foreground">{{ config.notice_banner.text }}</p>
                  </div>
                </div>
                <template v-if="openedExisting">
                  <div
                    v-for="m in sampleMessages"
                    :key="m.id"
                    class="flex flex-col"
                    :class="m.type === 'user' ? 'items-end' : 'items-start'"
                  >
                    <div
                      class="max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 break-words"
                      :class="
                        m.type === 'user'
                          ? 'bg-primary text-primary-foreground rounded-br-sm'
                          : 'bg-muted text-foreground rounded-bl-sm'
                      "
                    >
                      {{ m.content }}
                    </div>
                    <div v-if="m.type === 'agent'" class="text-[10px] text-muted-foreground mt-1">
                      {{ config.brand_name || $t('globals.terms.name') }}
                    </div>
                  </div>
                </template>
              </div>

              <!-- Message input -->
              <div class="border-t border-border shrink-0">
                <div class="p-2">
                  <div class="border border-input rounded-lg bg-background">
                    <div class="p-2">
                      <textarea
                        :placeholder="$t('globals.terms.typeMessage')"
                        rows="1"
                        readonly
                        class="w-full resize-none border-0 bg-transparent p-0 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
                      ></textarea>
                    </div>
                    <div class="flex justify-between items-center px-2 pb-2">
                      <div class="flex items-center gap-2 text-muted-foreground">
                        <Paperclip v-if="config.features?.file_upload" :size="18" />
                        <Smile v-if="config.features?.emoji" :size="18" />
                      </div>
                      <div
                        class="flex items-center justify-center h-9 w-9 rounded-full bg-primary text-primary-foreground"
                      >
                        <ArrowUp :size="16" />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </template>

            <div
              v-if="config.show_powered_by !== false"
              class="flex items-center justify-center pb-1.5 shrink-0"
            >
              <span class="text-[10px] text-muted-foreground/70"
                >Powered by <span class="font-medium">libredesk</span></span
              >
            </div>
          </template>

          <!-- Home + Messages (share the bottom nav) -->
          <template v-else>
            <div class="flex-1 min-h-0 relative">
              <!-- Home -->
              <div v-if="view === 'home'" class="h-full overflow-y-auto flex flex-col">
                <div class="relative" :style="headerStyle">
                  <div class="p-6">
                    <img
                      v-if="config.logo_url"
                      :src="config.logo_url"
                      :alt="config.brand_name"
                      class="max-h-7 max-w-full"
                    />
                    <div class="mt-16 font-bold text-3xl leading-tight" :class="textColorClass">
                      <h2 class="break-words">{{ parsedGreeting }}</h2>
                      <p class="mt-1 font-semibold" :class="subTextColorClass">
                        {{ parsedIntroduction }}
                      </p>
                    </div>
                  </div>
                  <div class="relative z-10 px-4 pb-4">
                    <Button
                      type="button"
                      class="w-full flex items-center justify-center gap-1"
                      @click="startNew"
                    >
                      {{ startButtonText }}
                      <ArrowRight :size="16" />
                    </Button>
                  </div>
                  <div
                    v-if="showFade"
                    class="absolute bottom-0 left-0 right-0 h-16 pointer-events-none"
                    :style="fadeStyle"
                  ></div>
                </div>

                <div v-if="homeApps.length" class="flex flex-col gap-3 p-4 bg-background">
                  <template v-for="(item, index) in homeApps" :key="index">
                    <Card
                      v-if="item.type === 'announcement'"
                      class="overflow-hidden rounded-md hover:bg-accent transition-colors"
                    >
                      <img
                        v-if="item.image_url"
                        :src="item.image_url"
                        :alt="item.title"
                        class="w-full h-auto"
                      />
                      <CardContent class="p-3 text-sm">
                        <div class="font-bold">
                          {{ item.title || $t('globals.terms.announcement') }}
                        </div>
                        <div v-if="item.description" class="text-muted-foreground mt-1">
                          {{ item.description }}
                        </div>
                      </CardContent>
                    </Card>
                    <Card v-else class="rounded-md hover:bg-accent transition-colors">
                      <CardContent class="p-4">
                        <div class="flex justify-between items-center">
                          <span class="text-sm text-primary font-medium">{{
                            item.text || item.url
                          }}</span>
                          <ExternalLink :size="18" class="text-muted-foreground" />
                        </div>
                      </CardContent>
                    </Card>
                  </template>
                </div>
              </div>

              <!-- Messages -->
              <div v-else class="h-full flex flex-col relative">
                <div class="flex items-center justify-center p-4 border-b border-border shrink-0">
                  <h3 class="text-base font-semibold text-foreground">
                    {{ $t('globals.terms.message', 2) }}
                  </h3>
                </div>
                <div class="flex-1 overflow-y-auto pb-20">
                  <div
                    class="p-4 border-b border-border hover:bg-accent/50 cursor-pointer transition-colors"
                    @click="openExisting"
                  >
                    <div class="flex items-center gap-3">
                      <div
                        class="size-10 rounded-full bg-secondary text-secondary-foreground flex items-center justify-center overflow-hidden shrink-0"
                      >
                        <img :src="launcherLogo" alt="" class="w-full h-full object-cover" />
                      </div>
                      <div class="flex-1 min-w-0">
                        <div class="text-sm font-medium text-foreground mb-0.5">
                          {{ config.brand_name || $t('globals.terms.name') }}
                        </div>
                        <p class="text-sm text-muted-foreground truncate">
                          {{ sampleLastMessage }}
                        </p>
                      </div>
                      <ChevronRight class="w-4 h-4 text-muted-foreground shrink-0" />
                    </div>
                  </div>
                </div>
                <div class="absolute bottom-0 inset-x-0">
                  <div
                    class="h-20 bg-gradient-to-t from-background via-background/80 to-transparent pointer-events-none"
                  ></div>
                  <div class="absolute bottom-4 inset-x-0 mx-auto w-fit z-10">
                    <Button type="button" @click="startNew">{{ messagesButtonText }}</Button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Bottom nav -->
            <div class="grid grid-cols-2 border-t border-border bg-background shrink-0">
              <button
                type="button"
                class="flex flex-col items-center gap-1 py-2"
                :class="view === 'home' ? 'text-foreground' : 'text-muted-foreground'"
                @click="view = 'home'"
              >
                <House class="w-5 h-5" />
                <span class="text-xs font-medium">{{ $t('globals.terms.home') }}</span>
              </button>
              <button
                type="button"
                class="flex flex-col items-center gap-1 py-2"
                :class="view === 'messages' ? 'text-foreground' : 'text-muted-foreground'"
                @click="view = 'messages'"
              >
                <MessagesSquare class="w-5 h-5" />
                <span class="text-xs font-medium">{{ $t('globals.terms.message', 2) }}</span>
              </button>
            </div>
          </template>
        </div>
      </div>
    </transition>

    <!-- Launcher -->
    <button
      type="button"
      class="absolute flex items-center justify-center rounded-full shadow-lg transition-transform hover:scale-105"
      :style="launcherStyle"
      @click="open = !open"
    >
      <ChevronDown v-if="open" :size="26" :style="{ color: launcherIconColor }" />
      <img v-else :src="launcherLogo" alt="" class="w-full h-full rounded-full object-cover" />
    </button>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardContent } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  House,
  MessagesSquare,
  ArrowRight,
  ArrowLeft,
  ArrowUp,
  ExternalLink,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  Paperclip,
  Smile,
  Maximize2
} from 'lucide-vue-next'
import { hexToHSL, getContrastingHSL } from '@shared-ui/utils/color'
import { renderTemplate } from '@shared-ui/utils/string'

const DEFAULT_LAUNCHER_LOGO = '/static/public/launcher-logo.png'
const HEX_COLOR = /^#([0-9a-f]{6}|[0-9a-f]{3})$/i
const LAUNCHER_SIZE = 52

const props = defineProps({
  config: {
    type: Object,
    default: () => ({})
  }
})

const { t } = useI18n()

const open = ref(true)
const view = ref('home')
// The preview can't send/receive, so a sample conversation stands in for a real one.
const openedExisting = ref(false)

const startNew = () => {
  openedExisting.value = false
  view.value = 'chat'
}
const openExisting = () => {
  openedExisting.value = true
  view.value = 'chat'
}

const isDark = computed(() => Boolean(props.config.dark_mode))

const primaryStyle = computed(() => {
  const primary = props.config.colors?.primary
  if (!HEX_COLOR.test(primary)) return {}
  return {
    '--primary': hexToHSL(primary),
    '--primary-foreground': getContrastingHSL(primary)
  }
})

const parsedGreeting = computed(() =>
  renderTemplate(props.config.greeting_message || '', { firstName: '', lastName: '' })
)
const parsedIntroduction = computed(() =>
  renderTemplate(props.config.introduction_message || '', { firstName: '', lastName: '' })
)

const launcherLogo = computed(() => props.config.launcher?.logo_url || DEFAULT_LAUNCHER_LOGO)

const replyExpectation = computed(() =>
  props.config.show_office_hours_in_chat ? props.config.chat_reply_expectation_message : ''
)

const headerStyle = computed(() => {
  const bg = props.config.home_screen?.background
  if (!bg?.type) return {}
  switch (bg.type) {
    case 'solid':
      return bg.color ? { backgroundColor: bg.color } : {}
    case 'gradient':
      return bg.gradient_start && bg.gradient_end
        ? { background: `linear-gradient(to bottom, ${bg.gradient_start}, ${bg.gradient_end})` }
        : {}
    case 'image':
      return bg.image_url
        ? {
            backgroundImage: `url(${bg.image_url})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center'
          }
        : {}
    default:
      return {}
  }
})

const headerTextColor = computed(() => props.config.home_screen?.header_text_color)
const textColorClass = computed(() => {
  if (headerTextColor.value === 'black') return 'text-black'
  if (headerTextColor.value === 'white') return 'text-white'
  return ''
})
const subTextColorClass = computed(() => {
  if (headerTextColor.value === 'black') return 'text-black/70'
  if (headerTextColor.value === 'white') return 'text-white/70'
  return 'text-muted-foreground'
})

const showFade = computed(
  () =>
    Boolean(props.config.home_screen?.background?.type) &&
    Boolean(props.config.home_screen?.fade_background)
)
const fadeStyle = { background: 'linear-gradient(to bottom, transparent, hsl(var(--background)))' }

const startButtonText = computed(
  () =>
    props.config.visitors?.start_conversation_button_text ||
    props.config.users?.start_conversation_button_text ||
    t('globals.messages.sendUsMessage')
)
const messagesButtonText = computed(
  () =>
    props.config.users?.start_conversation_button_text || t('globals.messages.startNewConversation')
)

const sampleMessages = computed(() => [
  {
    id: 1,
    type: 'agent',
    content: parsedIntroduction.value || t('globals.messages.sendUsMessage')
  },
  { id: 2, type: 'user', content: t('admin.inbox.livechat.preview.sampleUserMessage') },
  { id: 3, type: 'agent', content: t('admin.inbox.livechat.preview.sampleAgentReply') }
])
const sampleLastMessage = computed(
  () => sampleMessages.value[sampleMessages.value.length - 1].content
)

const homeApps = computed(() => props.config.home_apps || [])

const prechatFields = computed(() =>
  (props.config.prechat_form?.fields || [])
    .filter((f) => f.enabled)
    .sort((a, b) => (a.order || 0) - (b.order || 0))
)
const showPrechat = computed(
  () => Boolean(props.config.prechat_form?.enabled) && prechatFields.value.length > 0
)

const onLeft = computed(() => props.config.launcher?.position === 'left')

// Spacing is clamped so large values still render inside the preview stage.
const clampSpacing = (value) => Math.min(Number(value) || 20, 40)
const clampedSide = computed(() => clampSpacing(props.config.launcher?.spacing?.side))
const clampedBottom = computed(() => clampSpacing(props.config.launcher?.spacing?.bottom))
const windowBottom = computed(() => clampedBottom.value + LAUNCHER_SIZE + 12)

const launcherColor = computed(() => {
  const c = props.config.launcher?.color
  if (HEX_COLOR.test(c)) return c
  const primary = props.config.colors?.primary
  return HEX_COLOR.test(primary) ? primary : '#000000'
})
const launcherIconColor = computed(() => `hsl(${getContrastingHSL(launcherColor.value)})`)
const launcherStyle = computed(() => ({
  width: LAUNCHER_SIZE + 'px',
  height: LAUNCHER_SIZE + 'px',
  backgroundColor: launcherColor.value,
  bottom: clampedBottom.value + 'px',
  [onLeft.value ? 'left' : 'right']: clampedSide.value + 'px'
}))
</script>

<style scoped>
.ld-preview-window-enter-active,
.ld-preview-window-leave-active {
  transition:
    opacity 0.15s ease,
    transform 0.15s ease;
}
.ld-preview-window-enter-from,
.ld-preview-window-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
