<template>
  <div class="flex flex-wrap">
    <div class="flex flex-wrap gap-2">
      <div
        v-for="action in actions"
        :key="action.type"
        class="flex items-center border bg-background border-gray-200 rounded shadow-sm transition-all duration-300 ease-in-out hover:shadow-md group gap-2"
      >
        <div class="flex items-center space-x-2 px-2">
          <component
            :is="getIcon(action.type)"
            size="16"
            class="text-gray-500 text-primary"
          />
          <Tooltip>
            <TooltipTrigger as-child>
              <div
                class="max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-primary group-hover:text-gray-900 dark:group-hover:text-gray-100">
                {{ getDisplayValue(action) }}
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p class="text-sm">{{ getTooltip(action) }}</p>
            </TooltipContent>
          </Tooltip>
        </div>
        <button
          @click.prevent="onRemove(action)"
          class="p-2 text-gray-400 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50 rounded transition-colors duration-300 ease-in-out"
          title="Remove action"
        >
          <X size="14" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { X, Users, User, MessageSquare, Tags, Flag } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useI18n } from 'vue-i18n'

defineProps({
  actions: {
    type: Array,
    required: true
  },
  onRemove: {
    type: Function,
    required: true
  }
})

const { t } = useI18n()
const getIcon = (type) =>
  ({
    assign_team: Users,
    assign_user: User,
    set_status: MessageSquare,
    set_priority: Flag,
    add_tags: Tags,
    set_tags: Tags,
    remove_tags: Tags
  })[type]

const getDisplayValue = (action) => {
  if (action.display_value?.length) {
    return action.display_value.join(', ')
  }
  return action.value.join(', ')
}

const getTooltip = (action) => {
  switch (action.type) {
    case 'assign_team':
      return `${t('globals.messages.assign', { name: t('globals.terms.team').toLowerCase() })}: ${getDisplayValue(action)}`
    case 'assign_user':
      return `${t('globals.messages.assign', { name: t('globals.terms.agent').toLowerCase() })}: ${getDisplayValue(action)}`
    case 'set_status':
      return `${t('globals.messages.set', { name: t('globals.terms.status').toLowerCase() })}: ${getDisplayValue(action)}`
    case 'set_priority':
      return `${t('globals.messages.set', { name: t('globals.terms.priority').toLowerCase() })}: ${getDisplayValue(action)}`
    case 'add_tags':
      return `${t('globals.messages.add', { name: t('globals.terms.tag', 2).toLowerCase() })}: ${getDisplayValue(action)}`
    case 'set_tags':
      return `${t('globals.messages.set', { name: t('globals.terms.tag', 2).toLowerCase() })}: ${getDisplayValue(action)}`
    case 'remove_tags':
      return `${t('globals.messages.remove', { name: t('globals.terms.tag', 2).toLowerCase() })}: ${getDisplayValue(action)}`
    default:
      return `${t('globals.terms.action')}: ${action.type}, ${t('globals.terms.value').toLowerCase()}: ${getDisplayValue(action)}`
  }
}
</script>
