<template>
  <AdminPageWithHelp>
    <template #content>
      <Spinner v-if="pageLoading" />
      <div v-else>
        <div class="space-y-6">
          <div>
            <h3 class="text-lg font-medium">Shopify Integration</h3>
            <p class="text-sm text-muted-foreground">
              Connect your Shopify store to view customer profiles, order history, and lifetime
              spend directly in the conversation sidebar.
            </p>
          </div>

          <!-- OAuth success banner -->
          <div v-if="oauthSuccess" class="rounded-md border border-green-300 bg-green-50 p-4">
            <div class="flex items-center gap-2">
              <CheckCircle2 class="h-5 w-5 text-green-600" />
              <span class="text-sm font-medium text-green-800">
                Shopify app installed successfully. Your access token has been saved.
              </span>
            </div>
          </div>

          <form @submit.prevent="saveIntegration" class="space-y-4">
            <div class="space-y-2">
              <Label for="store_url">Store URL</Label>
              <Input
                id="store_url"
                v-model="form.store_url"
                placeholder="yourstore.myshopify.com"
              />
              <p class="text-xs text-muted-foreground">
                Your Shopify store's myshopify.com domain (without https://).
              </p>
            </div>

            <div class="space-y-2">
              <Label for="api_key">API Key (Client ID)</Label>
              <Input
                id="api_key"
                v-model="form.api_key"
                placeholder="e.g. 1a2b3c4d5e6f..."
              />
              <p class="text-xs text-muted-foreground">
                Found in your Shopify Custom App's "Client credentials" section.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="api_secret">API Secret Key (Client Secret)</Label>
              <Input
                id="api_secret"
                v-model="form.api_secret"
                type="password"
                placeholder="shpss_..."
              />
              <p class="text-xs text-muted-foreground">
                Found in your Shopify Custom App's "Client credentials" section.
              </p>
            </div>

            <div class="flex items-center space-x-2">
              <Switch id="enabled" :checked="form.enabled" @update:checked="form.enabled = $event" />
              <Label for="enabled">Enabled</Label>
            </div>

            <div class="flex flex-wrap gap-3 pt-2">
              <Button type="submit" :disabled="saving">
                {{ saving ? 'Saving...' : (isConfigured ? 'Update' : 'Save') }}
              </Button>
              <Button
                v-if="isConfigured && !hasAccessToken"
                type="button"
                variant="default"
                :disabled="authorizing"
                @click="startOAuth"
              >
                <ExternalLink class="h-4 w-4 mr-1" />
                {{ authorizing ? 'Redirecting...' : 'Install on Shopify' }}
              </Button>
              <Button
                v-if="isConfigured && hasAccessToken"
                type="button"
                variant="outline"
                :disabled="testing"
                @click="testConnection"
              >
                {{ testing ? 'Testing...' : 'Test Connection' }}
              </Button>
              <Button
                v-if="isConfigured"
                type="button"
                variant="destructive"
                :disabled="deleting"
                @click="deleteIntegration"
              >
                {{ deleting ? 'Removing...' : 'Remove' }}
              </Button>
            </div>
          </form>

          <!-- Connection status -->
          <div v-if="hasAccessToken && !testResult && !testError" class="rounded-md border p-4">
            <div class="flex items-center gap-2">
              <CheckCircle2 class="h-5 w-5 text-green-600" />
              <span class="text-sm font-medium">Access token configured</span>
            </div>
          </div>

          <div v-if="testResult" class="rounded-md border p-4">
            <div class="flex items-center gap-2">
              <CheckCircle2 class="h-5 w-5 text-green-600" />
              <span class="text-sm font-medium">Connected to {{ testResult.shop_name }}</span>
            </div>
          </div>

          <div v-if="testError" class="rounded-md border border-destructive p-4">
            <div class="flex items-center gap-2">
              <XCircle class="h-5 w-5 text-destructive" />
              <span class="text-sm text-destructive">{{ testError }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #help>
      <p>
        The Shopify integration enriches conversations with customer data from your Shopify store.
      </p>
      <p>
        When an agent opens a conversation, the sidebar will automatically display the customer's
        profile, recent orders, and lifetime spend based on their email address.
      </p>
      <p class="font-medium mt-2">Setup steps:</p>
      <ol class="list-decimal ml-4 space-y-1 text-sm">
        <li>Go to your Shopify Admin &rarr; Settings &rarr; Apps and sales channels</li>
        <li>Click "Develop apps" &rarr; "Create an app"</li>
        <li>
          Under "Configuration &rarr; Admin API access scopes", select
          <code class="text-xs bg-muted px-1 py-0.5 rounded">read_customers</code> and
          <code class="text-xs bg-muted px-1 py-0.5 rounded">read_orders</code>
        </li>
        <li>
          Set the <strong>App URL</strong> to:
          <code class="text-xs bg-muted px-1 py-0.5 rounded select-all">{{ rootURL }}</code>
        </li>
        <li>
          Add <strong>Allowed redirection URL</strong>:
          <code class="text-xs bg-muted px-1 py-0.5 rounded select-all break-all">{{ callbackURL }}</code>
        </li>
        <li>Copy the <strong>Client ID</strong> and <strong>Client secret</strong> and paste them here</li>
        <li>Click <strong>Save</strong>, then click <strong>Install on Shopify</strong></li>
      </ol>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Spinner } from '@/components/ui/spinner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { CheckCircle2, XCircle, ExternalLink } from 'lucide-vue-next'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { useIntegrationStore } from '@/stores/integration'
