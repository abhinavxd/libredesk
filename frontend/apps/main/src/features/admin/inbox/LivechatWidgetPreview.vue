<template>
  <div
    class="relative w-full h-[calc(100vh-13rem)] min-h-[620px] max-h-[880px] rounded-xl border border-border bg-muted overflow-hidden"
  >
    <!-- Widget window, themed independently of the admin app. -->
    <transition name="ld-preview-window">
      <div
        v-if="open"
        class="absolute top-4 inset-x-0 mx-auto max-w-[400px]"
        :style="{ bottom: windowBottom + 'px' }"
      >
        <div
          class="libredesk-widget-preview flex flex-col h-full bg-background text-foreground rounded-2xl overflow-hidden border border-border"
          :style="[primaryStyle, { boxShadow: IFRAME_BOX_SHADOW }]"
          :class="isDark ? 'dark' : 'light'"
        >
          <!-- Chat view -->
          <template v-if="view === 'chat'">
            <!-- Chat header -->
            <div class="flex items-center p-2 border-b border-border gap-3 shrink-0">
              <button
                type="button"
                :aria-label="$t('globals.messages.back')"
                :class="BACK_BUTTON_CLASS"
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
                      <span class="inline-block w-2 h-2 rounded-full bg-success"></span>
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
                <div v-if="prechatConfig.title" class="text-xl text-foreground mb-2 text-center">
                  {{ prechatConfig.title }}
                </div>
                <div v-for="field in prechatFields" :key="field.key" class="space-y-2">
                  <div v-if="field.type === 'checkbox'" class="flex items-start gap-3">
                    <div class="size-4 mt-0.5 rounded-md border border-input shrink-0"></div>
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
                <div
                  v-if="!openedExisting && quickReplies.length"
                  class="flex flex-wrap justify-end gap-2 mt-auto"
                >
                  <Button
                    v-for="reply in quickReplies"
                    :key="reply"
                    type="button"
                    variant="outline"
                    size="sm"
                    class="h-auto max-w-full rounded-full py-1.5 text-left whitespace-normal shadow-md"
                    >{{ reply }}</Button
                  >
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
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-foreground'
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
                  <div class="border border-input rounded-md bg-background">
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

          <template v-else>
            <div class="flex-1 min-h-0 relative">
              <div v-if="view === 'home'" class="relative h-full bg-background">
                <div
                  class="pointer-events-none absolute inset-x-0 top-0"
                  :style="[headerStyle, { height: `${headerHeight}px` }]"
                >
                  <div
                    v-if="showFade"
                    class="absolute inset-x-0 bottom-0 h-20"
                    :style="fadeStyle"
                  ></div>
                </div>
                <div class="relative h-full overflow-y-auto">
                  <div ref="headerRef">
                    <div class="px-7 pb-7 pt-7">
                      <img
                        v-if="config.logo_url"
                        :src="config.logo_url"
                        :alt="config.brand_name"
                        class="max-h-7 max-w-full"
                      />
                      <div class="mt-20" :class="textColorClass">
                        <h2
                          class="text-3xl font-semibold leading-tight tracking-tight break-words"
                          :class="subTextColorClass"
                        >
                          {{ parsedGreeting }}
                        </h2>
                        <p
                          class="mt-2 text-3xl font-semibold leading-tight tracking-tight break-words"
                        >
                          {{ parsedIntroduction }}
                        </p>
                      </div>
                    </div>
                    <div v-if="canStartConversation" class="relative z-10 px-4 pb-5">
                      <Button
                        type="button"
                        size="lg"
                        class="w-full font-semibold shadow-md"
                        @click="startNew"
                      >
                        {{ startButtonText }}
                        <ArrowRight :size="16" aria-hidden="true" />
                      </Button>
                    </div>
                  </div>

                  <div v-if="homeItems.length" class="flex flex-col gap-3 px-4 pt-1 pb-5">
                    <template v-for="(item, index) in homeItems" :key="index">
                      <section
                        v-if="item.type === 'help'"
                        class="space-y-1 rounded-xl border border-border/80 bg-card p-2 shadow-sm"
                      >
                        <Button
                          type="button"
                          variant="outline"
                          class="h-10 w-full justify-between px-2"
                          @click="view = 'help'"
                        >
                          {{ $t('widget.searchArticles') }}
                          <Search class="size-4" aria-hidden="true" />
                        </Button>
                        <Button
                          v-for="article in featuredArticles"
                          :key="article.id"
                          type="button"
                          variant="ghost"
                          class="w-full h-auto justify-start px-2 py-1.5 text-left whitespace-normal"
                          @click="view = 'help'"
                        >
                          {{ article.title }}
                        </Button>
                      </section>
                      <Card
                        v-else-if="item.type === 'announcement'"
                        class="overflow-hidden rounded-xl border-border/80 shadow-sm transition-[background-color,box-shadow] can-hover:hover:bg-accent can-hover:hover:shadow-md"
                      >
                        <img
                          v-if="item.image_url"
                          :src="item.image_url"
                          :alt="item.title"
                          class="w-full h-auto"
                        />
                        <CardContent class="p-4 text-sm">
                          <div class="font-semibold leading-snug">
                            {{ item.title || $t('globals.terms.announcement') }}
                          </div>
                          <div
                            v-if="item.description"
                            class="mt-1 text-muted-foreground leading-relaxed"
                          >
                            {{ item.description }}
                          </div>
                        </CardContent>
                      </Card>
                      <Card
                        v-else
                        class="rounded-xl border-border/80 shadow-sm transition-[background-color,box-shadow] can-hover:hover:bg-accent can-hover:hover:shadow-md"
                      >
                        <CardContent class="flex items-center gap-3 p-4">
                          <span
                            class="min-w-0 flex-1 text-sm font-medium leading-snug text-foreground"
                            >{{ item.text || item.url }}</span
                          >
                          <ExternalLink
                            :size="15"
                            class="shrink-0 text-muted-foreground"
                            aria-hidden="true"
                          />
                        </CardContent>
                      </Card>
                    </template>
                  </div>
                </div>
              </div>

              <!-- Help -->
              <div v-else-if="view === 'help'" class="h-full flex flex-col">
                <header class="flex items-center gap-2 border-b border-border p-3 shrink-0">
                  <button
                    v-if="helpCollection"
                    type="button"
                    :aria-label="$t('globals.messages.back')"
                    :class="BACK_BUTTON_CLASS"
                    @click="helpPath.pop()"
                  >
                    <ArrowLeft :size="16" />
                  </button>
                  <h3 class="font-medium truncate">{{ $t('globals.terms.help') }}</h3>
                </header>
                <div class="relative p-3 border-b border-border shrink-0">
                  <Search
                    class="absolute left-6 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                    aria-hidden="true"
                  />
                  <Input class="pl-10" :placeholder="$t('widget.searchArticles')" readonly />
                </div>
                <div class="flex-1 min-h-0 overflow-y-auto">
                  <p
                    v-if="!helpCollections.length && !helpArticles.length"
                    class="p-4 text-sm text-muted-foreground"
                  >
                    {{ $t('widget.noArticles') }}
                  </p>
                  <HelpCollectionList
                    v-else
                    :collection="helpCollection"
                    :collections="helpCollections"
                    :articles="helpArticles"
                    @collection="helpPath.push($event)"
                  />
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
                  <button
                    type="button"
                    class="w-full p-4 border-b border-border hover:bg-accent/50 cursor-pointer transition-colors text-left"
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
                  </button>
                </div>
                <div v-if="canStartFromMessages" class="absolute bottom-0 inset-x-0">
                  <div
                    class="h-20 bg-gradient-to-t from-background via-background/80 to-transparent pointer-events-none"
                  ></div>
                  <div class="absolute bottom-4 inset-x-0 mx-auto w-fit z-10">
                    <Button type="button" @click="startNew">{{ messagesButtonText }}</Button>
                  </div>
                </div>
              </div>
            </div>

            <div class="flex border-t border-border bg-background shrink-0">
              <button
                type="button"
                :class="[
                  PREVIEW_NAV_TAB_CLASS,
                  view === 'home' ? 'text-primary' : 'text-muted-foreground'
                ]"
                @click="view = 'home'"
              >
                <House class="w-5 h-5" aria-hidden="true" />
                <span class="text-xs font-medium">{{ $t('globals.terms.home') }}</span>
              </button>
              <button
                type="button"
                :class="[
                  PREVIEW_NAV_TAB_CLASS,
                  view === 'messages' ? 'text-primary' : 'text-muted-foreground'
                ]"
                @click="view = 'messages'"
              >
                <MessagesSquare class="w-5 h-5" aria-hidden="true" />
                <span class="text-xs font-medium">{{ $t('globals.terms.message', 2) }}</span>
              </button>
              <button
                v-if="showHelpTab"
                type="button"
                :class="[
                  PREVIEW_NAV_TAB_CLASS,
                  view === 'help' ? 'text-primary' : 'text-muted-foreground'
                ]"
                @click="view = 'help'"
              >
                <CircleQuestionMark class="w-5 h-5" aria-hidden="true" />
                <span class="text-xs font-medium">{{ $t('globals.terms.help') }}</span>
              </button>
            </div>
          </template>
        </div>
      </div>
    </transition>

    <!-- Proactive message, mirrors the card widget.js renders above the launcher. -->
    <div v-if="campaign && !open" class="absolute" :style="campaignStyle">
      <div
        class="flex items-start border border-border rounded-xl bg-background text-foreground overflow-hidden shadow-lg"
        :class="isDark ? 'dark' : 'light'"
      >
        <div class="flex items-start gap-2.5 flex-1 min-w-0 p-3 text-left">
          <img
            v-if="campaign.avatar"
            :src="campaign.avatar"
            alt=""
            class="size-7 rounded-full object-cover shrink-0"
          />
          <span class="min-w-0">
            <span class="block font-semibold text-sm">{{ campaign.sender }}</span>
            <span class="block text-sm line-clamp-3">{{ campaignText }}</span>
          </span>
        </div>
        <div class="flex items-center justify-center size-11 text-muted-foreground shrink-0">
          <X :size="16" />
        </div>
      </div>
    </div>

    <!-- Launcher -->
    <button
      type="button"
      :aria-label="open ? $t('globals.messages.closeChat') : $t('globals.messages.openChat')"
      class="absolute flex items-center justify-center rounded-full transition-transform hover:scale-105"
      :style="launcherStyle"
      @click="open = !open"
    >
      <ChevronDown
        v-if="open"
        :size="26"
        :style="{ color: launcherIconColor }"
        aria-hidden="true"
      />
      <img
        v-else
        :src="launcherLogo"
        alt=""
        :class="launcherIconFull ? 'rounded-full object-cover' : 'object-contain'"
        :style="launcherIconStyle"
      />
    </button>
  </div>
