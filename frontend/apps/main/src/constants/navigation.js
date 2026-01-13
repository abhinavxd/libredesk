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
        permission: 'business_hours:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.slaPolicy',
        href: '/admin/sla',
        permission: 'sla:manage',
        isTitleKeyPlural: true
      }
    ]
  },
  {
    titleKey: 'globals.terms.conversation',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.tag',
        href: '/admin/conversations/tags',
        permission: 'tags:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.macro',
        href: '/admin/conversations/macros',
        permission: 'macros:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.sharedView',
        href: '/admin/conversations/shared-views',
        permission: 'shared_views:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.status',
        href: '/admin/conversations/statuses',
        permission: 'status:manage',
        isTitleKeyPlural: true
      }
    ]
  },
  {
    titleKey: 'globals.terms.inbox',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.inbox',
        href: '/admin/inboxes',
        permission: 'inboxes:manage',
        isTitleKeyPlural: true
      }
    ]
  },
  {
    titleKey: 'globals.terms.teammate',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.agent',
        href: '/admin/teams/agents',
        permission: 'users:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.team',
        href: '/admin/teams/teams',
        permission: 'teams:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.role',
        href: '/admin/teams/roles',
        permission: 'roles:manage',
        isTitleKeyPlural: true
      },
      {
        titleKey: 'globals.terms.activityLog',
        href: '/admin/teams/activity-log',
        permission: 'activity_logs:manage',
        isTitleKeyPlural: true
      }
    ]
  },
  {
    titleKey: 'globals.terms.automation',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.automation',
        href: '/admin/automations',
        permission: 'automations:manage',
        isTitleKeyPlural: true
      }
    ]
  },
  {
    titleKey: 'globals.terms.customAttribute',
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.customAttribute',
        href: '/admin/custom-attributes',
        permission: 'custom_attributes:manage',
        isTitleKeyPlural: true
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
    isTitleKeyPlural: true,
    children: [
      {
        titleKey: 'globals.terms.template',
        href: '/admin/templates',
        permission: 'templates:manage',
        isTitleKeyPlural: true
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
        permission: 'webhooks:manage',
        isTitleKeyPlural: true
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
    href: '/contacts',
    isTitleKeyPlural: true
  }
]