import { useAppSettingsStore } from '@/stores/appSettings'

const emitter = useEmitter()
const appSettingsStore = useAppSettingsStore()
const route = useRoute()
const integrationStore = useIntegrationStore()
const pageLoading = ref(true)
const saving = ref(false)
const testing = ref(false)
const deleting = ref(false)
const authorizing = ref(false)
const testResult = ref(null)
const testError = ref(null)
const oauthSuccess = ref(false)

const form = ref({
  store_url: '',
  api_key: '',
  api_secret: '',
  enabled: true
})

const isConfigured = computed(() => integrationStore.shopifyConfigured)
const rootURL = computed(() => appSettingsStore.settings['app.root_url'] || window.location.origin)
const callbackURL = computed(() => `${rootURL.value}/api/v1/integrations/shopify/oauth/callback`)
const hasAccessToken = ref(false)
const isDummy = (val) => val && val.includes('\u2022')

onMounted(async () => {
  if (route.query.oauth === 'success') {
    oauthSuccess.value = true
    await integrationStore.fetchIntegrations()
  }

  try {
    const resp = await api.getIntegration('shopify')
    const data = resp.data.data
    const cfg = data.config || {}
    form.value = {
      store_url: cfg.store_url || '',
      api_key: cfg.api_key || '',
      api_secret: cfg.api_secret || '',
      enabled: data.enabled
    }
    hasAccessToken.value = !!(cfg.access_token && cfg.access_token.length > 0)
  } catch {
    // Not configured yet
  } finally {
    pageLoading.value = false
  }
})

const saveIntegration = async () => {
  saving.value = true
  testResult.value = null
  testError.value = null
  try {
    const config = {
      store_url: form.value.store_url
    }
    if (!isDummy(form.value.api_key)) {
      config.api_key = form.value.api_key
    }
    if (!isDummy(form.value.api_secret)) {
      config.api_secret = form.value.api_secret
    }

    const payload = {
      provider: 'shopify',
      enabled: form.value.enabled,
      config
    }

    await api.createIntegration(payload)
    await integrationStore.fetchIntegrations()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Shopify integration saved.' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    saving.value = false
  }
}

const startOAuth = async () => {
  authorizing.value = true
  try {
    const resp = await api.getShopifyOAuthURL()
    const url = resp.data.data.authorize_url
    window.location.href = url
  } catch (error) {
    authorizing.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const testConnection = async () => {
  testing.value = true
  testResult.value = null
  testError.value = null
  try {
    const resp = await api.testIntegration('shopify')
    testResult.value = resp.data.data
  } catch (error) {
    testError.value = handleHTTPError(error).message
  } finally {
    testing.value = false
  }
}

const deleteIntegration = async () => {
  deleting.value = true
  try {
    await api.deleteIntegration('shopify')
    await integrationStore.fetchIntegrations()
    form.value = { store_url: '', api_key: '', api_secret: '', enabled: true }
    hasAccessToken.value = false
    testResult.value = null
    oauthSuccess.value = false
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Shopify integration removed.' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    deleting.value = false
  }
}
</script>
