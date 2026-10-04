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
    component: () => import('@main/App.vue'),
    children: [
      {
        path: '/addresses/:addressID?',
        component: () => import('@main/layouts/inbox/InboxLayout.vue'),
        meta: { titleKey: 'address.title', hidePageHeader: true },
        children: [
          {
            path: '',
            name: 'address-inbox',
            component: () => import('@main/views/address/AddressView.vue'),
            children: [
              {
                path: 'conversation/:uuid',
                name: 'address-inbox-conversation',
                component: () => import('@main/views/conversation/ConversationDetailView.vue'),
                props: true,
                meta: { titleKey: 'address.title', hidePageHeader: true }
              }
            ]
          }
        ]
      },
      {
        path: '/reviews/:uuid?',
        name: 'reviews',
        component: () => import('@main/views/review/ReviewsView.vue'),
        props: true,
        meta: { titleKey: 'review.title', hidePageHeader: true }
      },
      // Search is retired; old bookmarks land on the first accessible address.
      { path: '/search', redirect: '/addresses' },
      {
        path: '/conversation/:uuid',
        name: 'conversation-redirect',
        component: () => import('@main/views/conversation/ConversationRedirectView.vue'),
        props: true,
        meta: { hidePageHeader: true }
      },
      // A direct deep link to an old mailbox never restores the retired global
      // list. It lands on the first address the user can read.
      { path: '/inboxes/:pathMatch(.*)*', redirect: '/addresses' },

      {
        path: '/admin',
        name: 'admin',
        component: () => import('@main/layouts/admin/AdminLayout.vue'),
        meta: { titleKey: 'globals.terms.admin' },
        children: [
          {
            path: 'general',
            name: 'general',
            component: () => import('@main/views/admin/general/General.vue'),
            meta: { titleKey: 'globals.terms.general' }
          },
          {
            path: 'key-distribution',
            name: 'key-distribution',
            component: () => import('@main/views/admin/keys/KeyDistribution.vue'),
            meta: { titleKey: 'keyDistribution.title' }
          },
          {
            path: 'users',
            name: 'users',
            component: () => import('@main/views/admin/users/UsersSettings.vue'),
            meta: { titleKey: 'users.title' }
          },
          {
            path: 'resources',
            name: 'system-resources',
            component: () => import('@main/views/admin/resources/ResourceMonitor.vue'),
            meta: { titleKey: 'admin.systemResources.title' }
          },
          {
            path: 'addresses',
            component: () => import('@main/views/admin/address/Addresses.vue'),
            meta: { titleKey: 'address.title', titleCount: 2 },
            children: [
              {
                path: '',
                name: 'address-list',
                component: () => import('@main/views/admin/address/AddressList.vue')
              },
              {
                path: 'new',
                name: 'new-address',
                component: () => import('@main/views/admin/address/AddressForm.vue'),
                meta: { titleKey: 'address.new' }
              },
              {
                path: ':id/edit',
                name: 'edit-address',
                component: () => import('@main/views/admin/address/AddressForm.vue'),
                props: true,
                meta: { titleKey: 'address.edit' }
              }
            ]
          },
          // Transport setup is deliberately not in the normal navigation. It
          // remains available for first-time IMAP/SMTP wiring, while aliases
          // and agent access are configured exclusively as Addresses.
          {
            path: 'inboxes',
            component: () => import('@main/views/admin/inbox/InboxView.vue'),
            meta: { titleKey: 'globals.terms.inbox', titleCount: 2 },
            children: [
              { path: '', name: 'inbox-list', component: () => import('@main/views/admin/inbox/InboxList.vue') },
              { path: 'new', name: 'new-inbox', component: () => import('@main/views/admin/inbox/NewInbox.vue'), meta: { titleKey: 'inbox.newInbox' } },
              { path: ':id/edit', props: true, name: 'edit-inbox', component: () => import('@main/views/admin/inbox/EditInbox.vue'), meta: { titleKey: 'inbox.edit' } }
            ]
          },
          {
            path: 'templates',
            component: () => import('@main/views/admin/templates/Templates.vue'),
            meta: { titleKey: 'globals.terms.template', titleCount: 2 },
            children: [
              { path: '', name: 'template-list', component: () => import('@main/views/admin/templates/TemplateList.vue') },
              { path: ':id/edit', name: 'edit-template', props: true, component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'), meta: { titleKey: 'template.edit' } },
              { path: 'new', name: 'new-template', props: true, component: () => import('@main/views/admin/templates/CreateEditTemplate.vue'), meta: { titleKey: 'template.new' } }
            ]
          },
          {
            path: 'sso',
            component: () => import('@main/views/admin/oidc/OIDC.vue'),
            name: 'sso',
            meta: { titleKey: 'globals.terms.sso' },
            children: [
              { path: '', name: 'sso-list', component: () => import('@main/views/admin/oidc/OIDCList.vue') },
              { path: ':id/edit', props: true, name: 'edit-sso', component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'), meta: { titleKey: 'oidc.edit' } },
              { path: 'new', name: 'new-sso', component: () => import('@main/views/admin/oidc/CreateEditOIDC.vue'), meta: { titleKey: 'oidc.new' } }
            ]
          },
          {
            path: 'webhooks',
            component: () => import('@main/views/admin/webhooks/Webhooks.vue'),
            name: 'webhooks',
            meta: { titleKey: 'globals.terms.webhook', titleCount: 2 },
            children: [
              { path: '', name: 'webhook-list', component: () => import('@main/views/admin/webhooks/WebhookList.vue') },
              { path: ':id/edit', props: true, name: 'edit-webhook', component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'), meta: { titleKey: 'webhook.edit' } },
              { path: 'new', name: 'new-webhook', component: () => import('@main/views/admin/webhooks/CreateEditWebhook.vue'), meta: { titleKey: 'webhook.new' } }
            ]
          }
        ]
      }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/addresses' }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes
})

router.beforeEach((to, from, next) => {
  if (to.fullPath !== from.fullPath) abortRouteScope()
  const appSettingsStore = useAppSettingsStore()
  const siteName = appSettingsStore.siteTitle
  const i18n = getI18n()
  const titleKey = to.meta?.titleKey
  const pageTitle = titleKey && i18n ? i18n.global.t(titleKey, to.meta?.titleCount || 1) : ''
  document.title = `${pageTitle} - ${siteName}`
  next()
})

export default router
