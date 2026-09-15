<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@/api'
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
import SwitchField from '@shared-ui/components/SwitchField.vue'
import WidgetConditions from './WidgetConditions.vue'
import { defaultCampaign } from './livechatFormSchema.js'
import { handleHTTPError } from '@shared-ui/utils/http'
import { ChevronRight, ChevronDown, Plus, CircleCheck, CircleSlash } from 'lucide-vue-next'

const MAX_CAMPAIGNS = 50
const STAT_KEYS = ['displayed', 'opened', 'dismissed', 'replied']
const SELECT_FIELDS = [{ key: 'audience', values: ['all', 'visitors', 'users'] }]
const DEVICES = ['desktop', 'mobile']

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  inboxId: { type: Number, default: 0 },
  cooldown: { type: Number, default: 24 }
})
const emit = defineEmits(['update:modelValue', 'update:cooldown', 'update:preview'])
const { t } = useI18n()

const selected = ref('')
const campaign = computed(() => props.modelValue.find((item) => item.id === selected.value))
const teams = ref([])
const agents = ref([])
const businessHours = ref([])
const error = ref('')
const pendingDelete = ref(null)
const stats = ref([])
const from = ref(new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10))
const to = ref(new Date().toISOString().slice(0, 10))
const senderName = (id) => {
  const agent = agents.value.find((item) => item.id === id)
  return agent ? `${agent.first_name} ${agent.last_name || ''}`.trim() : t('widget.brandBot')
}

const audienceLabel = (item) => t(`widget.campaign.${item.audience}`)

const pagesLabel = (item) =>
  item.include_urls.filter(Boolean).length
    ? item.include_urls.filter(Boolean).join(', ')
    : t('globals.messages.all')

// The rail preview mirrors what widget.js renders above the launcher for this campaign.
watch(
  [campaign, agents],
  () => {
    if (!campaign.value) {
      emit('update:preview', null)
      return
    }
    const agent = agents.value.find((item) => item.id === campaign.value.sender_id)
    emit('update:preview', {
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
    props.modelValue.map((item) => (item.id === selected.value ? { ...item, [key]: value } : item))
  )
}

const toggleEnabled = (id, enabled) =>
  emit(
    'update:modelValue',
    props.modelValue.map((item) => (item.id === id ? { ...item, enabled } : item))
  )

const add = (source) => {
  const item = source
    ? {
        ...JSON.parse(JSON.stringify(source)),
        id: crypto.randomUUID(),
        enabled: false,
        name: `${source.name} (${t('globals.terms.copy')})`
      }
    : defaultCampaign()
  emit('update:modelValue', [...props.modelValue, item])
  selected.value = item.id
}

const remove = () => {
  emit(
    'update:modelValue',
    props.modelValue.filter((item) => item.id !== pendingDelete.value)
  )
  if (selected.value === pendingDelete.value) selected.value = ''
  pendingDelete.value = null
}

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
  const responses = await Promise.allSettled([
    api.getTeams(),
    api.getUsersCompact({ enabled: true, per_page: 100 }),
    api.getAllBusinessHours()
  ])
  for (const [index, response] of responses.entries()) {
    if (response.status === 'rejected') {
      error.value = handleHTTPError(response.reason).message
      continue
    }
    const data = response.value.data.data
    ;[teams, agents, businessHours][index].value = data.results || data
  }
  await refreshStats()
})
</script>

