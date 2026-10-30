

import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
import { Eye, MessageSquarePlus } from 'lucide-vue-next'
import { navIconMap } from '@main/constants/navIcons'
import { adminNavItems } from '@main/constants/navigation'
import { permissions } from '@main/constants/permissions'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { SECTIONS } from '../sections'

export function useCreateCommands() {
  const router = useRouter()
  const i18n = useI18n()
  const emitter = useEmitter()

  const adminCreates = () =>
    adminNavItems
      .flatMap((group) => group.children)
      .filter((item) => item.createRouteName)
      .map((item) => ({
        id: `create.${item.createRouteName}`,
        label: i18n._(router.resolve({ name: item.createRouteName }).meta.titleMessage),
        keywords: [i18n._('Create'), i18n._('{count, plural, one {Admin} other {Admins}}')],
        section: SECTIONS.CREATE,
        icon: navIconMap[item.icon],
        permission: item.permission,
        run: () => router.push({ name: item.createRouteName })
      }))

  return computed(() => [
    {
      id: 'create.conversation',
      label: i18n._('New conversation'),
      keywords: [i18n._('Create')],
      section: SECTIONS.ACTIONS,
      icon: MessageSquarePlus,
      permission: permissions.CONVERSATIONS_WRITE,
      shortcut: ['Alt', 'C'],
      run: () => emitter.emit(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION, {})
    },
    {
      id: 'create.view',
      label: i18n._('New view'),
      keywords: [i18n._('Create'), i18n._('{count, plural, one {View} other {Views}}')],
      section: SECTIONS.ACTIONS,
      icon: Eye,
      permission: permissions.VIEW_MANAGE,
      run: () => emitter.emit(EMITTER_EVENTS.OPEN_VIEW_FORM)
    },
    ...adminCreates()
  ])
}
