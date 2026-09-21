<script setup>
const HINT_CLASS = 'text-xs text-muted-foreground'

import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { useUsersStore } from '@/stores/users'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Label } from '@shared-ui/components/ui/label'
import { Switch } from '@shared-ui/components/ui/switch'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem
} from '@shared-ui/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import Draggable from 'vuedraggable'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import SelectAgentCombobox from '@/components/combobox/SelectAgentCombobox.vue'
import SelectTeamCombobox from '@/components/combobox/SelectTeamCombobox.vue'
import WidgetConditions from './WidgetConditions.vue'
import { createCampaignSchema, defaultCampaign, newCampaignId } from './livechatFormSchema.js'
import { handleHTTPError } from '@shared-ui/utils/http'
import { isGoDuration } from '@shared-ui/utils/string'
import {
  ChevronRight,
  ChevronDown,
  GripVertical,
  Plus,
  CircleCheck,
  CircleSlash
} from 'lucide-vue-next'

const MAX_CAMPAIGNS = 50
const STAT_KEYS = ['displayed', 'opened', 'dismissed', 'replied']
const AUDIENCE_TRANSLATION_KEYS = {
  all: 'widget.campaign.all',
  visitors: 'admin.inbox.livechat.userSettings.visitors',
  users: 'widget.campaign.users'
}
const SELECT_FIELDS = [
  {
    key: 'audience',
    labelKey: 'globals.terms.audience',
    values: ['all', 'visitors', 'users']
  }
]
const DEVICES = ['desktop', 'mobile']

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  inboxId: { type: Number, default: 0 },
  brandName: { type: String, default: '' },
  cooldown: { type: String, default: '24h' },
  showErrors: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue', 'update:cooldown', 'update:preview'])
const { t } = useI18n()
const usersStore = useUsersStore()

const root = ref(null)
const selected = ref('')
const campaign = computed(() => props.modelValue.find((item) => item.id === selected.value))
const businessHours = ref([])
const error = ref('')
const pendingDelete = ref(null)
const stats = ref([])
const from = ref(new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10))
const to = ref(new Date().toISOString().slice(0, 10))
const brandSenderName = computed(() => props.brandName.trim() || t('widget.brandBot'))
const campaignEventExample = computed(
  () => `window.Libredesk.trackEvent(${JSON.stringify(campaign.value?.event || 'pricing_viewed')})`
)
const senderName = (id) => {
  const agent = usersStore.options.find((item) => String(item.value) === String(id))
  return agent?.label || brandSenderName.value
}

const campaignErrors = computed(() => {
  if (!props.showErrors || !campaign.value) return []
  const result = createCampaignSchema(t).safeParse(campaign.value)
  if (result.success) return []
  return result.error.issues.map((issue) => ({
    field: issue.path.join('.'),
    message: issue.message
  }))
})
const campaignFieldError = (...fields) =>
  campaignErrors.value.find((error) => fields.some((field) => error.field === field))?.message || ''
const campaignSectionError = (field) =>
  campaignErrors.value.find((error) => error.field === field || error.field.startsWith(`${field}.`))
    ?.message || ''
const cooldownError = computed(() => {
  if (!props.showErrors) return ''
  return isGoDuration(props.cooldown) ? '' : t('validation.invalidDuration')
})

const audienceLabel = (item) => t(AUDIENCE_TRANSLATION_KEYS[item.audience])

const pagesLabel = (item) =>
  item.include_urls.filter(Boolean).length
    ? item.include_urls.filter(Boolean).join(', ')
    : t('globals.messages.all')

// The rail preview mirrors what widget.js renders above the launcher for this campaign.
watch(
  [campaign, () => usersStore.options],
  () => {
    if (!campaign.value) {
      emit('update:preview', null)
      return
    }
    const agent = usersStore.options.find(
      (item) => String(item.value) === String(campaign.value.sender_id)
    )
    emit('update:preview', {
      id: campaign.value.id,
      sender: senderName(campaign.value.sender_id),
      avatar: agent?.avatar_url || '',
      message: campaign.value.message
    })
  },
  { deep: true, immediate: true }
)

