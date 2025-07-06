<template>
  <form @submit="onSubmit" class="space-y-6 w-full">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ $t('admin.inbox.name.description') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('globals.terms.enabled') }}</FormLabel>
          <FormDescription>{{ $t('admin.inbox.enabled.description') }}</FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="csat_enabled">
      <FormItem class="flex flex-row items-center justify-between box p-4">
        <div class="space-y-0.5">
          <FormLabel class="text-base">{{ $t('admin.inbox.csatSurveys') }}</FormLabel>
          <FormDescription>
            {{ $t('admin.inbox.csatSurveys.description_1') }}<br />
            {{ $t('admin.inbox.csatSurveys.description_2') }}
          </FormDescription>
        </div>
        <FormControl>
          <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
        </FormControl>
      </FormItem>
    </FormField>

    <!-- Livechat Configuration -->
    <div class="box p-4 space-y-6">
      <h3 class="font-semibold">{{ $t('admin.inbox.livechatConfig') }}</h3>

      <FormField v-slot="{ componentField }" name="config.brand_name">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.brandName') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.livechat.brandName.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <!-- Logo URL -->
      <FormField v-slot="{ componentField }" name="config.logo_url">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.livechat.logoUrl') }}</FormLabel>
          <FormControl>
            <Input type="url" placeholder="https://example.com/logo.png" v-bind="componentField" />
          </FormControl>
          <FormDescription>{{ $t('admin.inbox.livechat.logoUrl.description') }}</FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>

      <!-- Secret Key (readonly) -->
      <div v-if="hasSecretKey">
        <FormField v-slot="{ componentField }" name="config.secret_key">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.livechat.secretKey') }}</FormLabel>
            <FormControl>
              <div class="flex items-center gap-2">
                <Input
                  type="text"
                  v-bind="componentField"
                  readonly
                  class="font-mono text-sm bg-muted"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  @click="copyToClipboard(componentField.modelValue)"
                >
                  <Copy class="w-4 h-4" />
                </Button>
              </div>
            </FormControl>
            <FormDescription>{{
              $t('admin.inbox.livechat.secretKey.description')
            }}</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <!-- Launcher Configuration -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.launcher') }}</h4>

        <div class="grid grid-cols-2 gap-4">
          <!-- Launcher Position -->
          <FormField v-slot="{ componentField }" name="config.launcher.position">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.launcher.position') }}</FormLabel>
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue placeholder="Select position" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="left">{{
                      $t('admin.inbox.livechat.launcher.position.left')
                    }}</SelectItem>
                    <SelectItem value="right">{{
                      $t('admin.inbox.livechat.launcher.position.right')
                    }}</SelectItem>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <!-- Launcher Logo -->
          <FormField v-slot="{ componentField }" name="config.launcher.logo_url">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.launcher.logo') }}</FormLabel>
              <FormControl>
                <Input
                  type="url"
                  placeholder="https://example.com/launcher-logo.png"
                  v-bind="componentField"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <!-- Launcher Spacing Side -->
          <FormField v-slot="{ componentField }" name="config.launcher.spacing.side">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.launcher.spacing.side') }}</FormLabel>
              <FormControl>
                <Input type="number" placeholder="20" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{
                $t('admin.inbox.livechat.launcher.spacing.side.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>

          <!-- Launcher Spacing Bottom -->
          <FormField v-slot="{ componentField }" name="config.launcher.spacing.bottom">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.launcher.spacing.bottom') }}</FormLabel>
              <FormControl>
                <Input type="number" placeholder="20" v-bind="componentField" />
              </FormControl>
              <FormDescription>{{
                $t('admin.inbox.livechat.launcher.spacing.bottom.description')
              }}</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
      </div>

      <!-- Messages -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.messages') }}</h4>

        <FormField v-slot="{ componentField }" name="config.greeting_message">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.livechat.greetingMessage') }}</FormLabel>
            <FormControl>
              <Textarea
                v-bind="componentField"
                placeholder="Welcome! How can we help you today?"
                rows="2"
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.introduction_message">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.livechat.introductionMessage') }}</FormLabel>
            <FormControl>
              <Textarea v-bind="componentField" placeholder="We're here to help!" rows="2" />
            </FormControl>
            <FormMessage />
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="config.chat_introduction">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.livechat.chatIntroduction') }}</FormLabel>
            <FormControl>
              <Textarea
                v-bind="componentField"
                placeholder="Ask us anything, or share your feedback."
                rows="2"
              />
            </FormControl>
            <FormDescription>{{
              $t('admin.inbox.livechat.chatIntroduction.description')
            }}</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <!-- Office Hours -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.officeHours') }}</h4>

        <FormField
          v-slot="{ componentField, handleChange }"
          name="config.show_office_hours_in_chat"
        >
          <FormItem class="flex flex-row items-center justify-between box p-4">
            <div class="space-y-0.5">
              <FormLabel class="text-base">{{
                $t('admin.inbox.livechat.showOfficeHoursInChat')
              }}</FormLabel>
              <FormDescription>{{
                $t('admin.inbox.livechat.showOfficeHoursInChat.description')
              }}</FormDescription>
            </div>
            <FormControl>
              <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
            </FormControl>
          </FormItem>
        </FormField>

        <FormField
          v-slot="{ componentField, handleChange }"
          name="config.show_office_hours_after_assignment"
        >
          <FormItem class="flex flex-row items-center justify-between box p-4">
            <div class="space-y-0.5">
              <FormLabel class="text-base">{{
                $t('admin.inbox.livechat.showOfficeHoursAfterAssignment')
              }}</FormLabel>
              <FormDescription>{{
                $t('admin.inbox.livechat.showOfficeHoursAfterAssignment.description')
              }}</FormDescription>
            </div>
            <FormControl>
              <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
            </FormControl>
          </FormItem>
        </FormField>
      </div>

      <!-- Notice Banner -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.noticeBanner') }}</h4>

        <FormField v-slot="{ componentField, handleChange }" name="config.notice_banner.enabled">
          <FormItem class="flex flex-row items-center justify-between box p-4">
            <div class="space-y-0.5">
              <FormLabel class="text-base">{{
                $t('admin.inbox.livechat.noticeBanner.enabled')
              }}</FormLabel>
              <FormDescription>{{
                $t('admin.inbox.livechat.noticeBanner.enabled.description')
              }}</FormDescription>
            </div>
            <FormControl>
              <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
            </FormControl>
          </FormItem>
        </FormField>

        <FormField
          v-slot="{ componentField }"
          name="config.notice_banner.text"
          v-if="form.values.config?.notice_banner?.enabled"
        >
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.livechat.noticeBanner.text') }}</FormLabel>
            <FormControl>
              <Textarea
                v-bind="componentField"
                placeholder="Our response times are slower than usual. We're working hard to get to your message."
                rows="2"
              />
            </FormControl>
            <FormDescription>{{
              $t('admin.inbox.livechat.noticeBanner.text.description')
            }}</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <!-- Colors -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.colors') }}</h4>

        <div class="grid grid-cols-2 gap-4">
          <FormField v-slot="{ componentField }" name="config.colors.primary">
            <FormItem>
              <FormLabel>{{ $t('admin.inbox.livechat.colors.primary') }}</FormLabel>
              <FormControl>
                <Input type="color" v-bind="componentField" />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
      </div>

      <!-- Features -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.features') }}</h4>

        <div class="space-y-3">
          <FormField v-slot="{ componentField, handleChange }" name="config.features.file_upload">
            <FormItem class="flex flex-row items-center justify-between box p-4">
              <div class="space-y-0.5">
                <FormLabel class="text-base">{{
                  $t('admin.inbox.livechat.features.fileUpload')
                }}</FormLabel>
                <FormDescription>{{
                  $t('admin.inbox.livechat.features.fileUpload.description')
                }}</FormDescription>
              </div>
              <FormControl>
                <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
              </FormControl>
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField, handleChange }" name="config.features.emoji">
            <FormItem class="flex flex-row items-center justify-between box p-4">
              <div class="space-y-0.5">
                <FormLabel class="text-base">{{
                  $t('admin.inbox.livechat.features.emoji')
                }}</FormLabel>
                <FormDescription>{{
                  $t('admin.inbox.livechat.features.emoji.description')
                }}</FormDescription>
              </div>
              <FormControl>
                <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
              </FormControl>
            </FormItem>
          </FormField>
        </div>
      </div>

      <!-- External Links -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.externalLinks') }}</h4>

        <FormField name="config.external_links">
          <FormItem>
            <div class="space-y-3">
              <div
                v-for="(link, index) in externalLinks"
                :key="index"
                class="flex items-center gap-2 p-3 border rounded"
              >
                <div class="flex-1 grid grid-cols-2 gap-2">
                  <Input v-model="link.text" placeholder="Link Text" @input="updateExternalLinks" />
                  <Input
                    v-model="link.url"
                    placeholder="https://example.com"
                    @input="updateExternalLinks"
                  />
                </div>
                <Button type="button" variant="ghost" size="sm" @click="removeExternalLink(index)">
                  <X class="w-4 h-4" />
                </Button>
              </div>

              <Button type="button" variant="outline" size="sm" @click="addExternalLink">
                <Plus class="w-4 h-4 mr-2" />
                {{ $t('admin.inbox.livechat.externalLinks.add') }}
              </Button>
            </div>
            <FormDescription>
              {{ $t('admin.inbox.livechat.externalLinks.description') }}
            </FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>

      <!-- Trusted Domains -->
      <div class="space-y-4">
        <h4 class="font-medium text-foreground">{{ $t('admin.inbox.livechat.trustedDomains') }}</h4>

        <FormField v-slot="{ componentField }" name="config.trusted_domains">
          <FormItem>
            <FormLabel>{{ $t('admin.inbox.livechat.trustedDomains.list') }}</FormLabel>
            <FormControl>
              <Textarea
                v-bind="componentField"
                placeholder="example.com&#10;subdomain.example.com&#10;another-domain.com"
                rows="4"
              />
            </FormControl>
            <FormDescription>{{
              $t('admin.inbox.livechat.trustedDomains.description')
            }}</FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
      </div>
    </div>

    <!-- User-specific Settings with Tabs -->
    <div class="box p-4 space-y-6">
      <h3 class="font-semibold">{{ $t('admin.inbox.livechat.userSettings') }}</h3>

      <Tabs :model-value="selectedUserTab" @update:model-value="selectedUserTab = $event">
        <TabsList class="grid w-full grid-cols-2">
          <TabsTrigger value="visitors">
            {{ $t('admin.inbox.livechat.userSettings.visitors') }}
          </TabsTrigger>
          <TabsTrigger value="users">
            {{ $t('admin.inbox.livechat.userSettings.users') }}
          </TabsTrigger>
        </TabsList>

        <div class="space-y-4 mt-4">
          <!-- Visitors Settings -->
          <div v-show="selectedUserTab === 'visitors'" class="space-y-4">
            <FormField
              v-slot="{ componentField }"
              name="config.visitors.start_conversation_button_text"
            >
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.startConversationButtonText') }}</FormLabel>
                <FormControl>
                  <Input v-bind="componentField" placeholder="Start conversation" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField
              v-slot="{ componentField, handleChange }"
              name="config.visitors.allow_start_conversation"
            >
              <FormItem class="flex flex-row items-center justify-between box p-4">
                <div class="space-y-0.5">
                  <FormLabel class="text-base">{{
                    $t('admin.inbox.livechat.allowStartConversation')
                  }}</FormLabel>
                  <FormDescription>{{
                    $t('admin.inbox.livechat.allowStartConversation.visitors.description')
                  }}</FormDescription>
                </div>
                <FormControl>
                  <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
                </FormControl>
              </FormItem>
            </FormField>

            <FormField
              v-slot="{ componentField, handleChange }"
              name="config.visitors.prevent_multiple_conversations"
            >
              <FormItem class="flex flex-row items-center justify-between box p-4">
                <div class="space-y-0.5">
                  <FormLabel class="text-base">{{
                    $t('admin.inbox.livechat.preventMultipleConversations')
                  }}</FormLabel>
                  <FormDescription>{{
                    $t('admin.inbox.livechat.preventMultipleConversations.visitors.description')
                  }}</FormDescription>
                </div>
                <FormControl>
                  <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
                </FormControl>
              </FormItem>
            </FormField>
          </div>

          <!-- Users Settings -->
          <div v-show="selectedUserTab === 'users'" class="space-y-4">
            <FormField
              v-slot="{ componentField }"
              name="config.users.start_conversation_button_text"
            >
              <FormItem>
                <FormLabel>{{ $t('admin.inbox.livechat.startConversationButtonText') }}</FormLabel>
                <FormControl>
                  <Input v-bind="componentField" placeholder="Start conversation" />
                </FormControl>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField
              v-slot="{ componentField, handleChange }"
              name="config.users.allow_start_conversation"
            >
              <FormItem class="flex flex-row items-center justify-between box p-4">
                <div class="space-y-0.5">
                  <FormLabel class="text-base">{{
                    $t('admin.inbox.livechat.allowStartConversation')
                  }}</FormLabel>
                  <FormDescription>{{
                    $t('admin.inbox.livechat.allowStartConversation.users.description')
                  }}</FormDescription>
                </div>
                <FormControl>
                  <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
                </FormControl>
              </FormItem>
            </FormField>

            <FormField
              v-slot="{ componentField, handleChange }"
              name="config.users.prevent_multiple_conversations"
            >
              <FormItem class="flex flex-row items-center justify-between box p-4">
                <div class="space-y-0.5">
                  <FormLabel class="text-base">{{
                    $t('admin.inbox.livechat.preventMultipleConversations')
                  }}</FormLabel>
                  <FormDescription>{{
                    $t('admin.inbox.livechat.preventMultipleConversations.users.description')
                  }}</FormDescription>
                </div>
                <FormControl>
                  <Switch :checked="componentField.modelValue" @update:checked="handleChange" />
                </FormControl>
              </FormItem>
            </FormField>
          </div>
        </div>
      </Tabs>
    </div>

    <Button type="submit" :is-loading="isLoading" :disabled="isLoading">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch, computed, ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './livechatFormSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Switch } from '@shared-ui/components/ui/switch'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { Copy, Plus, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const { t } = useI18n()
const selectedUserTab = ref('visitors')
const externalLinks = ref([])

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    name: '',
    enabled: true,
    csat_enabled: false,
    config: {
      brand_name: '',
      logo_url: '',
      secret_key: '',
      launcher: {
        position: 'right',
        logo_url: '',
        spacing: {
          side: 20,
          bottom: 20
        }
      },
      greeting_message: '',
      introduction_message: '',
      chat_introduction: 'Ask us anything, or share your feedback.',
      show_office_hours_in_chat: false,
      show_office_hours_after_assignment: false,
      notice_banner: {
        enabled: false,
        text: 'Our response times are slower than usual. We regret the inconvenience caused.'
      },
      colors: {
        primary: '#2563eb',
      },
      features: {
        file_upload: true,
        emoji: true
      },
      trusted_domains: '',
      external_links: [],
      visitors: {
        start_conversation_button_text: 'Start conversation',
        allow_start_conversation: true,
        prevent_multiple_conversations: false
      },
      users: {
        start_conversation_button_text: 'Start conversation',
        allow_start_conversation: true,
        prevent_multiple_conversations: false
      }
    }
  }
})

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.messages.save')
})

