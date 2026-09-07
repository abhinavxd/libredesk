import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Eye, MessageSquarePlus } from 'lucide-vue-next'
import { navIconMap } from '@main/constants/navIcons'
import { adminNavItems } from '@main/constants/navigation'
import { permissions } from '@main/constants/permissions'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { SECTIONS } from '../sections'

export function useCreateCommands() {
  const router = useRouter()
  const { t } = useI18n()
  const emitter = useEmitter()

  const adminCreates = () =>
    adminNavItems
      .flatMap((group) => group.children)
      .filter((item) => item.createRouteName)
      .map((item) => ({
        id: `create.${item.createRouteName}`,
        label: t(router.resolve({ name: item.createRouteName }).meta.titleKey),
        keywords: [t('globals.messages.create'), t('globals.terms.admin')],
        section: SECTIONS.CREATE,
        icon: navIconMap[item.icon],
        permission: item.permission,
        run: () => router.push({ name: item.createRouteName })
      }))

  return computed(() => [
    {
      id: 'create.conversation',
      label: t('conversation.newConversation'),
      keywords: [t('globals.messages.create')],
      section: SECTIONS.ACTIONS,
      icon: MessageSquarePlus,
      permission: permissions.CONVERSATIONS_WRITE,
      shortcut: ['Alt', 'C'],
      run: () => emitter.emit(EMITTER_EVENTS.OPEN_CREATE_CONVERSATION, {})
    },
    {
      id: 'create.view',
      label: t('command.newView'),
      keywords: [t('globals.messages.create'), t('globals.terms.view')],
      section: SECTIONS.ACTIONS,
      icon: Eye,
      permission: permissions.VIEW_MANAGE,
      run: () => emitter.emit(EMITTER_EVENTS.OPEN_VIEW_FORM)
    },
    ...adminCreates()
  ])
}