</template>

<script setup>
const IFRAME_BOX_SHADOW = 'rgba(9, 14, 21, 0.9) 0px 5px 40px 0px'
const LAUNCHER_DROP_SHADOW =
  'drop-shadow(rgba(9, 14, 21, 0.54) 0px 1px 6px) drop-shadow(rgba(9, 14, 21, 0.9) 0px 2px 32px)'
const BACK_BUTTON_CLASS =
  'flex items-center justify-center size-8 rounded-md hover:bg-accent text-foreground'
const PREVIEW_NAV_TAB_CLASS = 'flex w-full flex-col items-center justify-center gap-1 py-4'

import { computed, ref, watch } from 'vue'
import { useElementSize } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardContent } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  House,
  MessagesSquare,
  CircleQuestionMark,
  ArrowRight,
  ArrowLeft,
  ArrowUp,
  ExternalLink,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  Paperclip,
  Smile,
  Maximize2,
  Search,
  X
} from 'lucide-vue-next'
import HelpCollectionList from '@shared-ui/components/HelpCollectionList.vue'
import { hexToHSL, getContrastingHSL } from '@shared-ui/utils/color'
import { renderTemplate } from '@shared-ui/utils/string'

const DEFAULT_LAUNCHER_LOGO = '/static/public/launcher-logo.png'
const HEX_COLOR = /^#([0-9a-f]{6}|[0-9a-f]{3})$/i
const LAUNCHER_SIZE = 52
const DEFAULT_LAUNCHER_ICON_SCALE = 100
const SNIPPET_LENGTH = 240