const update = (key, value) => {
  emit(
    'update:modelValue',
    props.modelValue.map((item) => {
      if (item.id !== selected.value) return item
      const next = { ...item, [key]: value }
      if (
        key === 'repeat' &&
        value !== 'interval' &&
        !(
          Number.isInteger(next.repeat_hours) &&
          next.repeat_hours >= 1 &&
          next.repeat_hours <= 8760
        )
      ) {
        next.repeat_hours = defaultCampaign().repeat_hours
      }
      return next
    })
  )
}

const toggleEnabled = (id, enabled) =>
  emit(
    'update:modelValue',
    props.modelValue.map((item) => (item.id === id ? { ...item, enabled } : item))
  )

const orderedCampaigns = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const add = (source) => {
  const item = source
    ? {
        ...JSON.parse(JSON.stringify(source)),
        id: newCampaignId(),
        enabled: false,
        name: `${source.name} (${t('globals.terms.copy')})`
      }
    : defaultCampaign()
  emit('update:modelValue', [...props.modelValue, item])
  selected.value = item.id
}

// Radix clears pendingDelete as the dialog closes, so the id comes from the open editor.
const remove = () => {
  const id = selected.value
  emit(
    'update:modelValue',
    props.modelValue.filter((item) => item.id !== id)
  )
  selected.value = ''
  pendingDelete.value = null
}

const showInvalidField = async (field) => {
  if (field === 'config.campaign_cooldown') {
    root.value?.querySelector('#campaign-cooldown')?.focus()
    return
  }
  const firstInvalid = props.modelValue
    .map((item) => ({ item, result: createCampaignSchema(t).safeParse(item) }))
    .find(({ result }) => !result.success)
  if (!firstInvalid) return
  selected.value = firstInvalid.item.id
  await nextTick()
  const fieldName = firstInvalid.result.error.issues[0]?.path.join('.') || field
  const targetID = [
    'name',
    'message',
    'include_urls',
    'exclude_urls',
    'delay_seconds',
    'event',
    'repeat_hours'
  ].includes(fieldName)
    ? `campaign-${fieldName}`
    : ''
  const section = fieldName.startsWith('conditions')
    ? '[data-campaign-conditions]'
    : fieldName === 'desktop'
      ? '[data-campaign-devices]'
      : fieldName === 'business_hours_id'
        ? '[data-campaign-business-hours]'
        : ''
  const target = targetID
    ? root.value?.querySelector(`#${targetID}`)
    : section
      ? root.value?.querySelector(section)
      : null
  if (targetID) target?.focus?.()
  target?.scrollIntoView({ block: 'nearest' })
}

defineExpose({ showInvalidField })

const refreshStats = async () => {
  if (!props.inboxId) return
  try {
    stats.value = (
      await api.getCampaignStats(props.inboxId, { from: from.value, to: to.value })
    ).data.data
  } catch (err) {
    error.value = handleHTTPError(err).message
  }
}

onMounted(async () => {
  try {
    const response = await api.getAllBusinessHours()
    const data = response.data.data
    businessHours.value = data.results || data
  } catch (err) {
    error.value = handleHTTPError(err).message
  }
  await refreshStats()
})
</script>

