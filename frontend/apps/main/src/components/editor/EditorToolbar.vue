<template>
  <div class="flex flex-wrap gap-1 items-center">
    <DropdownMenu v-if="showAi && aiPromptStore.prompts.length > 0">
      <DropdownMenuTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          class="flex items-center justify-center"
          :disabled="isGenerating"
        >
          <span class="flex items-center">
            <span class="text-medium">AI</span>
            <Bot size="14" class="ml-1" />
            <ChevronDown class="w-4 h-4 ml-2" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          v-for="prompt in aiPromptStore.prompts"
          :key="prompt.key"
          @select="complete(prompt.key)"
        >
          {{ prompt.title }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <DropdownMenu v-if="showArticleTools">
      <DropdownMenuTrigger as-child>
        <Button type="button" size="sm" variant="ghost" class="flex items-center justify-center">
          <span class="flex items-center">
            <Type size="14" />
            <span class="ml-1 text-xs font-medium">{{ getCurrentHeadingText() }}</span>
            <ChevronDown class="w-3 h-3 ml-1" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem @select="setParagraph">
          <span class="font-normal">{{ i18n._('Paragraph') }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem @select="() => setHeading(2)">
          <span class="text-xl font-bold">{{ i18n._('Heading {level}', { level: 2 }) }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem @select="() => setHeading(3)">
          <span class="text-base font-semibold">{{ i18n._('Heading {level}', { level: 3 }) }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem @select="() => setHeading(4)">
          <span class="text-sm font-semibold">{{ i18n._('Heading {level}', { level: 4 }) }}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n._('Bold')"
          @click.prevent="editor?.chain().focus().toggleBold().run()"
          :class="{ 'bg-secondary': editor?.isActive('bold') }"
        >
          <Bold size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n._('Bold') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n._('Italic')"
          @click.prevent="editor?.chain().focus().toggleItalic().run()"
          :class="{ 'bg-secondary': editor?.isActive('italic') }"
        >
          <Italic size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n._('Italic') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n._('Strikethrough')"
          @click.prevent="editor?.chain().focus().toggleStrike().run()"
          :class="{ 'bg-secondary': editor?.isActive('strike') }"
        >
          <Strikethrough size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n._('Strikethrough') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n._('Underline')"
          @click.prevent="editor?.chain().focus().toggleUnderline().run()"
          :class="{ 'bg-secondary': editor?.isActive('underline') }"
        >
          <UnderlineIcon size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n._('Underline') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n._('Bullet list')"
          @click.prevent="editor?.chain().focus().toggleBulletList().run()"
          :class="{ 'bg-secondary': editor?.isActive('bulletList') }"
        >
          <List size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n._('Bullet list') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n._('Numbered list')"
          @click.prevent="editor?.chain().focus().toggleOrderedList().run()"
          :class="{ 'bg-secondary': editor?.isActive('orderedList') }"
        >
          <ListOrdered size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n._('Numbered list') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="i18n.ngettext('Link', 'Links', 1)"
          @click.prevent="emit('openLink')"
          :class="{ 'bg-secondary': editor?.isActive('link') }"
        >
          <LinkIcon size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ i18n.ngettext('Link', 'Links', 1) }}</TooltipContent>
    </Tooltip>
    <template v-if="showArticleTools">
      <DropdownMenu v-if="editor?.isActive('codeBlock')">
        <Tooltip>
          <TooltipTrigger as-child>
            <DropdownMenuTrigger as-child>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                class="bg-secondary flex items-center"
                :aria-label="i18n._('Code block')"
              >
                <Code size="14" />
                <span class="ml-1 text-xs">{{ getCurrentLanguageLabel() }}</span>
                <ChevronDown class="w-3 h-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{{ i18n._('Code block') }}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent>
          <DropdownMenuItem
            class="text-destructive"
            @select="editor?.chain().focus().toggleCodeBlock().run()"
          >
            {{ i18n._('Remove') }}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <div class="max-h-64 overflow-y-auto">
            <DropdownMenuItem
              v-for="language in codeBlockLanguages"
              :key="language.label"
              @select="setCodeBlockLanguage(language.value)"
            >
              {{ language.label }}
            </DropdownMenuItem>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
      <Tooltip v-else>
        <TooltipTrigger as-child>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            :aria-label="i18n._('Code block')"
            @click.prevent="editor?.chain().focus().toggleCodeBlock().run()"
          >
            <Code size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._('Code block') }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            :aria-label="i18n._('Quote')"
            @click.prevent="editor?.chain().focus().toggleBlockquote().run()"
            :class="{ 'bg-secondary': editor?.isActive('blockquote') }"
          >
            <Quote size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._('Quote') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-for="a in alignments" :key="a.dir">
        <TooltipTrigger as-child>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            :aria-label="i18n._(a.label)"
            @click.prevent="editor?.chain().focus().setTextAlign(a.dir).run()"
            :class="{ 'bg-secondary': editor?.isActive({ textAlign: a.dir }) }"
          >
            <component :is="a.icon" size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._(a.label) }}</TooltipContent>
      </Tooltip>
      <DropdownMenu v-if="editor?.isActive('table')">
        <Tooltip>
          <TooltipTrigger as-child>
            <DropdownMenuTrigger as-child>
              <Button type="button" size="sm" variant="ghost" class="bg-secondary flex items-center" :aria-label="i18n._('Table')">
                <TableIcon size="14" />
                <ChevronDown class="w-3 h-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{{ i18n._('Table') }}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent>
          <DropdownMenuItem @select="editor?.chain().focus().addRowBefore().run()">{{ i18n._('Add row above') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().addRowAfter().run()">{{ i18n._('Add row below') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().addColumnBefore().run()">{{ i18n._('Add column left') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().addColumnAfter().run()">{{ i18n._('Add column right') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().deleteRow().run()">{{ i18n._('Delete row') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().deleteColumn().run()">{{ i18n._('Delete column') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleHeaderRow().run()">{{ i18n._('Toggle header row') }}</DropdownMenuItem>
          <DropdownMenuItem class="text-destructive" @select="editor?.chain().focus().deleteTable().run()">{{ i18n._('Delete table') }}</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Tooltip v-else>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="i18n._('Table')" @click.prevent="editor?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()">
            <TableIcon size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._('Table') }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="i18n._('Divider')" @click.prevent="editor?.chain().focus().setHorizontalRule().run()">
            <Minus size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._('Divider') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="enableInlineImages">
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="i18n.ngettext('Image', 'Images', 1)" @click.prevent="emit('openImage')">
            <ImageIcon size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n.ngettext('Image', 'Images', 1) }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="i18n._('YouTube video')" @click.prevent="emit('openYoutube')">
            <YoutubeIcon size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._('YouTube video') }}</TooltipContent>
      </Tooltip>
      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger as-child>
            <DropdownMenuTrigger as-child>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                class="flex items-center"
                :aria-label="i18n._('Callout')"
                :class="{ 'bg-secondary': editor?.isActive('callout') }"
              >
                <Info size="14" />
                <ChevronDown class="w-3 h-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{{ i18n._('Callout') }}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent>
          <template v-if="editor?.isActive('callout')">
            <DropdownMenuItem class="text-destructive" @select="editor?.chain().focus().unsetCallout().run()">{{ i18n._('Remove') }}</DropdownMenuItem>
            <DropdownMenuSeparator />
          </template>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('info').run()">{{ i18n._('Info') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('success').run()">{{ i18n._('Success') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('warning').run()">{{ i18n.ngettext('Warning', 'Warnings', 1) }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('danger').run()">{{ i18n._('Danger') }}</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="i18n._('Collapsible section')" @click.prevent="editor?.chain().focus().setDetails().run()">
            <ListCollapse size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ i18n._('Collapsible section') }}</TooltipContent>
      </Tooltip>
    </template>
  </div>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

import { ref } from 'vue'
import {
  ChevronDown,
  Bold,
  Italic,
  Strikethrough,
  Underline as UnderlineIcon,
  Bot,
  List,
  ListOrdered,
  Link as LinkIcon,
  Type,
  Code,
  Quote,
  Minus,
  Table as TableIcon,
  AlignLeft,
  AlignCenter,
  AlignRight,
  Info,
  ListCollapse,
  Image as ImageIcon,
  Youtube as YoutubeIcon
} from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useAiPromptStore } from '@main/stores/aiPrompt'
import { codeBlockLanguages } from './codeLanguages'

const props = defineProps({
  editor: { type: Object, default: null },
  showArticleTools: { type: Boolean, default: false },
  showAi: { type: Boolean, default: false },
  enableInlineImages: { type: Boolean, default: false }
})

const emit = defineEmits(['openLink', 'openYoutube', 'openImage', 'aiGenerationChange'])

const aiPromptStore = useAiPromptStore()
const isGenerating = ref(false)

const alignments = [
  { dir: 'left', icon: AlignLeft, label: msg({ message: 'Align left' }) },
  { dir: 'center', icon: AlignCenter, label: msg({ message: 'Align center' }) },
  { dir: 'right', icon: AlignRight, label: msg({ message: 'Align right' }) }
]

const setHeading = (level) => props.editor?.chain().focus().toggleHeading({ level }).run()
const setParagraph = () => props.editor?.chain().focus().setParagraph().run()

const setCodeBlockLanguage = (language) =>
  props.editor?.chain().focus().updateAttributes('codeBlock', { language }).run()

const getCurrentLanguageLabel = () => {
  const language = props.editor?.getAttributes('codeBlock').language
  return codeBlockLanguages.find((l) => l.value === language)?.label || language
}

const getCurrentHeadingText = () => {
  if (!props.editor) return 'P'
  for (let level = 2; level <= 4; level++) {
    if (props.editor.isActive('heading', { level })) return `H${level}`
  }
  return 'P'
}

const complete = async (promptKey) => {
  if (!props.editor || isGenerating.value) return

  isGenerating.value = true
  emit('aiGenerationChange', true)
  try {
    const content = await aiPromptStore.complete(promptKey, props.editor.getHTML())
    if (content !== null) props.editor.commands.setContent(content, true)
  } finally {
    isGenerating.value = false
    emit('aiGenerationChange', false)
  }
}
</script>
