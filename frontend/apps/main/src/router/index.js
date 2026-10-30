
import { defineMessage as msg } from '@lingui/core/macro'
import { createRouter, createWebHistory } from 'vue-router'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { getI18n } from '@main/i18n'
import { abortRouteScope } from '@main/api'

const routes = [
  {
    path: '/',
    component: () => import('@main/OuterApp.vue'),
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('@main/views/auth/UserLoginView.vue'),
        meta: { titleMessage: msg({ message: 'Sign in' }) }
      },
      {
        path: 'reset-password',
        name: 'reset-password',
        component: () => import('@main/views/auth/ResetPasswordView.vue'),
        meta: { titleMessage: msg({ message: 'Reset password' }) }
      },
      {
        path: 'set-password',
        name: 'set-password',
        component: () => import('@main/views/auth/SetPasswordView.vue'),
        meta: { titleMessage: msg({ message: 'Set new password' }) }
      }
    ]
  },
  {
    path: '/',
    component: () => import('@main/App.vue'),
    children: [
      {
        path: 'contacts',
        name: 'contacts',
        component: () => import('@main/views/contact/ContactsView.vue'),
        meta: { titleMessage: msg({ message: 'All contacts' }) }
      },
      {
        path: 'contacts/:id',
        name: 'contact-detail',
        component: () => import('@main/views/contact/ContactDetailView.vue'),
        meta: { titleMessage: msg({ message: '{count, plural, one {Contact} other {Contacts}}' }), titleCount: 2 }
      },
      {
        path: '/reports',
        name: 'reports',
        redirect: '/reports/overview',
        children: [
          {
            path: 'overview',
            name: 'overview',
            component: () => import('@main/views/reports/OverviewView.vue'),
            meta: { titleMessage: msg({ message: 'Overview' }) }
          }
        ]
      },
      {
        path: '/inboxes/teams/:teamID',
        name: 'teams',
        props: true,
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        meta: { titleMessage: msg({ message: '{count, plural, one {Team Inbox} other {Team Inboxes}}' }), hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'team-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Team Inbox} other {Team Inboxes}}' }) },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'team-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { titleMessage: msg({ message: '{count, plural, one {Team Inbox} other {Team Inboxes}}' }), hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: '/inboxes/views/:viewID',
        name: 'views',
        props: true,
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        meta: { titleMessage: msg({ message: '{count, plural, one {View} other {Views}}' }), hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'view-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {View} other {Views}}' }) },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'view-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { titleMessage: msg({ message: '{count, plural, one {View} other {Views}}' }), hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: 'inboxes/search',
        name: 'search',
        component: () => import('@main/views/search/SearchView.vue'),
        meta: { titleMessage: msg({ message: 'Search' }), hidePageHeader: true }
      },
      {
        path: '/inboxes/:type(assigned|unassigned|all|mentioned)?',
        name: 'inboxes',
        redirect: '/inboxes/assigned',
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        props: true,
        meta: { titleMessage: msg({ message: '{count, plural, one {Inbox} other {Inboxes}}' }), hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: {
              titleMessage: msg({ message: '{count, plural, one {Inbox} other {Inboxes}}' }),
              typeMessage: (route) => {
                if (route.params.type === 'assigned') return msg({ message: 'My inbox' })
                if (route.params.type === 'mentioned') return msg({ message: 'Mentions' })
                if (route.params.type === 'unassigned') return msg({ message: 'Unassigned' })
                if (route.params.type === 'all') return msg({ message: 'All' })
                return ''
              }
            },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: {
                  titleMessage: msg({ message: '{count, plural, one {Inbox} other {Inboxes}}' }),
                  typeMessage: (route) => {
                    if (route.params.type === 'assigned') return msg({ message: 'My inbox' })
                    if (route.params.type === 'mentioned') return msg({ message: 'Mentions' })
                    if (route.params.type === 'unassigned') return msg({ message: 'Unassigned' })
                    if (route.params.type === 'all') return msg({ message: 'All' })
                    return ''
                  },
                  hidePageHeader: true
                }
              }
            ]
          }
        ]
      },
      {
        path: '/account/:page?',
        name: 'account',
        redirect: '/account/profile',
        component: () => import('@main/layouts/account/AccountLayout.vue'),
        props: true,
        meta: { titleMessage: msg({ message: '{count, plural, one {Account} other {Accounts}}' }) },
        children: [
          {
            path: 'profile',
            name: 'profile',
            component: () => import('@main/views/account/profile/ProfileEditView.vue'),
            meta: { titleMessage: msg({ message: 'Edit profile' }) }
          },
          {
            path: 'notifications',
            name: 'account-notifications',
            component: () =>
              import('@main/views/account/notifications/NotificationPreferences.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Notification} other {Notifications}}' }), titleCount: 2 }
          }
        ]
      },
      {
        path: '/admin',
        name: 'admin',
        component: () => import('@main/layouts/admin/AdminLayout.vue'),
        meta: { titleMessage: msg({ message: '{count, plural, one {Admin} other {Admins}}' }) },
        children: [
          {
            path: 'custom-attributes',
            name: 'custom-attributes',
            component: () => import('@main/views/admin/custom-attributes/CustomAttributes.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Custom attribute} other {Custom attributes}}' }), titleCount: 2 }
          },
          {
            path: 'general',
            name: 'general',
            component: () => import('@main/views/admin/general/General.vue'),
            meta: { titleMessage: msg({ message: 'General' }) }
          },
          {
            path: 'ai',
            redirect: { name: 'ai-providers' }
          },
          {
            path: 'help-center',
            component: () => import('@main/views/admin/help-center/HelpCenter.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Help center} other {Help centers}}' }) },
            children: [
              {
                path: '',
                name: 'help-center-list',
                component: () => import('@main/views/admin/help-center/HelpCenterList.vue')
              },
              {
                path: ':id/customize',
                name: 'help-center-customize',
                props: true,
                component: () => import('@main/views/admin/help-center/HelpCenterCustomize.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Help center} other {Help centers}}' }) }
              },
              {
                path: ':id/tree/:locale?',
                name: 'help-center-tree',
                props: true,
                component: () => import('@main/views/admin/help-center/HelpCenterTree.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Help center} other {Help centers}}' }) }
              }
            ]
          },
          {
            path: 'ai/providers',
            name: 'ai-providers',
            component: () => import('@main/views/admin/ai/AIProviders.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Provider} other {Providers}}' }), titleCount: 2 }
          },
          {
            path: 'ai/snippets',
            name: 'ai-snippets',
            component: () => import('@main/views/admin/ai/AISnippets.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Snippet} other {Snippets}}' }), titleCount: 2 }
          },
          {
            path: 'ai/editor-prompts',
            name: 'ai-editor-prompts',
            component: () => import('@main/views/admin/ai/AIEditorPrompts.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Editor prompt} other {Editor prompts}}' }), titleCount: 2 }
          },
          {
            path: 'ai/suggestions',
            name: 'ai-suggestions',
            component: () => import('@main/views/admin/ai/AISuggestions.vue'),
            meta: { titleMessage: msg({ message: 'Suggestions' }) }
          },
          {
            path: 'ai/tools',
            name: 'ai-tools',
            component: () => import('@main/views/admin/ai/AITools.vue'),
            meta: { titleMessage: msg({ message: 'Tools' }) }
          },
          {
            path: 'ai/assistants',
            name: 'ai-assistants',
            component: () => import('@main/views/admin/ai/AIAssistants.vue'),
            meta: { titleMessage: msg({ message: 'Assistants' }) }
          },
          {
            path: 'ai/tools/new',
            name: 'new-ai-tool',
            component: () => import('@main/views/admin/ai/CreateOrEditTool.vue'),
            meta: { titleMessage: msg({ message: 'New tool' }) }
          },
          {
            path: 'ai/tools/:id/edit',
            name: 'edit-ai-tool',
            props: true,
            component: () => import('@main/views/admin/ai/CreateOrEditTool.vue'),
            meta: { titleMessage: msg({ message: 'Edit tool' }) }
          },
          {
            path: 'ai/assistants/new',
            name: 'new-ai-assistant',
            component: () => import('@main/views/admin/ai/CreateOrEditAssistant.vue'),
            meta: { titleMessage: msg({ message: 'New assistant' }) }
          },
          {
            path: 'ai/assistants/:id/edit',
            name: 'edit-ai-assistant',
            props: true,
            component: () => import('@main/views/admin/ai/CreateOrEditAssistant.vue'),
            meta: { titleMessage: msg({ message: 'Edit assistant' }) }
          },
          {
            path: 'business-hours',
            component: () => import('@main/views/admin/business-hours/BusinessHours.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Business hour} other {Business hours}}' }), titleCount: 2 },
            children: [
              {
                path: '',
                name: 'business-hours-list',
                component: () => import('@main/views/admin/business-hours/BusinessHoursList.vue')
              },
              {
                path: 'new',
                name: 'new-business-hours',
                component: () =>
                  import('@main/views/admin/business-hours/CreateOrEditBusinessHours.vue'),
                meta: { titleMessage: msg({ message: 'New business hour' }) }
              },
              {
                path: ':id/edit',
                name: 'edit-business-hours',
                props: true,
                component: () =>
                  import('@main/views/admin/business-hours/CreateOrEditBusinessHours.vue'),
                meta: { titleMessage: msg({ message: 'Edit business hour' }) }
              }
            ]
          },
          {
            path: 'sla',
            component: () => import('@main/views/admin/sla/SLA.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {SLA} other {SLAs}}' }) },
            children: [
              {
                path: '',
                name: 'sla-list',
                component: () => import('@main/views/admin/sla/SLAList.vue')
              },
              {
                path: 'new',
                name: 'new-sla',
                component: () => import('@main/views/admin/sla/CreateEditSLA.vue'),
                meta: { titleMessage: msg({ message: 'New SLA policy' }) }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sla',
                component: () => import('@main/views/admin/sla/CreateEditSLA.vue'),
                meta: { titleMessage: msg({ message: 'Edit SLA policy' }) }
              }
            ]
          },
          {
            path: 'inboxes',
            component: () => import('@main/views/admin/inbox/InboxView.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Inbox} other {Inboxes}}' }), titleCount: 2 },
            children: [
              {
                path: '',
                name: 'inbox-list',
                component: () => import('@main/views/admin/inbox/InboxList.vue')
              },
              {
                path: 'new',
                name: 'new-inbox',
                component: () => import('@main/views/admin/inbox/NewInbox.vue'),
                meta: { titleMessage: msg({ message: 'New inbox' }) }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-inbox',
                component: () => import('@main/views/admin/inbox/EditInbox.vue'),
                meta: { titleMessage: msg({ message: 'Edit inbox' }) }
              }
            ]
          },
          {
            path: 'whatsapp/templates',
            meta: { titleMessage: msg({ message: 'WhatsApp templates' }) },
            children: [
              {
                path: '',
                name: 'whatsapp-templates',
                component: () => import('@main/views/admin/whatsapp/WhatsAppTemplateList.vue')
              },
              {
                path: 'new',
                name: 'whatsapp-template-new',
                component: () => import('@main/views/admin/whatsapp/CreateEditWhatsAppTemplate.vue'),
                meta: { titleMessage: msg({ message: 'New template' }) }
              },
              {
                path: ':id/edit',
                name: 'whatsapp-template-edit',
                component: () => import('@main/views/admin/whatsapp/CreateEditWhatsAppTemplate.vue'),
                props: true,
                meta: { titleMessage: msg({ message: 'Edit' }) }
              }
            ]
          },
          {
            path: 'notification',
            component: () => import('@main/features/admin/notification/NotificationSetting.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Notification} other {Notifications}}' }), titleCount: 2 }
          },
          {
            path: 'teams',
            meta: { titleMessage: msg({ message: '{count, plural, one {Team} other {Teams}}' }), titleCount: 2 },
            children: [
              {
                path: 'agents',
                component: () => import('@main/views/admin/agents/Agents.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Agent} other {Agents}}' }), titleCount: 2 },
                children: [
                  {
                    path: '',
                    name: 'agent-list',
                    component: () => import('@main/views/admin/agents/AgentList.vue')
                  },
                  {
                    path: 'new',
                    name: 'new-agent',
                    component: () => import('@main/views/admin/agents/CreateAgent.vue'),
                    meta: { titleMessage: msg({ message: 'New agent' }) }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-agent',
                    component: () => import('@main/views/admin/agents/EditAgent.vue'),
                    meta: { titleMessage: msg({ message: 'Edit agent' }) }
                  }
                ]
              },
              {
                path: 'teams',
                component: () => import('@main/views/admin/teams/Teams.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Team} other {Teams}}' }), titleCount: 2 },
                children: [
                  {
                    path: '',
                    name: 'team-list',
                    component: () => import('@main/views/admin/teams/TeamList.vue')
                  },
                  {
                    path: 'new',
                    name: 'new-team',
                    component: () => import('@main/views/admin/teams/CreateTeamForm.vue'),
                    meta: { titleMessage: msg({ message: 'New team' }) }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-team',
                    component: () => import('@main/views/admin/teams/EditTeamForm.vue'),
                    meta: { titleMessage: msg({ message: 'Edit team' }) }
                  }
                ]
              },
              {
                path: 'roles',
                component: () => import('@main/views/admin/roles/Roles.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Role} other {Roles}}' }), titleCount: 2 },
                children: [
                  {
                    path: '',
                    name: 'role-list',
                    component: () => import('@main/views/admin/roles/RoleList.vue')
                  },
                  {
                    path: 'new',
                    name: 'new-role',
                    component: () => import('@main/views/admin/roles/NewRole.vue'),
                    meta: { titleMessage: msg({ message: 'New role' }) }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-role',
                    component: () => import('@main/views/admin/roles/EditRole.vue'),
                    meta: { titleMessage: msg({ message: 'Edit role' }) }
                  }
                ]
              },
              {
                path: 'activity-log',
                name: 'activity-log',
                component: () => import('@main/views/admin/activity-log/ActivityLog.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Activity log} other {Activity logs}}' }), titleCount: 2 }
              }
            ]
          },
          {
            path: 'automations',
            component: () => import('@main/views/admin/automations/Automation.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Automation} other {Automations}}' }), titleCount: 2 },
            children: [
              {
                path: '',
                name: 'automation-list',
                component: () => import('@main/views/admin/automations/AutomationList.vue')
              },
              {
                path: 'new',
                props: true,
                name: 'new-automation',
                component: () => import('@main/views/admin/automations/CreateOrEditRule.vue'),
                meta: { titleMessage: msg({ message: 'New rule' }) }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-automation',
                component: () => import('@main/views/admin/automations/CreateOrEditRule.vue'),
                meta: { titleMessage: msg({ message: 'Edit rule' }) }
              }
            ]
          },
          {
            path: 'templates',
            component: () => import('@main/views/admin/templates/Templates.vue'),
            meta: { titleMessage: msg({ message: '{count, plural, one {Template} other {Templates}}' }), titleCount: 2 },
            children: [
              {
                path: '',
                name: 'template-list',
                component: () => import('@main/views/admin/templates/TemplateList.vue')
              },
              {
                path: ':id/edit',
                name: 'edit-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { titleMessage: msg({ message: 'Edit template' }) }
              },
              {
                path: 'new',
                name: 'new-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { titleMessage: msg({ message: 'New template' }) }
              }
            ]
          },
          {
            path: 'sso',
            component: () => import('@main/views/admin/oidc/OIDC.vue'),
            name: 'sso',
            meta: { titleMessage: msg({ message: '{count, plural, one {SSO} other {SSOs}}' }) },
            children: [
              {
                path: '',
                name: 'sso-list',
                component: () => import('@main/views/admin/oidc/OIDCList.vue')
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sso',
                component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { titleMessage: msg({ message: 'Edit SSO' }) }
              },
              {
                path: 'new',
                name: 'new-sso',
                component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { titleMessage: msg({ message: 'New SSO' }) }
              }
            ]
          },
          {
            path: 'webhooks',
            component: () => import('@main/views/admin/webhooks/Webhooks.vue'),
            name: 'webhooks',
            meta: { titleMessage: msg({ message: '{count, plural, one {Webhook} other {Webhooks}}' }), titleCount: 2 },
            children: [
              {
                path: '',
                name: 'webhook-list',
                component: () => import('@main/views/admin/webhooks/WebhookList.vue')
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-webhook',
                component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'),
                meta: { titleMessage: msg({ message: 'Edit webhook' }) }
              },
              {
                path: 'new',
                name: 'new-webhook',
                component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'),
                meta: { titleMessage: msg({ message: 'New webhook' }) }
              }
            ]
          },
          {
            path: 'context-links',
            component: () => import('@main/views/admin/context-links/ContextLinks.vue'),
            name: 'context-links',
            meta: { titleMessage: msg({ message: '{count, plural, one {Context link} other {Context links}}' }), titleCount: 2 },
            children: [
              {
                path: '',
                name: 'context-link-list',
                component: () => import('@main/views/admin/context-links/ContextLinkList.vue')
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-context-link',
                component: () =>
                  import('@main/views/admin/context-links/CreateEditContextLink.vue'),
                meta: { titleMessage: msg({ message: 'Edit context link' }) }
              },
              {
                path: 'new',
                name: 'new-context-link',
                component: () =>
                  import('@main/views/admin/context-links/CreateEditContextLink.vue'),
                meta: { titleMessage: msg({ message: 'New context link' }) }
              }
            ]
          },
          {
            path: 'conversations',
            meta: { titleMessage: msg({ message: '{count, plural, one {Conversation} other {Conversations}}' }), titleCount: 2 },
            children: [
              {
                path: 'tags',
                component: () => import('@main/views/admin/tags/TagsView.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Tag} other {Tags}}' }), titleCount: 2 }
              },
              {
                path: 'statuses',
                component: () => import('@main/views/admin/status/StatusView.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Status} other {Statuses}}' }), titleCount: 2 }
              },
              {
                path: 'macros',
                component: () => import('@main/views/admin/macros/Macros.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Macro} other {Macros}}' }), titleCount: 2 },
                children: [
                  {
                    path: '',
                    name: 'macro-list',
                    component: () => import('@main/views/admin/macros/MacroList.vue')
                  },
                  {
                    path: 'new',
                    name: 'new-macro',
                    component: () => import('@main/views/admin/macros/CreateMacro.vue'),
                    meta: { titleMessage: msg({ message: 'New macro' }) }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-macro',
                    component: () => import('@main/views/admin/macros/EditMacro.vue'),
                    meta: { titleMessage: msg({ message: 'Edit macro' }) }
                  }
                ]
              },
              {
                path: 'shared-views',
                component: () => import('@main/views/admin/shared-views/SharedViews.vue'),
                meta: { titleMessage: msg({ message: '{count, plural, one {Shared view} other {Shared views}}' }), titleCount: 2 },
                children: [
                  {
                    path: '',
                    name: 'shared-view-list',
                    component: () => import('@main/views/admin/shared-views/SharedViewList.vue')
                  },
                  {
                    path: 'new',
                    name: 'new-shared-view',
                    component: () => import('@main/views/admin/shared-views/CreateSharedView.vue'),
                    meta: { titleMessage: msg({ message: 'New shared view' }) }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-shared-view',
                    component: () => import('@main/views/admin/shared-views/EditSharedView.vue'),
                    meta: { titleMessage: msg({ message: 'Edit shared view' }) }
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: () => {
      return '/inboxes/assigned'
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes
})

const preloadErrors = new WeakSet()
const reloadKey = 'libredesk:preload-reload'

window.addEventListener('vite:preloadError', (event) => {
  preloadErrors.add(event.payload)
})

router.onError((error, to) => {
  if (!preloadErrors.has(error) || sessionStorage.getItem(reloadKey) === to.fullPath) return
  sessionStorage.setItem(reloadKey, to.fullPath)
  window.location.assign(to.fullPath)
})

router.afterEach((to, from, failure) => {
  if (!failure) sessionStorage.removeItem(reloadKey)
})

router.beforeEach((to, from, next) => {
  // Cancel in-flight requests.
  if (to.fullPath !== from.fullPath) {
    abortRouteScope()
  }

  const appSettingsStore = useAppSettingsStore()
  const siteName = appSettingsStore.settings?.['app.site_name'] || 'libredesk'
  const i18n = getI18n()
  const typeMessage = typeof to.meta?.typeMessage === 'function' ? to.meta.typeMessage(to) : ''
  const titleMessage = typeMessage || to.meta?.titleMessage
  const pageTitle = titleMessage && i18n
    ? i18n.global._(titleMessage, to.meta?.titleCount || 1)
    : ''
  document.title = `${pageTitle} - ${siteName}`
  next()
})

export default router
