<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <!-- Summary Section -->
    <div class="bg-muted/30 box py-6 px-3" v-if="!isNewForm">
      <div class="flex items-start gap-6">
        <Avatar class="w-20 h-20">
          <AvatarImage :src="props.initialValues.avatar_url || ''" :alt="Avatar" />
          <AvatarFallback>
            {{ getInitials(props.initialValues.first_name, props.initialValues.last_name) }}
          </AvatarFallback>
        </Avatar>

        <div class="space-y-4 flex-2">
          <div class="flex items-center gap-3">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-foreground">
              {{ props.initialValues.first_name }} {{ props.initialValues.last_name }}
            </h3>
            <Badge :class="['px-2 rounded-full text-xs font-medium', availabilityStatus.color]">
              {{ availabilityStatus.text }}
            </Badge>
          </div>

          <div class="flex flex-wrap items-center gap-6">
            <div class="flex items-center gap-2">
              <Clock class="w-5 h-5 text-gray-400" />
              <div>
                <p class="text-sm text-gray-500">{{ $t('globals.terms.lastActive') }}</p>
                <p class="text-sm font-medium text-gray-700 dark:text-foreground">
                  {{
                    props.initialValues.last_active_at
                      ? format(new Date(props.initialValues.last_active_at), 'PPpp')
                      : 'N/A'
                  }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <LogIn class="w-5 h-5 text-gray-400" />
              <div>
                <p class="text-sm text-gray-500">{{ $t('globals.terms.lastLogin') }}</p>
                <p class="text-sm font-medium text-gray-700 dark:text-foreground">
                  {{
                    props.initialValues.last_login_at
                      ? format(new Date(props.initialValues.last_login_at), 'PPpp')
                      : 'N/A'
                  }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- API Key Management Section -->
    <div class="bg-muted/30 box p-4 space-y-4" v-if="!isNewForm">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <p class="text-base font-semibold text-gray-900 dark:text-foreground">
            {{ $t('globals.terms.apiKey', 2) }}
          </p>
          <p class="text-sm text-gray-500">
            {{ $t('admin.agent.apiKey.description') }}
          </p>
        </div>
      </div>

      <!-- API Key Display -->
      <div v-if="apiKeyData.api_key" class="space-y-3">
        <div class="flex items-center justify-between p-3 bg-background border rounded-md">
          <div class="flex items-center gap-3">
            <Key class="w-4 h-4 text-gray-400" />
            <div>
              <p class="text-sm font-medium">{{ $t('globals.terms.apiKey') }}</p>
              <p class="text-xs text-gray-500 font-mono">{{ apiKeyData.api_key }}</p>
            </div>
          </div>
          <div class="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              @click="regenerateAPIKey"
              :disabled="isAPIKeyLoading"
            >
              <RotateCcw class="w-4 h-4 mr-1" />
              {{ $t('globals.messages.regenerate') }}
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              @click="revokeAPIKey"
              :disabled="isAPIKeyLoading"
            >
              <Trash2 class="w-4 h-4 mr-1" />
              {{ $t('globals.messages.revoke') }}
            </Button>
          </div>
        </div>

        <!-- Last Used Info -->
        <div v-if="apiKeyLastUsedAt" class="text-xs text-gray-500">
          {{ $t('globals.messages.lastUsed') }}:
          {{ format(new Date(apiKeyLastUsedAt), 'PPpp') }}
        </div>
      </div>

      <!-- No API Key State -->
      <div v-else class="text-center py-6">
        <Key class="w-8 h-8 text-gray-400 mx-auto mb-2" />
        <p class="text-sm text-gray-500 mb-3">{{ $t('admin.agent.apiKey.noKey') }}</p>
        <Button type="button" @click="generateAPIKey" :disabled="isAPIKeyLoading">
          <Plus class="w-4 h-4 mr-1" />
          {{ $t('globals.messages.generate', { name: $t('globals.terms.apiKey') }) }}
        </Button>
      </div>
    </div>

    <!-- API Key Display Dialog -->
    <Dialog v-model:open="showAPIKeyDialog">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {{ $t('globals.messages.generated', { name: $t('globals.terms.apiKey') }) }}
          </DialogTitle>
          <DialogDescription> </DialogDescription>
        </DialogHeader>
        <div class="space-y-4">
          <div>
            <Label class="text-sm font-medium">{{ $t('globals.terms.apiKey') }}</Label>
            <div class="flex items-center gap-2 mt-1">
              <Input v-model="newAPIKeyData.api_key" readonly class="font-mono text-sm" />
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="copyToClipboard(newAPIKeyData.api_key)"
              >
                <Copy class="w-4 h-4" />
              </Button>
            </div>
          </div>
          <div>
            <Label class="text-sm font-medium">{{ $t('globals.terms.secret') }}</Label>
            <div class="flex items-center gap-2 mt-1">
              <Input v-model="newAPIKeyData.api_secret" readonly class="font-mono text-sm" />
              <Button
                type="button"
                variant="outline"
                size="sm"
                @click="copyToClipboard(newAPIKeyData.api_secret)"
              >
                <Copy class="w-4 h-4" />
              </Button>
            </div>
          </div>
          <Alert>
            <AlertTriangle class="h-4 w-4" />
            <AlertTitle>{{ $t('globals.terms.warning') }}</AlertTitle>
            <AlertDescription>
              {{ $t('admin.agent.apiKey.warningMessage') }}
            </AlertDescription>
          </Alert>
        </div>
        <DialogFooter>
          <Button @click="closeAPIKeyModal">{{ $t('globals.messages.close') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Form Fields -->
    <FormField v-slot="{ field }" name="first_name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.firstName') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="last_name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.lastName') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="email">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.email') }}</FormLabel>
        <FormControl>
          <Input type="email" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="teams">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.team', 2) }}</FormLabel>
        <FormControl>
          <SelectTag
            :items="teamOptions"
            :placeholder="t('globals.messages.select', { name: t('globals.terms.team', 2) })"
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="roles">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('globals.terms.role', 2) }}</FormLabel>
        <FormControl>
          <SelectTag
            :items="roleOptions"
            :placeholder="
              t('globals.messages.select', {
                name: $t('globals.terms.role', 2)
              })
            "
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="availability_status" v-if="!isNewForm">
      <FormItem>
        <FormLabel>{{ t('globals.terms.availabilityStatus') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField" v-model="componentField.modelValue">
            <SelectTrigger>
              <SelectValue
                :placeholder="
                  t('globals.messages.select', {
                    name: t('globals.terms.availabilityStatus')
                  })
                "
              />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="active_group">{{ t('globals.terms.active') }}</SelectItem>
                <SelectItem value="away_manual">{{ t('globals.terms.away') }}</SelectItem>
                <SelectItem value="away_and_reassigning">
                  {{ t('globals.terms.awayReassigning') }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="new_password" v-if="!isNewForm">
      <FormItem v-auto-animate>
        <FormLabel>{{ t('globals.terms.setPassword') }}</FormLabel>
        <FormControl>
          <Input type="password" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="send_welcome_email" v-slot="{ value, handleChange }" v-if="isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>{{ $t('globals.terms.sendWelcomeEmail') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ value, handleChange }" type="checkbox" name="enabled" v-if="!isNewForm">
      <FormItem class="flex flex-row items-start gap-x-3 space-y-0">
        <FormControl>
          <Checkbox :checked="value" @update:checked="handleChange" />
        </FormControl>
        <div class="space-y-1 leading-none">
          <FormLabel> {{ $t('globals.terms.enabled') }} </FormLabel>
          <FormMessage />
        </div>
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, onMounted, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { Badge } from '@/components/ui/badge'
import { Clock, LogIn, Key, RotateCcw, Trash2, Plus, Copy, AlertTriangle } from 'lucide-vue-next'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { SelectTag } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { format } from 'date-fns'
import api from '@/api'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: 'Submit'
  },
  isNewForm: {
    type: Boolean,
    required: false,
    default: false
  },
  isLoading: {
    Type: Boolean,
    required: false
  }
})
const { t } = useI18n()
const teams = ref([])
const roles = ref([])
const emitter = useEmitter()

