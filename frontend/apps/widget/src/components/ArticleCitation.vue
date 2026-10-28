<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'

defineProps({
  number: { type: String, required: true },
  title: { type: String, required: true },
  href: { type: String, required: true }
})

const { t } = useI18n()
const open = ref(false)
const articleLink = ref(null)
let hoverOpened = false
let closeTimer

const cancelClose = () => clearTimeout(closeTimer)

const openOnHover = (event) => {
  cancelClose()
  if (event.pointerType !== 'mouse' || open.value) return
  hoverOpened = true
  open.value = true
}

const closeOnLeave = () => {
  if (hoverOpened) closeTimer = setTimeout(() => (open.value = false), 150)
}

const activate = () => {
  cancelClose()
  hoverOpened = false
}

const focusArticle = (event) => {
  event.preventDefault()
  if (!hoverOpened) articleLink.value?.$el?.focus()
}

const preventHoverFocus = (event) => {
  if (hoverOpened) event.preventDefault()
}

onBeforeUnmount(cancelClose)
</script>

<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        type="button"
        variant="secondary"
        :aria-label="`${number} ${title}`"
        class="mx-0.5 h-auto rounded-full bg-foreground/10 px-1.5 py-0.5 text-sm leading-none !text-foreground hover:bg-foreground/15"
        @pointerenter="openOnHover"
        @pointerleave="closeOnLeave"
        @click="activate"
      >
        {{ number }}
      </Button>
    </PopoverTrigger>
    <PopoverContent
      side="top"
      :side-offset="6"
      class="flex w-auto max-w-60 flex-col items-start gap-1 p-3 text-xs"
      @pointerenter="cancelClose"
      @pointerleave="closeOnLeave"
      @open-auto-focus="focusArticle"
      @close-auto-focus="preventHoverFocus"
    >
      <span class="flex items-center gap-1 text-muted-foreground">
        <BookOpen class="h-3 w-3" aria-hidden="true" />
        {{ t('globals.terms.helpCenterArticle') }}
      </span>
      <Button
        ref="articleLink"
        as="a"
        variant="link"
        :href="href"
        target="_blank"
        rel="noopener noreferrer"
        :aria-label="title"
        class="h-auto justify-start whitespace-normal break-words p-0 text-left text-xs text-popover-foreground no-underline hover:text-popover-foreground hover:underline"
      >
        {{ title }}
      </Button>
    </PopoverContent>
  </Popover>
</template>