<template>
  <div ref="root" class="space-y-6">
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>

    <!-- Campaign list -->
    <template v-if="!campaign">
      <p class="text-sm text-muted-foreground">{{ t('widget.campaignsHint') }}</p>

      <div class="space-y-2 max-w-md">
        <Label for="campaign-cooldown">{{ t('widget.campaignCooldown') }}</Label>
        <Input
          id="campaign-cooldown"
          type="text"
          placeholder="1h"
          :model-value="cooldown"
          aria-describedby="campaign-cooldown-hint"
          @update:model-value="emit('update:cooldown', $event)"
          :aria-invalid="!!cooldownError"
        />
        <p id="campaign-cooldown-hint" :class="HINT_CLASS">
          {{ t('globals.messages.golangDurationHoursMinutes') }}
        </p>
        <p v-if="cooldownError" role="alert" class="text-sm text-destructive">
          {{ cooldownError }}
        </p>
      </div>

      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">{{ t('widget.proactiveMessages') }}</h4>
        <p v-if="modelValue.length > 1" class="text-sm text-muted-foreground">
          {{ t('widget.campaignPriorityHint') }}
        </p>

        <Draggable
          v-if="modelValue.length"
          v-model="orderedCampaigns"
          item-key="id"
          :animation="200"
          handle=".drag-handle"
          :force-fallback="true"
          fallback-on-body
          :fallback-tolerance="3"
          ghost-class="drag-ghost"
          class="space-y-2"
        >
          <template #item="{ element: item }">
            <div
              class="flex items-center gap-3 border rounded-md hover:bg-accent/50 transition-colors"
            >
              <div
                v-if="modelValue.length > 1"
                class="drag-handle ml-2 cursor-move text-muted-foreground"
              >
                <GripVertical class="size-4" aria-hidden="true" />
              </div>
              <Switch
                :class="modelValue.length > 1 ? '' : 'ml-3'"
                :checked="item.enabled"
                :aria-label="t('globals.terms.enabled')"
                @update:checked="toggleEnabled(item.id, $event)"
              />
              <button
                type="button"
                class="flex min-w-0 flex-1 items-center gap-3 py-3 pr-3 text-left"
                @click="selected = item.id"
              >
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium text-foreground truncate">
                      {{ item.name || t('widget.newCampaign') }}
                    </span>
                    <Badge variant="secondary" class="gap-1 shrink-0">
                      <component :is="item.enabled ? CircleCheck : CircleSlash" class="size-3" />
                      {{ item.enabled ? t('globals.terms.enabled') : t('globals.terms.paused') }}
                    </Badge>
                  </div>
                  <p class="text-xs text-muted-foreground truncate mt-0.5">
                    {{ audienceLabel(item) }} · {{ pagesLabel(item) }}
                  </p>
                </div>
                <ChevronRight class="size-4 text-muted-foreground shrink-0" />
              </button>
            </div>
          </template>
        </Draggable>

        <Button
          type="button"
          variant="outline"
          size="sm"
          :disabled="modelValue.length >= MAX_CAMPAIGNS"
          @click="add()"
        >
          <Plus class="size-4" />
          {{ t('widget.addCampaign') }}
        </Button>
        <p v-if="modelValue.length >= MAX_CAMPAIGNS" :class="HINT_CLASS">
          {{ t('widget.campaignLimitReached') }}
        </p>
      </div>

      <!-- Results -->
      <Collapsible v-if="inboxId" class="space-y-4">
        <CollapsibleTrigger
          type="button"
          class="flex items-center gap-2 text-base font-semibold text-foreground group"
        >
          <ChevronDown class="size-4 transition-transform group-data-[state=closed]:-rotate-90" />
          {{ t('widget.campaignStats') }}
        </CollapsibleTrigger>
        <CollapsibleContent class="space-y-4">
          <div class="flex flex-wrap gap-3 items-end">
            <div class="space-y-2">
              <Label for="campaign-stats-from">{{ t('globals.terms.from') }}</Label>
              <Input id="campaign-stats-from" v-model="from" type="date" />
            </div>
            <div class="space-y-2">
              <Label for="campaign-stats-to">{{ t('globals.terms.to') }}</Label>
              <Input id="campaign-stats-to" v-model="to" type="date" />
            </div>
            <Button type="button" variant="outline" @click="refreshStats">
              {{ t('globals.terms.refresh') }}
            </Button>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ t('globals.terms.name') }}</TableHead>
                <TableHead v-for="key in STAT_KEYS" :key="key" class="text-right">
                  {{ t(`widget.campaign.${key}`) }}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="row in stats" :key="row.campaign_id">
                <TableCell>
                  {{
                    modelValue.find((item) => item.id === row.campaign_id)?.name ||
                    t('widget.deletedCampaign')
                  }}
                </TableCell>
                <TableCell v-for="key in STAT_KEYS" :key="key" class="text-right tabular-nums">
                  {{ row[key] }}
                </TableCell>
              </TableRow>
              <TableRow v-if="!stats.length">
                <TableCell :colspan="STAT_KEYS.length + 1" class="text-muted-foreground">
                  {{ t('globals.messages.noResultsFound') }}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CollapsibleContent>
      </Collapsible>
    </template>

    <!-- Campaign editor -->
    <template v-else>
      <div data-campaign-editor class="flex items-center justify-between gap-2">
        <Button type="button" variant="ghost" size="sm" class="-ml-2" @click="selected = ''">
          <ChevronRight class="size-4 rotate-180" />
          {{ t('widget.backToCampaigns') }}
        </Button>
        <div class="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            :disabled="modelValue.length >= MAX_CAMPAIGNS"
            @click="add(campaign)"
          >
            {{ t('globals.terms.duplicate') }}
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            @click="pendingDelete = campaign.id"
          >
            {{ t('globals.messages.delete') }}
          </Button>
        </div>
      </div>

      <SwitchField
        :title="t('globals.terms.enabled')"
        :checked="campaign.enabled"
        @update:checked="update('enabled', $event)"
      />

      <!-- Message -->
      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">{{ t('globals.terms.message', 1) }}</h4>

        <div class="space-y-2">
          <Label for="campaign-name">{{ t('globals.terms.name') }}</Label>
          <Input
            id="campaign-name"
            :model-value="campaign.name"
            maxlength="128"
            @update:model-value="update('name', $event)"
            :aria-invalid="!!campaignFieldError('name')"
          />
          <p v-if="campaignFieldError('name')" role="alert" class="text-sm text-destructive">
            {{ campaignFieldError('name') }}
          </p>
        </div>

        <div class="space-y-2">
          <Label for="campaign-message">{{ t('globals.terms.message', 1) }}</Label>
          <Textarea
            id="campaign-message"
            :model-value="campaign.message"
            maxlength="10000"
            rows="4"
            @update:model-value="update('message', $event)"
            :aria-invalid="!!campaignFieldError('message')"
          />
          <p v-if="campaignFieldError('message')" role="alert" class="text-sm text-destructive">
            {{ campaignFieldError('message') }}
          </p>
        </div>

        <div class="grid sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="campaign-sender">{{ t('globals.terms.sender') }}</Label>
            <SelectAgentCombobox
              id="campaign-sender"
              :model-value="campaign.sender_id === 0 ? 'brand' : String(campaign.sender_id)"
              :prepend-items="[{ value: 'brand', label: brandSenderName }]"
              @update:model-value="update('sender_id', $event === 'brand' ? 0 : Number($event))"
            />
          </div>

          <div class="space-y-2">
            <Label for="campaign-team">{{ t('widget.replyTeam') }}</Label>
            <SelectTeamCombobox
              id="campaign-team"
              :model-value="campaign.team_id === 0 ? 'none' : String(campaign.team_id)"
              include-none
              @update:model-value="update('team_id', $event === 'none' ? 0 : Number($event))"
            />
          </div>

          <div v-for="field in SELECT_FIELDS" :key="field.key" class="space-y-2">
            <Label :for="`campaign-${field.key}`">{{ t(field.labelKey) }}</Label>
            <Select
              :model-value="campaign[field.key]"
              @update:model-value="update(field.key, $event)"
            >
              <SelectTrigger :id="`campaign-${field.key}`"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="value in field.values" :key="value" :value="value">
                  {{ t(AUDIENCE_TRANSLATION_KEYS[value]) }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <!-- Who sees it -->
      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">
          {{ t('widget.campaign.section.audience') }}
        </h4>

        <div data-campaign-devices class="space-y-3">
          <p class="text-sm font-medium text-foreground">{{ t('globals.terms.device', 2) }}</p>
          <SwitchField
            v-for="device in DEVICES"
            :key="device"
            :title="t(`globals.terms.${device}`)"
            :checked="campaign[device]"
            @update:checked="update(device, $event)"
          />
          <p v-if="campaignFieldError('desktop')" role="alert" class="text-sm text-destructive">
            {{ campaignFieldError('desktop') }}
          </p>
        </div>

        <div data-campaign-conditions class="space-y-2">
          <p class="text-sm font-medium text-foreground">{{ t('globals.messages.visibleWhen') }}</p>
          <WidgetConditions
            :model-value="campaign.conditions"
            @update:model-value="update('conditions', $event)"
          />
          <p
            v-if="campaignSectionError('conditions')"
            role="alert"
            class="text-sm text-destructive"
          >
            {{ campaignSectionError('conditions') }}
          </p>
        </div>
      </div>

      <!-- Where and when -->
      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">
          {{ t('widget.campaign.section.placement') }}
        </h4>

        <div v-for="key in ['include_urls', 'exclude_urls']" :key="key" class="space-y-2">
          <Label :for="`campaign-${key}`">{{ t(`widget.campaign.${key}`) }}</Label>
          <Textarea
            :id="`campaign-${key}`"
            :model-value="campaign[key].join('\n')"
            :placeholder="
              key === 'include_urls' ? '/pricing\nhttps://example.com/docs/*' : '/checkout/*'
            "
            rows="3"
            @update:model-value="update(key, $event.split('\n'))"
            :aria-invalid="!!campaignFieldError(key)"
          />
          <p :class="HINT_CLASS">{{ t('widget.urlPatternsHint') }}</p>
          <p v-if="campaignFieldError(key)" role="alert" class="text-sm text-destructive">
            {{ campaignFieldError(key) }}
          </p>
        </div>

        <div class="grid sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="campaign-delay_seconds">{{ t('widget.campaign.delay_seconds') }}</Label>
            <Input
              id="campaign-delay_seconds"
              type="number"
              min="0"
              max="86400"
              :model-value="campaign.delay_seconds"
              @update:model-value="update('delay_seconds', Number($event))"
              :aria-invalid="!!campaignFieldError('delay_seconds')"
            />
            <p
              v-if="campaignFieldError('delay_seconds')"
              role="alert"
              class="text-sm text-destructive"
            >
              {{ campaignFieldError('delay_seconds') }}
            </p>
          </div>

          <div class="space-y-2">
            <Label for="campaign-event">{{ t('widget.campaign.event') }}</Label>
            <Input
              id="campaign-event"
              :model-value="campaign.event"
              placeholder="pricing_viewed"
              maxlength="128"
              aria-describedby="campaign-event-hint campaign-event-example"
              @update:model-value="update('event', $event)"
            />
            <p id="campaign-event-hint" :class="HINT_CLASS">
              {{ t('widget.campaignEventHint') }}
            </p>
            <code
              id="campaign-event-example"
              class="block overflow-x-auto rounded-md border bg-muted px-3 py-2 text-xs"
              >{{ campaignEventExample }}</code
            >
          </div>

          <div class="space-y-2">
            <Label for="campaign-business_hours">{{ t('widget.campaign.business_hours') }}</Label>
            <Select
              :model-value="campaign.business_hours"
              @update:model-value="update('business_hours', $event)"
            >
              <SelectTrigger id="campaign-business_hours"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="value in ['any', 'inside', 'outside']"
                  :key="value"
                  :value="value"
                >
                  {{ t(`widget.campaign.${value}`) }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div
            v-if="campaign.business_hours !== 'any'"
            data-campaign-business-hours
            class="space-y-2"
          >
            <Label for="campaign-hours">{{ t('globals.terms.businessHour', 2) }}</Label>
            <Select
              :model-value="String(campaign.business_hours_id)"
              @update:model-value="update('business_hours_id', Number($event))"
            >
              <SelectTrigger id="campaign-hours"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="item in businessHours" :key="item.id" :value="String(item.id)">
                  {{ item.name }}
                </SelectItem>
              </SelectContent>
            </Select>
            <p
              v-if="campaignFieldError('business_hours_id')"
              role="alert"
              class="text-sm text-destructive"
            >
              {{ campaignFieldError('business_hours_id') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Repeat -->
      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">
          {{ t('globals.terms.repeat') }}
        </h4>

        <div class="grid sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="campaign-repeat">{{ t('widget.campaign.repeat') }}</Label>
            <Select :model-value="campaign.repeat" @update:model-value="update('repeat', $event)">
              <SelectTrigger id="campaign-repeat"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="value in ['once', 'session', 'interval']"
                  :key="value"
                  :value="value"
                >
                  {{ t(`widget.campaign.${value}`) }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div v-if="campaign.repeat === 'interval'" class="space-y-2">
            <Label for="campaign-repeat_hours">{{ t('widget.campaign.repeat_hours') }}</Label>
            <Input
              id="campaign-repeat_hours"
              type="number"
              min="1"
              max="8760"
              :model-value="campaign.repeat_hours"
              @update:model-value="update('repeat_hours', Number($event))"
              :aria-invalid="!!campaignFieldError('repeat_hours')"
            />
            <p
              v-if="campaignFieldError('repeat_hours')"
              role="alert"
              class="text-sm text-destructive"
            >
              {{ campaignFieldError('repeat_hours') }}
            </p>
          </div>
        </div>
      </div>
    </template>

    <AlertDialog
      :open="!!pendingDelete"
      @update:open="pendingDelete = $event ? pendingDelete : null"
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
          <AlertDialogDescription>{{ t('confirm.deleteCampaign') }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel type="button">{{ t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction type="button" variant="destructive" @click="remove">
            {{ t('globals.messages.delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