const apiKeyData = ref({
  api_key: props.initialValues?.api_key || '',
  api_secret: ''
})
const apiKeyLastUsedAt = ref(props.initialValues?.api_key_last_used_at || null)
const newAPIKeyData = ref({
  api_key: '',
  api_secret: ''
})
const showAPIKeyDialog = ref(false)
const isAPIKeyLoading = ref(false)

onMounted(async () => {
  try {
    const [teamsResp, rolesResp] = await Promise.allSettled([api.getTeams(), api.getRoles()])
    teams.value = teamsResp.value.data.data
    roles.value = rolesResp.value.data.data
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.errorFetching')
    })
  }
})

const availabilityStatus = computed(() => {
  const status = form.values.availability_status
  if (status === 'active_group') return { text: t('globals.terms.active'), color: 'bg-green-500' }
  if (status === 'away_manual') return { text: t('globals.terms.away'), color: 'bg-yellow-500' }
  if (status === 'away_and_reassigning')
    return { text: t('globals.terms.awayReassigning'), color: 'bg-orange-500' }
  return { text: t('globals.terms.offline'), color: 'bg-gray-400' }
})

const teamOptions = computed(() =>
  teams.value.map((team) => ({ label: team.name, value: team.name }))
)
const roleOptions = computed(() =>
  roles.value.map((role) => ({ label: role.name, value: role.name }))
)

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSubmit = form.handleSubmit((values) => {
  if (values.availability_status === 'active_group') {
    values.availability_status = 'online'
  }
  props.submitForm(values)
})

