<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Label } from '@shared-ui/components/ui/label'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem
} from '@shared-ui/components/ui/select'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import ReorderButtons from '@shared-ui/components/ReorderButtons.vue'
import { GripVertical, X, FileText } from 'lucide-vue-next'
import Draggable from 'vuedraggable'

const MAX_FEATURED = 10
const AUDIENCES = ['visitors', 'users']

const props = defineProps({
  modelValue: { type: Object, required: true },
  centers: { type: Array, default: () => [] },
  articles: { type: Array, default: () => [] },
  failed: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()

const update = (key, value) => emit('update:modelValue', { ...props.modelValue, [key]: value })
const updateAudience = (audience, key, value) =>
  update(audience, { ...props.modelValue[audience], [key]: value })

const selectHelpCenter = (value) =>
  emit('update:modelValue', {
    ...props.modelValue,
    help_center_id: Number(value),
    featured_ids: []
  })

const featuredIds = computed(() => props.modelValue.featured_ids || [])

const titleOf = (id) => props.articles.find((article) => article.id === id)?.title

const featured = computed({
  get: () => featuredIds.value.map((id) => ({ id, title: titleOf(id) })),
  set: (items) =>
    update(
      'featured_ids',
      items.map((item) => item.id)
    )
})

const moveFeatured = (index, direction) => {
  const ids = [...featuredIds.value]
  const [id] = ids.splice(index, 1)
  ids.splice(index + direction, 0, id)
  update('featured_ids', ids)
}

const unusedArticles = computed(() =>
  props.articles.filter((article) => !featuredIds.value.includes(article.id))
)
</script>

<template>
  <div class="space-y-8">
    <div class="space-y-2">
      <Label for="widget-help-center">{{ t('globals.terms.helpCenter') }}</Label>
      <Select
        :model-value="String(modelValue.help_center_id || 0)"
        @update:model-value="selectHelpCenter"
      >
        <SelectTrigger id="widget-help-center" class="max-w-md"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
          <SelectItem
            v-for="center in centers"
            :key="center.id"
            :value="String(center.id)"
            :disabled="!center.is_active"
          >
            {{ center.name }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <p v-if="failed" role="alert" class="text-sm text-destructive">
      {{ t('globals.messages.helpLoadError') }}
    </p>

    <template v-if="modelValue.help_center_id">

      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">{{ t('widget.helpPlacement') }}</h4>
        <div class="grid sm:grid-cols-2 gap-4">
          <div
            v-for="audience in AUDIENCES"
            :key="audience"
            :data-help-audience="audience"
            class="space-y-4 border rounded-md p-4"
          >
            <p class="text-sm font-medium text-foreground">
              {{ t(audience === 'visitors' ? 'globals.terms.visitor' : 'globals.terms.users', 2) }}
            </p>
            <SwitchField
              :title="t('widget.showHelpTab')"
              :checked="modelValue[audience]?.tab"
              @update:checked="updateAudience(audience, 'tab', $event)"
            />
          </div>
        </div>

        <p class="text-sm text-muted-foreground">{{ t('widget.helpOnHomeHint') }}</p>
      </div>

      <div data-featured-articles class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">
          {{ t('globals.terms.featuredArticle', 2) }}
        </h4>
        <p class="text-sm text-muted-foreground">{{ t('widget.featuredArticlesHint') }}</p>

        <Draggable
          v-model="featured"
          item-key="id"
          :animation="200"
          handle=".drag-handle"
          :force-fallback="true"
          fallback-on-body
          :fallback-tolerance="3"
          ghost-class="drag-ghost"
          class="space-y-2"
        >
          <template #item="{ element: item, index }">
            <div class="flex items-center gap-2 p-2 border rounded-md">
              <div class="drag-handle cursor-move text-muted-foreground">
                <GripVertical class="size-4" aria-hidden="true" />
              </div>
              <ReorderButtons
                :index="index"
                :length="featured.length"
                @move="moveFeatured(index, $event)"
              />
              <FileText class="size-4 shrink-0 text-muted-foreground" />
              <span
                class="flex-1 min-w-0 text-sm break-words"
                :class="{ 'text-muted-foreground': !item.title }"
              >
                {{ item.title || t('globals.messages.articleUnavailable') }}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                :aria-label="t('globals.terms.remove')"
                @click="
                  update(
                    'featured_ids',
                    featuredIds.filter((_, i) => i !== index)
                  )
                "
              >
                <X class="size-4" aria-hidden="true" />
              </Button>
            </div>
          </template>
        </Draggable>

        <Select
          :disabled="featuredIds.length >= MAX_FEATURED || !unusedArticles.length"
          model-value=""
          @update:model-value="update('featured_ids', [...featuredIds, Number($event)])"
        >
          <SelectTrigger class="max-w-md">
            <SelectValue :placeholder="t('placeholders.selectArticle')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="article in unusedArticles"
              :key="article.id"
              :value="String(article.id)"
            >
              {{ article.title }} ({{ article.locale }})
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </template>
  </div>
</template>
