<template>
  <div class="flex flex-wrap gap-1 items-center">
    <DropdownMenu v-if="aiPrompts.length > 0">
      <DropdownMenuTrigger as-child>
        <Button type="button" size="sm" variant="ghost" class="flex items-center justify-center">
          <span class="flex items-center">
            <span class="text-medium">AI</span>
            <Bot size="14" class="ml-1" />
            <ChevronDown class="w-4 h-4 ml-2" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          v-for="prompt in aiPrompts"
          :key="prompt.key"
          @select="emit('aiPrompt', prompt.key)"
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
          <span class="font-normal">{{ $t('editor.paragraph') }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem @select="() => setHeading(2)">
          <span class="text-xl font-bold">{{ $t('editor.heading', { level: 2 }) }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem @select="() => setHeading(3)">
          <span class="text-base font-semibold">{{ $t('editor.heading', { level: 3 }) }}</span>
        </DropdownMenuItem>
        <DropdownMenuItem @select="() => setHeading(4)">
          <span class="text-sm font-semibold">{{ $t('editor.heading', { level: 4 }) }}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('globals.terms.bold')"
          @click.prevent="editor?.chain().focus().toggleBold().run()"
          :class="{ 'bg-secondary': editor?.isActive('bold') }"
        >
          <Bold size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('globals.terms.bold') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('globals.terms.italic')"
          @click.prevent="editor?.chain().focus().toggleItalic().run()"
          :class="{ 'bg-secondary': editor?.isActive('italic') }"
        >
          <Italic size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('globals.terms.italic') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('editor.tooltip.strikethrough')"
          @click.prevent="editor?.chain().focus().toggleStrike().run()"
          :class="{ 'bg-secondary': editor?.isActive('strike') }"
        >
          <Strikethrough size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('editor.tooltip.strikethrough') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('editor.tooltip.underline')"
          @click.prevent="editor?.chain().focus().toggleUnderline().run()"
          :class="{ 'bg-secondary': editor?.isActive('underline') }"
        >
          <UnderlineIcon size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('editor.tooltip.underline') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('editor.tooltip.bulletList')"
          @click.prevent="editor?.chain().focus().toggleBulletList().run()"
          :class="{ 'bg-secondary': editor?.isActive('bulletList') }"
        >
          <List size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('editor.tooltip.bulletList') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('editor.tooltip.orderedList')"
          @click.prevent="editor?.chain().focus().toggleOrderedList().run()"
          :class="{ 'bg-secondary': editor?.isActive('orderedList') }"
        >
          <ListOrdered size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('editor.tooltip.orderedList') }}</TooltipContent>
    </Tooltip>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          :aria-label="$t('globals.terms.link')"
          @click.prevent="emit('openLink')"
          :class="{ 'bg-secondary': editor?.isActive('link') }"
        >
          <LinkIcon size="14" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{{ $t('globals.terms.link') }}</TooltipContent>
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
                :aria-label="$t('editor.tooltip.codeBlock')"
              >
                <Code size="14" />
                <span class="ml-1 text-xs">{{ getCurrentLanguageLabel() }}</span>
                <ChevronDown class="w-3 h-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{{ $t('editor.tooltip.codeBlock') }}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent>
          <DropdownMenuItem
            class="text-destructive"
            @select="editor?.chain().focus().toggleCodeBlock().run()"
          >
            {{ $t('globals.terms.remove') }}
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
            :aria-label="$t('editor.tooltip.codeBlock')"
            @click.prevent="editor?.chain().focus().toggleCodeBlock().run()"
          >
            <Code size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('editor.tooltip.codeBlock') }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            :aria-label="$t('editor.tooltip.blockquote')"
            @click.prevent="editor?.chain().focus().toggleBlockquote().run()"
            :class="{ 'bg-secondary': editor?.isActive('blockquote') }"
          >
            <Quote size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('editor.tooltip.blockquote') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-for="a in alignments" :key="a.dir">
        <TooltipTrigger as-child>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            :aria-label="$t(a.label)"
            @click.prevent="editor?.chain().focus().setTextAlign(a.dir).run()"
            :class="{ 'bg-secondary': editor?.isActive({ textAlign: a.dir }) }"
          >
            <component :is="a.icon" size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t(a.label) }}</TooltipContent>
      </Tooltip>
      <DropdownMenu v-if="editor?.isActive('table')">
        <Tooltip>
          <TooltipTrigger as-child>
            <DropdownMenuTrigger as-child>
              <Button type="button" size="sm" variant="ghost" class="bg-secondary flex items-center" :aria-label="$t('editor.tooltip.table')">
                <TableIcon size="14" />
                <ChevronDown class="w-3 h-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{{ $t('editor.tooltip.table') }}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent>
          <DropdownMenuItem @select="editor?.chain().focus().addRowBefore().run()">{{ $t('editor.table.addRowBefore') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().addRowAfter().run()">{{ $t('editor.table.addRowAfter') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().addColumnBefore().run()">{{ $t('editor.table.addColumnBefore') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().addColumnAfter().run()">{{ $t('editor.table.addColumnAfter') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().deleteRow().run()">{{ $t('editor.table.deleteRow') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().deleteColumn().run()">{{ $t('editor.table.deleteColumn') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleHeaderRow().run()">{{ $t('editor.table.toggleHeaderRow') }}</DropdownMenuItem>
          <DropdownMenuItem class="text-destructive" @select="editor?.chain().focus().deleteTable().run()">{{ $t('editor.table.deleteTable') }}</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Tooltip v-else>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="$t('editor.tooltip.table')" @click.prevent="editor?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()">
            <TableIcon size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('editor.tooltip.table') }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="$t('editor.tooltip.horizontalRule')" @click.prevent="editor?.chain().focus().setHorizontalRule().run()">
            <Minus size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('editor.tooltip.horizontalRule') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="enableInlineImages">
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="$t('globals.terms.image', 1)" @click.prevent="emit('openImage')">
            <ImageIcon size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('globals.terms.image', 1) }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="$t('editor.tooltip.youtube')" @click.prevent="emit('openYoutube')">
            <YoutubeIcon size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('editor.tooltip.youtube') }}</TooltipContent>
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
                :aria-label="$t('editor.tooltip.callout')"
                :class="{ 'bg-secondary': editor?.isActive('callout') }"
              >
                <Info size="14" />
                <ChevronDown class="w-3 h-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
          </TooltipTrigger>
          <TooltipContent>{{ $t('editor.tooltip.callout') }}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent>
          <template v-if="editor?.isActive('callout')">
            <DropdownMenuItem class="text-destructive" @select="editor?.chain().focus().unsetCallout().run()">{{ $t('globals.terms.remove') }}</DropdownMenuItem>
            <DropdownMenuSeparator />
          </template>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('info').run()">{{ $t('globals.terms.info') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('success').run()">{{ $t('globals.terms.success') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('warning').run()">{{ $t('globals.terms.warning') }}</DropdownMenuItem>
          <DropdownMenuItem @select="editor?.chain().focus().toggleCallout('danger').run()">{{ $t('globals.terms.danger') }}</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Tooltip>
        <TooltipTrigger as-child>
          <Button type="button" size="sm" variant="ghost" :aria-label="$t('editor.tooltip.collapsible')" @click.prevent="editor?.chain().focus().setDetails().run()">
            <ListCollapse size="14" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{{ $t('editor.tooltip.collapsible') }}</TooltipContent>
      </Tooltip>
    </template>
  </div>
</template>

<script setup>
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
import { codeBlockLanguages } from './codeLanguages'

const props = defineProps({
  editor: { type: Object, default: null },
  showArticleTools: { type: Boolean, default: false },
  aiPrompts: { type: Array, default: () => [] },
  enableInlineImages: { type: Boolean, default: false }
})

const emit = defineEmits(['openLink', 'openYoutube', 'openImage', 'aiPrompt'])

const alignments = [
  { dir: 'left', icon: AlignLeft, label: 'editor.tooltip.alignLeft' },
  { dir: 'center', icon: AlignCenter, label: 'editor.tooltip.alignCenter' },
  { dir: 'right', icon: AlignRight, label: 'editor.tooltip.alignRight' }
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
</script>