const getInitials = (firstName, lastName) => {
  if (!firstName && !lastName) return ''
  if (!firstName) return lastName.charAt(0).toUpperCase()
  if (!lastName) return firstName.charAt(0).toUpperCase()
  return `${firstName.charAt(0).toUpperCase()}${lastName.charAt(0).toUpperCase()}`
}

const generateAPIKey = async () => {
  if (!props.initialValues?.id) return

  try {
    isAPIKeyLoading.value = true
    const response = await api.generateAPIKey(props.initialValues.id)
    if (response.data) {
      const responseData = response.data.data
      newAPIKeyData.value = {
        api_key: responseData.api_key,
        api_secret: responseData.api_secret
      }
      apiKeyData.value.api_key = responseData.api_key

      // Clear the last used timestamp since this is a new API key
      apiKeyLastUsedAt.value = null

      showAPIKeyDialog.value = true
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('globals.messages.generatedSuccessfully', {
          name: t('globals.terms.apiKey')
        })
      })
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.errorGenerating', {
        name: t('globals.terms.apiKey')
      })
    })
  } finally {
    isAPIKeyLoading.value = false
  }
}

const regenerateAPIKey = async () => {
  await generateAPIKey()
}

const revokeAPIKey = async () => {
  if (!props.initialValues?.id) return
  try {
    isAPIKeyLoading.value = true
    await api.revokeAPIKey(props.initialValues.id)
    apiKeyData.value.api_key = ''
    apiKeyData.value.api_secret = ''
    apiKeyLastUsedAt.value = null
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.revokedSuccessfully', {
        name: t('globals.terms.apiKey')
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.errorRevoking', {
        name: t('globals.terms.apiKey')
      })
    })
  } finally {
    isAPIKeyLoading.value = false
  }
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.copied')
    })
  } catch (error) {
    console.error('Error copying to clipboard:', error)
  }
}

const closeAPIKeyModal = () => {
  showAPIKeyDialog.value = false
  newAPIKeyData.value = { api_key: '', api_secret: '' }
}

watch(
  () => props.initialValues,
  (newValues) => {
    // Hack.
    if (Object.keys(newValues).length > 0) {
      setTimeout(() => {
        if (
          newValues.availability_status === 'away' ||
          newValues.availability_status === 'offline' ||
          newValues.availability_status === 'online'
        ) {
          newValues.availability_status = 'active_group'
        }
        form.setValues(newValues)
        form.setFieldValue(
          'teams',
          newValues.teams.map((team) => team.name)
        )

        // Update API key data
        apiKeyData.value.api_key = newValues.api_key || ''
        apiKeyLastUsedAt.value = newValues.api_key_last_used_at || null
      }, 0)
    }
  },
  { deep: true, immediate: true }
)
</script>
