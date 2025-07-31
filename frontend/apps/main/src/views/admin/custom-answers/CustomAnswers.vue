<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="space-y-4">
        <div>
          <h3 class="font-semibold text-sm mb-2">{{ $t('ai.customAnswer.helpTitle') }}</h3>
          <p class="text-sm text-gray-600 mb-3">{{ $t('ai.customAnswer.helpDescription') }}</p>
        </div>

        <div>
          <h4 class="font-medium text-sm mb-2">{{ $t('ai.customAnswer.helpHowItWorks') }}</h4>
          <ul class="text-sm text-gray-600 space-y-1 list-disc list-inside">
            <li>{{ $t('ai.customAnswer.helpStep1') }}</li>
            <li>{{ $t('ai.customAnswer.helpStep2') }}</li>
            <li>{{ $t('ai.customAnswer.helpStep3') }}</li>
          </ul>
        </div>

        <div>
          <h4 class="font-medium text-sm mb-2">{{ $t('ai.customAnswer.helpBestPractices') }}</h4>
          <ul class="text-sm text-gray-600 space-y-1 list-disc list-inside">
            <li>{{ $t('ai.customAnswer.helpTip1') }}</li>
            <li>{{ $t('ai.customAnswer.helpTip2') }}</li>
            <li>{{ $t('ai.customAnswer.helpTip3') }}</li>
          </ul>
        </div>

        <div class="p-3 bg-blue-50 rounded-lg">
          <div class="flex items-start">
            <div class="flex-shrink-0">
              <svg class="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="ml-3">
              <p class="text-sm text-blue-700">{{ $t('ai.customAnswer.helpNote') }}</p>
            </div>
          </div>
        </div>
      </div>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>