const props = defineProps({
  userType: {
    type: String,
    default: 'visitors'
  },
  config: {
    type: Object,
    default: () => ({})
  }
})

const { t } = useI18n()

const open = ref(true)
const headerRef = ref(null)
const { height: headerHeight } = useElementSize(headerRef)

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

const isDark = computed(() => Boolean(props.config.dark))

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
const fadeStyle = {
  background:
    'linear-gradient(to bottom, transparent 0%, hsl(var(--background) / 0.08) 20%, hsl(var(--background) / 0.32) 45%, hsl(var(--background) / 0.72) 72%, hsl(var(--background)) 100%)'
}

const userTypeConfig = computed(() => props.config[props.userType] || {})

const canStartConversation = computed(() => Boolean(userTypeConfig.value.allow_start_conversation))

// The messages tab always previews an existing conversation, so it follows the multiple rule too.
const canStartFromMessages = computed(
  () => canStartConversation.value && userTypeConfig.value.prevent_multiple_conversations !== true
)

const startButtonText = computed(
  () => userTypeConfig.value.start_conversation_button_text || t('globals.messages.sendUsMessage')
)
const messagesButtonText = computed(
  () =>
    userTypeConfig.value.start_conversation_button_text ||
    t('globals.messages.startNewConversation')
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

const helpConfig = computed(() => props.config.help || {})
const helpAudience = computed(() => helpConfig.value[props.userType] || {})
const helpEnabled = computed(() => Boolean(helpConfig.value.help_center_id))
const showHelpTab = computed(() => helpEnabled.value && Boolean(helpAudience.value.tab))

const featuredArticles = computed(() =>
  (helpConfig.value.featured_ids || [])
    .map((id) => (props.config.help_articles || []).find((article) => article.id === id))
    .filter(Boolean)
)

const helpPath = ref([])
const helpCollection = computed(() => helpPath.value.at(-1))
const helpCollections = computed(
  () => helpCollection.value?.children || props.config.help_tree || []
)
const helpArticles = computed(() => helpCollection.value?.articles || [])
watch(
  () => helpConfig.value.help_center_id,
  () => {
    helpPath.value = []
  }
)

const homeItems = computed(() =>
  (props.config.home_apps || []).filter((item) => item.type !== 'help' || helpEnabled.value)
)

const campaign = computed(() => props.config.preview_campaign || null)
watch(
  () => campaign.value?.id ?? null,
  (campaignId) => {
    if (campaignId !== null) open.value = false
  }
)
const campaignText = computed(() => campaign.value?.message?.slice(0, SNIPPET_LENGTH) || '')
const quickReplies = computed(() => {
  const replies = userTypeConfig.value.quick_replies
  if (Array.isArray(replies)) return replies
  return typeof replies === 'string'
    ? replies
        .split('\n')
        .map((reply) => reply.trim())
        .filter(Boolean)
    : []
})

const prechatConfig = computed(() => {
  const shared = props.config.prechat_form || {}
  const audience = shared[props.userType]
  return audience && typeof audience === 'object' ? audience : shared
})
const prechatFields = computed(() =>
  (prechatConfig.value.fields || [])
    .filter((f) => f.enabled)
    .sort((a, b) => (a.order || 0) - (b.order || 0))
)
const showPrechat = computed(
  () =>
    Boolean(props.config.prechat_form?.enabled) &&
    (prechatConfig.value.enabled ?? true) &&
    prechatFields.value.length > 0
)

const onLeft = computed(() => props.config.launcher?.position === 'left')

// Spacing is clamped so large values still render inside the preview stage.
const clampSpacing = (value) => Math.min(Number(value) || 20, 40)
const clampedSide = computed(() => clampSpacing(props.config.launcher?.spacing?.side))
const clampedBottom = computed(() => clampSpacing(props.config.launcher?.spacing?.bottom))
const windowBottom = computed(() => clampedBottom.value + LAUNCHER_SIZE + 12)
const launcherIconScale = computed(
  () => Number(props.config.launcher?.icon_scale) || DEFAULT_LAUNCHER_ICON_SCALE
)
const launcherIconFull = computed(() => launcherIconScale.value >= 100)
const launcherIconStyle = computed(() => ({
  width: launcherIconScale.value + '%',
  height: launcherIconScale.value + '%'
}))

const launcherColor = computed(() => {
  const c = props.config.launcher?.color
  if (HEX_COLOR.test(c)) return c
  const primary = props.config.colors?.primary
  return HEX_COLOR.test(primary) ? primary : '#000000'
})
const launcherIconColor = computed(() => `hsl(${getContrastingHSL(launcherColor.value)})`)
const campaignStyle = computed(() => ({
  bottom: windowBottom.value + 'px',
  [onLeft.value ? 'left' : 'right']: clampedSide.value + 'px',
  width: `min(280px, calc(100% - ${clampedSide.value * 2}px))`
}))
const launcherStyle = computed(() => ({
  filter: LAUNCHER_DROP_SHADOW,
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