<template>
  <div class="space-y-6">
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>

    <!-- Campaign list -->
    <template v-if="!campaign">
      <p class="text-sm text-muted-foreground">{{ t('widget.campaignsHint') }}</p>

      <div class="space-y-2 max-w-md">
        <Label for="campaign-cooldown">{{ t('widget.campaignCooldown') }}</Label>
        <Input
          id="campaign-cooldown"
          type="number"
          min="1"
          max="8760"
          :model-value="cooldown"
          @update:model-value="emit('update:cooldown', Number($event))"
        />
      </div>

      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">{{ t('widget.proactiveMessages') }}</h4>

        <div v-if="modelValue.length" class="space-y-2">
          <div
            v-for="item in modelValue"
            :key="item.id"
            class="flex items-center gap-3 p-3 border rounded-md hover:bg-accent/50 transition-colors cursor-pointer"
            role="button"
            tabindex="0"
            @click="selected = item.id"
            @keydown.enter.prevent="selected = item.id"
            @keydown.space.prevent="selected = item.id"
          >
            <Switch
              :checked="item.enabled"
              :aria-label="t('globals.terms.enabled')"
              @click.stop
              @update:checked="toggleEnabled(item.id, $event)"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-foreground truncate">
                  {{ item.name || t('widget.newCampaign') }}
                </span>
                <Badge :variant="item.enabled ? 'success' : 'secondary'" class="gap-1 shrink-0">
                  <component :is="item.enabled ? CircleCheck : CircleSlash" class="size-3" />
                  {{ item.enabled ? t('globals.terms.enabled') : t('globals.terms.paused') }}
                </Badge>
              </div>
              <p class="text-xs text-muted-foreground truncate mt-0.5">
                {{ audienceLabel(item) }} · {{ pagesLabel(item) }}
              </p>
            </div>
            <ChevronRight class="size-4 text-muted-foreground shrink-0" />
          </div>
        </div>

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
        <p v-if="modelValue.length >= MAX_CAMPAIGNS" class="text-xs text-muted-foreground">
          {{ t('widget.campaignLimitReached') }}
        </p>
      </div>

      <!-- Results -->
      <Collapsible v-if="inboxId" class="space-y-4">
        <CollapsibleTrigger
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
      <div class="flex items-center justify-between gap-2">
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
          />
        </div>

        <div class="space-y-2">
          <Label for="campaign-message">{{ t('globals.terms.message', 1) }}</Label>
          <Textarea
            id="campaign-message"
            :model-value="campaign.message"
            maxlength="10000"
            rows="4"
            @update:model-value="update('message', $event)"
          />
        </div>

        <div class="grid sm:grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="campaign-sender">{{ t('widget.campaignSender') }}</Label>
            <Select
              :model-value="String(campaign.sender_id)"
              @update:model-value="update('sender_id', Number($event))"
            >
              <SelectTrigger id="campaign-sender"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="0">{{ t('widget.brandBot') }}</SelectItem>
                <SelectItem v-for="agent in agents" :key="agent.id" :value="String(agent.id)">
                  {{ agent.first_name }} {{ agent.last_name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="space-y-2">
            <Label for="campaign-team">{{ t('widget.replyTeam') }}</Label>
            <Select
              :model-value="String(campaign.team_id)"
              @update:model-value="update('team_id', Number($event))"
            >
              <SelectTrigger id="campaign-team"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="0">{{ t('globals.terms.none') }}</SelectItem>
                <SelectItem v-for="team in teams" :key="team.id" :value="String(team.id)">
                  {{ team.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div v-for="field in SELECT_FIELDS" :key="field.key" class="space-y-2">
            <Label :for="`campaign-${field.key}`">{{ t(`widget.campaign.${field.key}`) }}</Label>
            <Select
              :model-value="campaign[field.key]"
              @update:model-value="update(field.key, $event)"
            >
              <SelectTrigger :id="`campaign-${field.key}`"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="value in field.values" :key="value" :value="value">
                  {{ t(`widget.campaign.${value}`) }}
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

        <div class="space-y-3">
          <p class="text-sm font-medium text-foreground">{{ t('widget.campaign.devices') }}</p>
          <SwitchField
            v-for="device in DEVICES"
            :key="device"
            :title="t(`globals.terms.${device}`)"
            :checked="campaign[device]"
            @update:checked="update(device, $event)"
          />
        </div>

        <div class="space-y-2">
          <p class="text-sm font-medium text-foreground">{{ t('globals.messages.visibleWhen') }}</p>
          <WidgetConditions
            :model-value="campaign.conditions"
            @update:model-value="update('conditions', $event)"
          />
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
            rows="3"
            @update:model-value="update(key, $event.split('\n'))"
          />
          <p class="text-xs text-muted-foreground">{{ t('widget.urlPatternsHint') }}</p>
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
            />
          </div>

          <div class="space-y-2">
            <Label for="campaign-event">{{ t('widget.campaign.event') }}</Label>
            <Input
              id="campaign-event"
              :model-value="campaign.event"
              maxlength="128"
              @update:model-value="update('event', $event)"
            />
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

          <div v-if="campaign.business_hours !== 'any'" class="space-y-2">
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
          </div>
        </div>
      </div>

      <!-- Repeat -->
      <div class="space-y-4">
        <h4 class="text-base font-semibold text-foreground">
          {{ t('widget.campaign.section.repeat') }}
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
            />
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
          <AlertDialogDescription>{{
            t('confirm.deleteCampaign')
          }}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" @click="remove">
            {{ t('globals.messages.delete') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
