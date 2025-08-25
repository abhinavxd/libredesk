export const reportsNavItems = [
  {
    titleKey: 'globals.terms.overview',
    href: '/reports/overview',
    permission: 'reports:manage'
  }
]

export const adminNavItems = [
  {
    titleKey: 'globals.terms.workspace',
    children: [
      {
        titleKey: 'globals.terms.general',
        href: '/admin/general',
        permission: 'general_settings:manage'
      },
      {
        titleKey: 'globals.terms.businessHour',
        href: '/admin/business-hours',
        permission: 'business_hours:manage'
      },
      {
        titleKey: 'globals.terms.slaPolicy',
        href: '/admin/sla',
        permission: 'sla:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.conversation',
    children: [
      {
        titleKey: 'globals.terms.tag',
        href: '/admin/conversations/tags',
        permission: 'tags:manage'
      },
      {
        titleKey: 'globals.terms.macro',
        href: '/admin/conversations/macros',
        permission: 'macros:manage'
      },
      {
        titleKey: 'globals.terms.status',
        href: '/admin/conversations/statuses',
        permission: 'status:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.inbox',
    children: [
      {
        titleKey: 'globals.terms.inbox',
        href: '/admin/inboxes',
        permission: 'inboxes:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.teammate',
    children: [
      {
        titleKey: 'globals.terms.agent',
        href: '/admin/teams/agents',
        permission: 'users:manage'
      },
      {
        titleKey: 'globals.terms.team',
        href: '/admin/teams/teams',
        permission: 'teams:manage'
      },
      {
        titleKey: 'globals.terms.role',
        href: '/admin/teams/roles',
        permission: 'roles:manage'
      },
      {
        titleKey: 'globals.terms.activityLog',
        href: '/admin/teams/activity-log',
        permission: 'activity_logs:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.automation',
    children: [
      {
        titleKey: 'globals.terms.automation',
        href: '/admin/automations',
        permission: 'automations:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.customAttribute',
    children: [
      {
        titleKey: 'globals.terms.customAttribute',
        href: '/admin/custom-attributes',
        permission: 'custom_attributes:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.notification',
    children: [
      {
        titleKey: 'globals.terms.email',
        href: '/admin/notification',
        permission: 'notification_settings:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.template',
    children: [
      {
        titleKey: 'globals.terms.template',
        href: '/admin/templates',
        permission: 'templates:manage'
      }
    ]
  },
  {
    titleKey: 'globals.terms.security',
    children: [
      {
        titleKey: 'globals.terms.sso',
        href: '/admin/sso',
        permission: 'oidc:manage'
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
        permission: 'webhooks:manage'
      }
    ]
  }
]

export const accountNavItems = [
  {
    titleKey: 'globals.terms.profile',
    href: '/account/profile'
  }
]

export const contactNavItems = [
  {
    titleKey: 'globals.terms.contact',
    href: '/contacts'
  }
]
