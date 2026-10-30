<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

import { usePreferredReducedMotion } from '@vueuse/core'
const HINT_CLASS = 'text-xs text-muted-foreground'

import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useI18n } from '@shared-ui/i18n'
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
import ReorderButtons from '@shared-ui/components/ReorderButtons.vue'
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
const STAT_MESSAGES = {
  displayed: msg({ message: 'Displayed' }),
  opened: msg({ message: 'Opened' }),
  dismissed: msg({ message: 'Dismissed' }),
  replied: msg({ message: 'Replied' })
}
const AUDIENCE_MESSAGES = {
  all: msg({ message: 'Everyone' }),
  visitors: msg({ message: 'Visitors' }),
  users: msg({ message: '{count, plural, one {Signed-in user} other {Signed-in users}}' })
}
const SELECT_FIELDS = [
  {
    key: 'audience',
    labelMessage: msg({ message: 'Audience' }),
    values: ['all', 'visitors', 'users']
  }
]

const reducedMotion = usePreferredReducedMotion()

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  inboxId: { type: Number, default: 0 },
  brandName: { type: String, default: '' },
  cooldown: { type: String, default: '24h' },
  showErrors: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue', 'update:cooldown', 'update:preview'])
const i18n = useI18n()
const devices = computed(() => ({
  desktop: i18n._('Desktop'),
  mobile: i18n._('Mobile')
}))
const urlFields = computed(() => ({
  include_urls: i18n._('Include pages'),
  exclude_urls: i18n._('Exclude pages')
}))
const businessHoursOptions = computed(() => ({
  any: i18n._('Any time'),
  inside: i18n._('During business hours'),
  outside: i18n._('Outside business hours')
}))
const repeatOptions = computed(() => ({
  once: i18n._('Once per person'),
  session: i18n._('Once per session'),
  interval: i18n._('After an interval')
}))
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
const brandSenderName = computed(() => props.brandName.trim() || i18n._('Brand bot'))
const campaignEventExample = computed(
  () => `window.Libredesk.trackEvent(${JSON.stringify(campaign.value?.event || 'pricing_viewed')})`
)
const senderName = (id) => {
  const agent = usersStore.options.find((item) => String(item.value) === String(id))
  return agent?.label || brandSenderName.value
}

const campaignErrors = computed(() => {
  if (!props.showErrors || !campaign.value) return []
  const result = createCampaignSchema(i18n).safeParse(campaign.value)
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
  return isGoDuration(props.cooldown) ? '' : i18n._('Invalid duration format. Please use a valid format (e.g. 30m, 1h, 48h).')
})

const audienceLabel = (audience) => i18n._(AUDIENCE_MESSAGES[audience], 2)

const pagesLabel = (item) =>
  item.include_urls.filter(Boolean).length
    ? item.include_urls.filter(Boolean).join(', ')
    : i18n._('All')

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

const moveCampaign = (index, direction) => {
  const campaigns = [...props.modelValue]
  const [campaign] = campaigns.splice(index, 1)
  campaigns.splice(index + direction, 0, campaign)
  emit('update:modelValue', campaigns)
}

