export const adminNavItems = [
  {
    titleKey: 'globals.terms.workspace',
    children: [
      {
        titleKey: 'globals.terms.general',
        href: '/admin/general',
        permission: 'general_settings:manage',
        icon: 'Settings'
      },
      {
        titleKey: 'users.title',
        href: '/admin/users',
        permission: 'users:manage',
        icon: 'UsersRound'
      },
      {
        titleKey: 'admin.systemResources.title',
        href: '/admin/resources',
        permission: 'general_settings:manage',
        icon: 'Gauge'
      }
    ]
  },

  {
    titleKey: 'globals.terms.channel',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'address.title',
        href: '/admin/addresses',
        createRouteName: 'new-address',
        permission: 'inboxes:manage',
        isTitleKeyPlural: true,
        icon: 'Mail'
      }
    ]
  },
  {
    titleKey: 'globals.terms.conversation',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.template',
        href: '/admin/templates',
        createRouteName: 'new-template',
        permission: 'templates:manage',
        isTitleKeyPlural: true,
        icon: 'FileText'
      }
    ]
  },

  {
    titleKey: 'globals.terms.security',
    children: [
      {
        titleKey: 'globals.terms.sso',
        href: '/admin/sso',
        createRouteName: 'new-sso',
        permission: 'oidc:manage',
        icon: 'KeyRound'
      }
    ]
  },
  {
    titleKey: 'globals.terms.integration',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.webhook',
        href: '/admin/webhooks',
        createRouteName: 'new-webhook',
        permission: 'webhooks:manage',
        isTitleKeyPlural: true,
        icon: 'Webhook'
      }
    ]
  }
]
