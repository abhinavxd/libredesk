import { createRouter, createWebHistory } from 'vue-router'
import App from '@main/App.vue'
import OuterApp from '@main/OuterApp.vue'
import InboxLayout from '@main/layouts/inbox/InboxLayout.vue'
import AccountLayout from '@main/layouts/account/AccountLayout.vue'
import AdminLayout from '@main/layouts/admin/AdminLayout.vue'
import { useAppSettingsStore } from '../stores/appSettings'
import { getI18n } from '../i18n'

const routes = [
  {
    path: '/',
    component: OuterApp,
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('@main/views/auth/UserLoginView.vue'),
        meta: { titleKey: 'auth.signInButton' }
      },
      {
        path: 'reset-password',
        name: 'reset-password',
        component: () => import('@main/views/auth/ResetPasswordView.vue'),
        meta: { titleKey: 'auth.resetPassword' }
      },
      {
        path: 'set-password',
        name: 'set-password',
        component: () => import('@main/views/auth/SetPasswordView.vue'),
        meta: { titleKey: 'auth.setNewPassword' }
      }
    ]
  },
  {
    path: '/',
    component: App,
    children: [
      {
        path: 'contacts',
        name: 'contacts',
        component: () => import('@main/views/contact/ContactsView.vue'),
        meta: { titleKey: 'contact.allContacts' }
      },
      {
        path: 'contacts/:id',
        name: 'contact-detail',
        component: () => import('@main/views/contact/ContactDetailView.vue'),
        meta: { titleKey: 'globals.terms.contact', titleCount: 2 }
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
            meta: { titleKey: 'globals.terms.overview' }
          }
        ]
      },
      {
        path: '/inboxes/teams/:teamID',
        name: 'teams',
        props: true,
        component: InboxLayout,
        meta: { titleKey: 'globals.terms.teamInbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'team-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { titleKey: 'globals.terms.teamInbox' },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'team-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { titleKey: 'globals.terms.teamInbox', hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: '/inboxes/views/:viewID',
        name: 'views',
        props: true,
        component: InboxLayout,
        meta: { titleKey: 'globals.terms.inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'view-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { titleKey: 'globals.terms.inbox' },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'view-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { titleKey: 'globals.terms.inbox', hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: 'inboxes/search',
        name: 'search',
        component: () => import('@main/views/search/SearchView.vue'),
        meta: { titleKey: 'globals.terms.search', hidePageHeader: true }
      },
      {
        path: '/inboxes/:type(assigned|unassigned|all|mentioned)?',
        name: 'inboxes',
        redirect: '/inboxes/assigned',
        component: InboxLayout,
        props: true,
        meta: { titleKey: 'globals.terms.inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: {
              titleKey: 'globals.terms.inbox',
              typeKey: (route) => {
                if (route.params.type === 'assigned') return 'conversation.myInbox'
                if (route.params.type === 'mentioned') return 'conversation.mentions'
                if (route.params.type === 'unassigned') return 'conversation.unassigned'
                if (route.params.type === 'all') return 'conversation.all'
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
                  titleKey: 'globals.terms.inbox',
                  typeKey: (route) => {
                    if (route.params.type === 'assigned') return 'conversation.myInbox'
                    if (route.params.type === 'mentioned') return 'conversation.mentions'
                    if (route.params.type === 'unassigned') return 'conversation.unassigned'
                    if (route.params.type === 'all') return 'conversation.all'
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
        component: AccountLayout,
        props: true,
        meta: { titleKey: 'globals.terms.account' },
        children: [
          {
            path: 'profile',
            name: 'profile',
            component: () => import('@main/views/account/profile/ProfileEditView.vue'),
            meta: { titleKey: 'account.editProfile' }
          }
        ]
      },
      {
        path: '/admin',
        name: 'admin',
        component: AdminLayout,
        meta: { titleKey: 'globals.terms.admin' },
        children: [
          {
            path: 'custom-attributes',
            name: 'custom-attributes',
            component: () => import('@main/views/admin/custom-attributes/CustomAttributes.vue'),
            meta: { titleKey: 'globals.terms.customAttribute', titleCount: 2 }
          },
          {
            path: 'general',
            name: 'general',
            component: () => import('@main/views/admin/general/General.vue'),
            meta: { titleKey: 'globals.terms.general' }
          },
          {
            path: 'business-hours',
            component: () => import('@main/views/admin/business-hours/BusinessHours.vue'),
            meta: { titleKey: 'globals.terms.businessHour', titleCount: 2 },
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
                meta: { titleKey: 'businessHour.new' }
              },
              {
                path: ':id/edit',
                name: 'edit-business-hours',
                props: true,
                component: () =>
                  import('@main/views/admin/business-hours/CreateOrEditBusinessHours.vue'),
                meta: { titleKey: 'businessHour.edit' }
              }
            ]
          },
          {
            path: 'sla',
            component: () => import('@main/views/admin/sla/SLA.vue'),
            meta: { titleKey: 'globals.terms.sla' },
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
                meta: { titleKey: 'sla.new' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sla',
                component: () => import('@main/views/admin/sla/CreateEditSLA.vue'),
                meta: { titleKey: 'sla.edit' }
              }
            ]
          },
          {
            path: 'inboxes',
            component: () => import('@main/views/admin/inbox/InboxView.vue'),
            meta: { titleKey: 'globals.terms.inbox', titleCount: 2 },
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
                meta: { titleKey: 'inbox.new' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-inbox',
                component: () => import('@main/views/admin/inbox/EditInbox.vue'),
                meta: { titleKey: 'inbox.edit' }
              }
            ]
          },
          {
            path: 'notification',
            component: () => import('@main/features/admin/notification/NotificationSetting.vue'),
            meta: { titleKey: 'globals.terms.notification', titleCount: 2 }
          },
          {
            path: 'teams',
            meta: { titleKey: 'globals.terms.team', titleCount: 2 },
            children: [
              {
                path: 'agents',
                component: () => import('@main/views/admin/agents/Agents.vue'),
                meta: { titleKey: 'globals.terms.agent', titleCount: 2 },
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
                    meta: { titleKey: 'agent.new' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-agent',
                    component: () => import('@main/views/admin/agents/EditAgent.vue'),
                    meta: { titleKey: 'agent.edit' }
                  }
                ]
              },
              {
                path: 'teams',
                component: () => import('@main/views/admin/teams/Teams.vue'),
                meta: { titleKey: 'globals.terms.team', titleCount: 2 },
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
                    meta: { titleKey: 'team.new' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-team',
                    component: () => import('@main/views/admin/teams/EditTeamForm.vue'),
                    meta: { titleKey: 'team.edit' }
                  }
                ]
              },
              {
                path: 'roles',
                component: () => import('@main/views/admin/roles/Roles.vue'),
                meta: { titleKey: 'globals.terms.role', titleCount: 2 },
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
                    meta: { titleKey: 'role.new' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-role',
                    component: () => import('@main/views/admin/roles/EditRole.vue'),
                    meta: { titleKey: 'role.edit' }
                  }
                ]
              },
              {
                path: 'activity-log',
                name: 'activity-log',
                component: () => import('@main/views/admin/activity-log/ActivityLog.vue'),
                meta: { titleKey: 'globals.terms.activityLog', titleCount: 2 }
              }
            ]
          },
          {
            path: 'automations',
            component: () => import('@main/views/admin/automations/Automation.vue'),
            name: 'automations',
            meta: { titleKey: 'globals.terms.automation', titleCount: 2 },
            children: [
              {
                path: 'new',
                props: true,
                name: 'new-automation',
                component: () => import('@main/views/admin/automations/CreateOrEditRule.vue'),
                meta: { titleKey: 'automation.newRule' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-automation',
                component: () => import('@main/views/admin/automations/CreateOrEditRule.vue'),
                meta: { titleKey: 'automation.editRule' }
              }
            ]
          },
          {
            path: 'templates',
            component: () => import('@main/views/admin/templates/Templates.vue'),
            name: 'templates',
            meta: { titleKey: 'globals.terms.template', titleCount: 2 },
            children: [
              {
                path: ':id/edit',
                name: 'edit-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { titleKey: 'template.edit' }
              },
              {
                path: 'new',
                name: 'new-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { titleKey: 'template.new' }
              }
            ]
          },
          {
            path: 'sso',
            component: () => import('@main/views/admin/oidc/OIDC.vue'),
            name: 'sso',
            meta: { titleKey: 'globals.terms.sso' },
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
                meta: { titleKey: 'oidc.edit' }
              },
              {
                path: 'new',
                name: 'new-sso',
                component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { titleKey: 'oidc.new' }
              }
            ]
          },
          {
            path: 'webhooks',
            component: () => import('@main/views/admin/webhooks/Webhooks.vue'),
            name: 'webhooks',
            meta: { titleKey: 'globals.terms.webhook', titleCount: 2 },
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
                meta: { titleKey: 'webhook.edit' }
              },
              {
                path: 'new',
                name: 'new-webhook',
                component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'),
                meta: { titleKey: 'webhook.new' }
              }
            ]
          },
          {
            path: 'conversations',
            meta: { titleKey: 'globals.terms.conversation', titleCount: 2 },
            children: [
              {
                path: 'tags',
                component: () => import('@main/views/admin/tags/TagsView.vue'),
                meta: { titleKey: 'globals.terms.tag', titleCount: 2 }
              },
              {
                path: 'statuses',
                component: () => import('@main/views/admin/status/StatusView.vue'),
                meta: { titleKey: 'globals.terms.status', titleCount: 2 }
              },
              {
                path: 'macros',
                component: () => import('@main/views/admin/macros/Macros.vue'),
                meta: { titleKey: 'globals.terms.macro', titleCount: 2 },
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
                    meta: { titleKey: 'macro.new' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-macro',
                    component: () => import('@main/views/admin/macros/EditMacro.vue'),
                    meta: { titleKey: 'macro.editMacro' }
                  }
                ]
              },
              {
                path: 'shared-views',
                component: () => import('@main/views/admin/shared-views/SharedViews.vue'),
                meta: { titleKey: 'globals.terms.sharedView', titleCount: 2 },
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
                    meta: { titleKey: 'sharedView.new' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-shared-view',
                    component: () => import('@main/views/admin/shared-views/EditSharedView.vue'),
                    meta: { titleKey: 'sharedView.editSharedView' }
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

router.beforeEach((to, from, next) => {
  const appSettingsStore = useAppSettingsStore()
  const siteName = appSettingsStore.settings?.['app.site_name'] || 'Libredesk'
  const i18n = getI18n()
  const typeKey = typeof to.meta?.typeKey === 'function' ? to.meta.typeKey(to) : ''
  const titleKey = typeKey || to.meta?.titleKey
  const pageTitle = titleKey && i18n
    ? i18n.global.t(titleKey, to.meta?.titleCount || 1)
    : ''
  document.title = `${pageTitle} - ${siteName}`
  next()
})

export default router
