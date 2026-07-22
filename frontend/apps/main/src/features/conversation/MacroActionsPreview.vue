<template>
  <div class="flex flex-wrap">
    <div class="flex flex-wrap gap-2">
      <div
        v-for="action in actions"
        :key="action.type"
        class="flex items-center border bg-background rounded-md transition-colors duration-150 hover:bg-accent/50 group gap-2"
      >
        <div class="flex items-center space-x-2 px-2">
          <component
            :is="getIcon(action.type)"
            size="16"
            class="text-muted-foreground"
          />
          <Tooltip>
            <TooltipTrigger as-child>
              <div
                class="max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-foreground">
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
          class="p-2 text-muted-foreground hover:text-destructive focus:outline-none rounded-md transition-colors duration-150"
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
  const prefixes = {
    assign_team: t('actions.assignTeam'),
    assign_user: t('actions.assignAgent'),
    set_status: t('actions.setStatus'),
    set_priority: t('actions.setPriority'),
    add_tags: t('actions.addTags'),
    set_tags: t('actions.setTags'),
    remove_tags: t('actions.removeTags')
  }
  const prefix = prefixes[action.type] || action.type
  return `${prefix}: ${getDisplayValue(action)}`
}
</script>
