import { createRouter, createWebHistory } from 'vue-router'
import App from '@main/App.vue'
import OuterApp from '@main/OuterApp.vue'
import InboxLayout from '@main/layouts/inbox/InboxLayout.vue'
import AccountLayout from '@main/layouts/account/AccountLayout.vue'
import AdminLayout from '@main/layouts/admin/AdminLayout.vue'
import { useAppSettingsStore } from '../stores/appSettings'

const routes = [
  {
    path: '/',
    component: OuterApp,
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('@main/views/auth/UserLoginView.vue'),
        meta: { title: 'Login' }
      },
      {
        path: 'reset-password',
        name: 'reset-password',
        component: () => import('@main/views/auth/ResetPasswordView.vue'),
        meta: { title: 'Reset Password' }
      },
      {
        path: 'set-password',
        name: 'set-password',
        component: () => import('@main/views/auth/SetPasswordView.vue'),
        meta: { title: 'Set Password' }
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
        meta: { title: 'All contacts' }
      },
      {
        path: 'contacts/:id',
        name: 'contact-detail',
        component: () => import('@main/views/contact/ContactDetailView.vue'),
        meta: { title: 'Contacts' }
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
            meta: { title: 'Overview' }
          }
        ]
      },
      {
        path: '/inboxes/teams/:teamID',
        name: 'teams',
        props: true,
        component: InboxLayout,
        meta: { title: 'Team inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'team-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { title: 'Team inbox' },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'team-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { title: 'Team inbox', hidePageHeader: true }
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
        meta: { title: 'View inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'view-inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: { title: 'View inbox' },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'view-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { title: 'View inbox', hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: 'inboxes/search',
        name: 'search',
        component: () => import('@main/views/search/SearchView.vue'),
        meta: { title: 'Search', hidePageHeader: true }
      },
      {
        path: '/inboxes/:type(assigned|unassigned|all)?',
        name: 'inboxes',
        redirect: '/inboxes/assigned',
        component: InboxLayout,
        props: true,
        meta: { title: 'Inbox', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'inbox',
            component: () => import('@main/views/inbox/InboxView.vue'),
            meta: {
              title: 'Inbox',
              type: (route) => (route.params.type === 'assigned' ? 'My inbox' : route.params.type)
            },
            children: [
              {
                path: 'conversation/:uuid',
                name: 'inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: {
                  title: 'Inbox',
                  type: (route) =>
                    route.params.type === 'assigned' ? 'My inbox' : route.params.type,
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
        meta: { title: 'Account' },
        children: [
          {
            path: 'profile',
            name: 'profile',
            component: () => import('@main/views/account/profile/ProfileEditView.vue'),
            meta: { title: 'Edit Profile' }
          }
        ]
      },
      {
        path: '/admin',
        name: 'admin',
        component: AdminLayout,
        meta: { title: 'Admin' },
        children: [
          {
            path: 'custom-attributes',
            name: 'custom-attributes',
            component: () => import('@main/views/admin/custom-attributes/CustomAttributes.vue'),
            meta: { title: 'Custom attributes' }
          },
          {
            path: 'general',
            name: 'general',
            component: () => import('@main/views/admin/general/General.vue'),
            meta: { title: 'General' }
          },
          {
            path: 'business-hours',
            component: () => import('@main/views/admin/business-hours/BusinessHours.vue'),
            meta: { title: 'Business Hours' },
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
                meta: { title: 'New Business Hours' }
              },
              {
                path: ':id/edit',
                name: 'edit-business-hours',
                props: true,
                component: () =>
                  import('@main/views/admin/business-hours/CreateOrEditBusinessHours.vue'),
                meta: { title: 'Edit Business Hours' }
              }
            ]
          },
          {
            path: 'sla',
            component: () => import('@main/views/admin/sla/SLA.vue'),
            meta: { title: 'SLA' },
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
                meta: { title: 'New SLA' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-sla',
                component: () => import('@main/views/admin/sla/CreateEditSLA.vue'),
                meta: { title: 'Edit SLA' }
              }
            ]
          },
          {
            path: 'inboxes',
            component: () => import('@main/views/admin/inbox/InboxView.vue'),
            meta: { title: 'Inboxes' },
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
                meta: { title: 'New Inbox' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-inbox',
                component: () => import('@main/views/admin/inbox/EditInbox.vue'),
                meta: { title: 'Edit Inbox' }
              }
            ]
          },
          {
            path: 'notification',
            component: () => import('@main/features/admin/notification/NotificationSetting.vue'),
            meta: { title: 'Notification Settings' }
          },
          {
            path: 'teams',
            meta: { title: 'Teams' },
            children: [
              {
                path: 'agents',
                component: () => import('@main/views/admin/agents/Agents.vue'),
                meta: { title: 'Agents' },
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
                    meta: { title: 'Create agent' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    component: () => import('@main/views/admin/agents/EditAgent.vue'),
                    meta: { title: 'Edit agent' }
                  }
                ]
              },
              {
                path: 'teams',
                component: () => import('@main/views/admin/teams/Teams.vue'),
                meta: { title: 'Teams' },
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
                    meta: { title: 'Create Team' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-team',
                    component: () => import('@main/views/admin/teams/EditTeamForm.vue'),
                    meta: { title: 'Edit Team' }
                  }
                ]
              },
              {
                path: 'roles',
                component: () => import('@main/views/admin/roles/Roles.vue'),
                meta: { title: 'Roles' },
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
                    meta: { title: 'Create Role' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-role',
                    component: () => import('@main/views/admin/roles/EditRole.vue'),
                    meta: { title: 'Edit Role' }
                  }
                ]
              },
              {
                path: 'activity-log',
                name: 'activity-log',
                component: () => import('@main/views/admin/activity-log/ActivityLog.vue'),
                meta: { title: 'Activity Log' }
              }
            ]
          },
          {
            path: 'automations',
            component: () => import('@main/views/admin/automations/Automation.vue'),
            name: 'automations',
            meta: { title: 'Automations' },
            children: [
              {
                path: 'new',
                props: true,
                name: 'new-automation',
                component: () => import('@main/views/admin/automations/CreateOrEditRule.vue'),
                meta: { title: 'Create Automation' }
              },
              {
                path: ':id/edit',
                props: true,
                name: 'edit-automation',
                component: () => import('@main/views/admin/automations/CreateOrEditRule.vue'),
                meta: { title: 'Edit Automation' }
              }
            ]
          },
          {
            path: 'templates',
            component: () => import('@main/views/admin/templates/Templates.vue'),
            name: 'templates',
            meta: { title: 'Templates' },
            children: [
              {
                path: ':id/edit',
                name: 'edit-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { title: 'Edit Template' }
              },
              {
                path: 'new',
                name: 'new-template',
                props: true,
                component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'),
                meta: { title: 'New Template' }
              }
            ]
          },
          {
            path: 'sso',
            component: () => import('@main/views/admin/oidc/OIDC.vue'),
            name: 'sso',
            meta: { title: 'SSO' },
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
                meta: { title: 'Edit SSO' }
              },
              {
                path: 'new',
                name: 'new-sso',
                component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'),
                meta: { title: 'New SSO' }
              }
            ]
          },
          {
            path: 'webhooks',
            component: () => import('@main/views/admin/webhooks/Webhooks.vue'),
            name: 'webhooks',
            meta: { title: 'Webhooks' },
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
                meta: { title: 'Edit Webhook' }
              },
              {
                path: 'new',
                name: 'new-webhook',
                component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'),
                meta: { title: 'New Webhook' }
              }
            ]
          },
          {
            path: 'conversations',
            meta: { title: 'Conversations' },
            children: [
              {
                path: 'tags',
                component: () => import('@main/views/admin/tags/TagsView.vue'),
                meta: { title: 'Tags' }
              },
              {
                path: 'statuses',
                component: () => import('@main/views/admin/status/StatusView.vue'),
                meta: { title: 'Statuses' }
              },
              {
                path: 'macros',
                component: () => import('@main/views/admin/macros/Macros.vue'),
                meta: { title: 'Macros' },
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
                    meta: { title: 'Create Macro' }
                  },
                  {
                    path: ':id/edit',
                    props: true,
                    name: 'edit-macro',
                    component: () => import('@main/views/admin/macros/EditMacro.vue'),
                    meta: { title: 'Edit Macro' }
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
  // Make page title with the route name and site name
  const appSettingsStore = useAppSettingsStore()
  const siteName = appSettingsStore.settings?.['app.site_name'] || 'Libredesk'
  const pageTitle = to.meta?.title || ''
  document.title = `${pageTitle} - ${siteName}`
  next()
})

export default router