const hasSecretKey = computed(() => {
  return form.values.config?.secret_key && form.values.config.secret_key.trim() !== ''
})

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    // You could emit a toast here for success feedback
  } catch (err) {
    console.error('Failed to copy text: ', err)
  }
}

const addExternalLink = () => {
  externalLinks.value.push({ text: '', url: '' })
  updateExternalLinks()
}

const removeExternalLink = (index) => {
  externalLinks.value.splice(index, 1)
  updateExternalLinks()
}

const updateExternalLinks = () => {
  form.setFieldValue('config.external_links', externalLinks.value)
}

const onSubmit = form.handleSubmit(async (values) => {
  // Transform trusted_domains from textarea to array
  if (values.config.trusted_domains) {
    values.config.trusted_domains = values.config.trusted_domains
      .split('\n')
      .map((domain) => domain.trim())
      .filter((domain) => domain)
  }

  // Filter out incomplete external links before submission
  if (values.config.external_links) {
    values.config.external_links = values.config.external_links.filter(
      (link) => link.text && link.url
    )
  }

  await props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) {
      return
    }

    // Transform trusted_domains array back to textarea format
    if (newValues.config?.trusted_domains && Array.isArray(newValues.config.trusted_domains)) {
      newValues.config.trusted_domains = newValues.config.trusted_domains.join('\n')
    }

    // Set external links for the reactive array
    if (newValues.config?.external_links) {
      externalLinks.value = [...newValues.config.external_links]
    }
    form.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