const add = (source) => {
  const item = source
    ? {
        ...JSON.parse(JSON.stringify(source)),
        id: newCampaignId(),
        enabled: false,
        name: `${source.name} (${i18n._('Copy')})`
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
    .map((item) => ({ item, result: createCampaignSchema(i18n).safeParse(item) }))
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

const visibleStats = computed(() =>
  stats.value.filter(
    (row) =>
      props.modelValue.some((item) => item.id === row.campaign_id) ||
      STAT_KEYS.some((key) => row[key] > 0)
  )
)

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
      <p class="ui-description">{{ i18n._('Show a message on matching pages before the chat is opened. A conversation is created only when someone replies to it.') }}</p>

      <div class="space-y-2 max-w-md">
        <Label for="campaign-cooldown">{{ i18n._('Minimum time between proactive messages') }}</Label>
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
          {{ i18n._('Duration in hours or minutes. Example: 1h, 30m, 1h30m') }}
        </p>
        <p v-if="cooldownError" role="alert" class="text-sm text-destructive">
          {{ cooldownError }}
        </p>
      </div>

      <div class="space-y-4">
        <h4 class="ui-heading">
          {{ i18n.ngettext('Proactive message', 'Proactive messages', 2) }}
        </h4>
        <p v-if="modelValue.length > 1" class="text-sm text-muted-foreground">
          {{ i18n._('Messages are checked from top to bottom. The first match is shown.') }}
        </p>

        <Draggable
          v-if="modelValue.length"
          v-model="orderedCampaigns"
          item-key="id"
          :animation="reducedMotion === 'reduce' ? 0 : 200"
          handle=".drag-handle"
          :force-fallback="true"
          fallback-on-body
          :fallback-tolerance="3"
          ghost-class="drag-ghost"
          class="space-y-2"
        >
          <template #item="{ element: item, index }">
            <div
              class="flex items-center gap-3 border rounded-md hover:bg-accent/50 transition-colors"
            >
              <div
                v-if="modelValue.length > 1"
                class="drag-handle ml-2 cursor-move text-muted-foreground"
              >
                <GripVertical class="size-4" aria-hidden="true" />
              </div>
              <ReorderButtons
                v-if="modelValue.length > 1"
                :index="index"
                :length="modelValue.length"
                @move="moveCampaign(index, $event)"
              />
              <Switch
                :class="modelValue.length > 1 ? '' : 'ml-3'"
                :checked="item.enabled"
                :aria-label="i18n._('Enabled')"
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
                      {{ item.name || i18n._('New proactive message') }}
                    </span>
                    <Badge variant="secondary" class="gap-1 shrink-0">
                      <component :is="item.enabled ? CircleCheck : CircleSlash" class="size-3" />
                      {{ item.enabled ? i18n._('Enabled') : i18n._('Paused') }}
                    </Badge>
                  </div>
                  <p class="text-xs text-muted-foreground truncate mt-0.5">
                    {{ audienceLabel(item.audience) }} · {{ pagesLabel(item) }}
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
          {{ i18n._('Add proactive message') }}
        </Button>
        <p v-if="modelValue.length >= MAX_CAMPAIGNS" :class="HINT_CLASS">
          {{ i18n._('You have reached the limit of 50 proactive messages. Delete one to add another.') }}
        </p>
      </div>

      <!-- Results -->
      <Collapsible v-if="inboxId" class="space-y-4">
        <CollapsibleTrigger
          type="button"
          class="flex items-center gap-2 text-base font-semibold text-foreground group"
        >
          <ChevronDown class="size-4 transition-transform group-data-[state=closed]:-rotate-90" />
          {{ i18n._('Proactive message results') }}
        </CollapsibleTrigger>
        <CollapsibleContent class="space-y-4">
          <div class="flex flex-wrap gap-3 items-end">
            <div class="space-y-2">
              <Label for="campaign-stats-from">{{ i18n.pgettext('Date range', 'From') }}</Label>
              <Input id="campaign-stats-from" v-model="from" type="date" />
            </div>
            <div class="space-y-2">
              <Label for="campaign-stats-to">{{ i18n.pgettext('Date range', 'To') }}</Label>
              <Input id="campaign-stats-to" v-model="to" type="date" />
            </div>
            <Button type="button" variant="outline" @click="refreshStats">
              {{ i18n._('Refresh') }}
            </Button>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{{ i18n.ngettext('Name', 'Names', 1) }}</TableHead>
                <TableHead v-for="key in STAT_KEYS" :key="key" class="text-right">
                  {{ i18n._(STAT_MESSAGES[key]) }}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="row in visibleStats" :key="row.campaign_id">
                <TableCell>
                  {{
                    modelValue.find((item) => item.id === row.campaign_id)?.name ||
                    i18n._('Deleted message')
                  }}
                </TableCell>
                <TableCell v-for="key in STAT_KEYS" :key="key" class="text-right tabular-nums">
                  {{ row[key] }}
                </TableCell>
              </TableRow>
              <TableRow v-if="!visibleStats.length">
                <TableCell :colspan="STAT_KEYS.length + 1" class="text-muted-foreground">
                  {{ i18n._('No results found') }}
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
          {{ i18n._('Back to proactive messages') }}
        </Button>
        <div class="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            :disabled="modelValue.length >= MAX_CAMPAIGNS"
            @click="add(campaign)"
          >
            {{ i18n._('Duplicate') }}
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            @click="pendingDelete = campaign.id"
          >
            {{ i18n._('Delete') }}
          </Button>
        </div>
      </div>

      <SwitchField
        :title="i18n._('Enabled')"
        :checked="campaign.enabled"
        @update:checked="update('enabled', $event)"
      />

      <!-- Message -->
      <div class="space-y-4">
        <h4 class="ui-heading">{{ i18n.ngettext('Message', 'Messages', 1) }}</h4>

        <div class="space-y-2">
          <Label for="campaign-name">{{ i18n.ngettext('Name', 'Names', 1) }}</Label>
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
          <Label for="campaign-message">{{ i18n.ngettext('Message', 'Messages', 1) }}</Label>
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
            <Label for="campaign-sender">{{ i18n._('Sender') }}</Label>
            <SelectAgentCombobox
              id="campaign-sender"
              :model-value="campaign.sender_id === 0 ? 'brand' : String(campaign.sender_id)"
              :prepend-items="[{ value: 'brand', label: brandSenderName }]"
              @update:model-value="update('sender_id', $event === 'brand' ? 0 : Number($event))"
            />
          </div>

          <div class="space-y-2">
            <Label for="campaign-team">{{ i18n._('Route replies to team') }}</Label>
            <SelectTeamCombobox
              id="campaign-team"
              :model-value="campaign.team_id === 0 ? 'none' : String(campaign.team_id)"
              include-none
              @update:model-value="update('team_id', $event === 'none' ? 0 : Number($event))"
            />
          </div>

          <div v-for="field in SELECT_FIELDS" :key="field.key" class="space-y-2">
            <Label :for="`campaign-${field.key}`">{{ i18n._(field.labelMessage) }}</Label>
            <Select
              :model-value="campaign[field.key]"
              @update:model-value="update(field.key, $event)"
            >
              <SelectTrigger :id="`campaign-${field.key}`"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="value in field.values" :key="value" :value="value">
                  {{ audienceLabel(value) }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <!-- Who sees it -->
      <div class="space-y-4">
        <h4 class="ui-heading">
          {{ i18n._('Who sees it') }}
        </h4>

        <div data-campaign-devices class="space-y-3">
          <p class="text-sm font-medium text-foreground">{{ i18n.ngettext('Device', 'Devices', 2) }}</p>
          <SwitchField
            v-for="(label, device) in devices"
            :key="device"
            :title="label"
            :checked="campaign[device]"
            @update:checked="update(device, $event)"
          />
          <p v-if="campaignFieldError('desktop')" role="alert" class="text-sm text-destructive">
            {{ campaignFieldError('desktop') }}
          </p>
        </div>

        <div data-campaign-conditions class="space-y-2">
          <p class="text-sm font-medium text-foreground">{{ i18n._('Visible when') }}</p>
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
        <h4 class="ui-heading">
          {{ i18n._('Where and when') }}
        </h4>

        <div v-for="(label, key) in urlFields" :key="key" class="space-y-2">
          <Label :for="`campaign-${key}`">{{ label }}</Label>
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
          <p :class="HINT_CLASS">{{ i18n._('One URL or path per line. Use * to match any characters. Empty includes match every page.') }}</p>
          <p v-if="campaignFieldError(key)" role="alert" class="text-sm text-destructive">
            {{ campaignFieldError(key) }}
          </p>
        </div>

        <div class="grid sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="campaign-delay_seconds">{{ i18n._('Active time on page (seconds)') }}</Label>
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
            <Label for="campaign-event">{{ i18n._('Optional event name') }}</Label>
            <Input
              id="campaign-event"
              :model-value="campaign.event"
              placeholder="pricing_viewed"
              maxlength="128"
              aria-describedby="campaign-event-hint campaign-event-example"
              @update:model-value="update('event', $event)"
            />
            <p id="campaign-event-hint" :class="HINT_CLASS">
              {{ i18n._('Show this message only after your website tracks this event.') }}
            </p>
            <code
              id="campaign-event-example"
              class="block overflow-x-auto rounded-md border bg-muted px-3 py-2 text-xs"
              >{{ campaignEventExample }}</code
            >
          </div>

          <div class="space-y-2">
            <Label for="campaign-business_hours">{{ i18n._('Business hours rule') }}</Label>
            <Select
              :model-value="campaign.business_hours"
              @update:model-value="update('business_hours', $event)"
            >
              <SelectTrigger id="campaign-business_hours"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="(label, value) in businessHoursOptions"
                  :key="value"
                  :value="value"
                >
                  {{ label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div
            v-if="campaign.business_hours !== 'any'"
            data-campaign-business-hours
            class="space-y-2"
          >
            <Label for="campaign-hours">{{ i18n.ngettext('Business hour', 'Business hours', 2) }}</Label>
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

      <div class="space-y-4">
        <div class="grid sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="campaign-repeat">{{ i18n._('Repeat') }}</Label>
            <Select :model-value="campaign.repeat" @update:model-value="update('repeat', $event)">
              <SelectTrigger id="campaign-repeat"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="(label, value) in repeatOptions"
                  :key="value"
                  :value="value"
                >
                  {{ label }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div v-if="campaign.repeat === 'interval'" class="space-y-2">
            <Label for="campaign-repeat_hours">{{ i18n._('Repeat after (hours)') }}</Label>
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
          <AlertDialogTitle>{{ i18n._('Are you absolutely sure?') }}</AlertDialogTitle>
          <AlertDialogDescription>{{ i18n._('This proactive message will be removed when you save the inbox.') }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel type="button">{{ i18n._('Cancel') }}</AlertDialogCancel>
          <AlertDialogAction type="button" variant="destructive" @click="remove">
            {{ i18n._('Delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
