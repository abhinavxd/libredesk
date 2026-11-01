
import { defineMessage as msg } from '@lingui/core/macro'
export const reportsNavItems = [
  {
    titleMessage: msg({ message: 'Overview' }),
    href: '/reports/overview',
    permission: 'reports:manage',
    icon: 'BarChart3'
  }
]

export const adminNavItems = [
  {
    titleMessage: msg({ message: 'Workspace' }),
    children: [
      {
        titleMessage: msg({ message: 'General' }),
        href: '/admin/general',
        permission: 'general_settings:manage',
        icon: 'Settings'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Business hour} other {Business hours}}' }),
        href: '/admin/business-hours',
        createRouteName: 'new-business-hours',
        permission: 'business_hours:manage',
        isTitlePlural: true,
        icon: 'Clock'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {SLA policy} other {SLA policies}}' }),
        href: '/admin/sla',
        createRouteName: 'new-sla',
        permission: 'sla:manage',
        isTitlePlural: true,
        icon: 'Timer'
      }
    ]
  },
  {
    titleMessage: msg({ message: '{count, plural, one {Help center} other {Help centers}}' }),
    badge: 'BETA',
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Help center} other {Help centers}}' }),
        href: '/admin/help-center',
        permission: 'help_center:manage',
        icon: 'BookOpen'
      }
    ]
  },
  {
    titleMessage: msg({ message: 'AI' }),
    badge: 'BETA',
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Provider} other {Providers}}' }),
        href: '/admin/ai/providers',
        permission: 'ai:manage',
        isTitlePlural: true,
        icon: 'Sparkles'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Snippet} other {Snippets}}' }),
        href: '/admin/ai/snippets',
        permission: 'ai:manage',
        isTitlePlural: true,
        icon: 'NotebookText'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Editor prompt} other {Editor prompts}}' }),
        href: '/admin/ai/editor-prompts',
        permission: 'ai:manage',
        isTitlePlural: true,
        icon: 'WandSparkles'
      },
      {
        titleMessage: msg({ message: 'Suggestions' }),
        href: '/admin/ai/suggestions',
        permission: 'ai:manage',
        isTitlePlural: true,
        icon: 'Lightbulb'
      },
      {
        titleMessage: msg({ message: 'Tools' }),
        href: '/admin/ai/tools',
        createRouteName: 'new-ai-tool',
        permission: 'ai:manage',
        isTitlePlural: true,
        icon: 'Wrench'
      },
      {
        titleMessage: msg({ message: 'Assistants' }),
        href: '/admin/ai/assistants',
        createRouteName: 'new-ai-assistant',
        permission: 'ai:manage',
        isTitlePlural: true,
        icon: 'Bot'
      }
    ]
  },
  {
    titleMessage: msg({ message: 'Channel' }),
    isTitlePlural: true,
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Inbox} other {Inboxes}}' }),
        href: '/admin/inboxes',
        createRouteName: 'new-inbox',
        permission: 'inboxes:manage',
        isTitlePlural: true,
        icon: 'Inbox'
      },
      {
        titleMessage: msg({ message: 'WhatsApp templates' }),
        href: '/admin/whatsapp/templates',
        permission: 'inboxes:manage',
        icon: 'WhatsApp'
      }
    ]
  },
  {
    titleMessage: msg({ message: '{count, plural, one {Conversation} other {Conversations}}' }),
    isTitlePlural: true,
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Status} other {Statuses}}' }),
        href: '/admin/conversations/statuses',
        permission: 'status:manage',
        isTitlePlural: true,
        icon: 'CircleDot'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Tag} other {Tags}}' }),
        href: '/admin/conversations/tags',
        permission: 'tags:manage',
        isTitlePlural: true,
        icon: 'Tag'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Custom attribute} other {Custom attributes}}' }),
        href: '/admin/custom-attributes',
        permission: 'custom_attributes:manage',
        isTitlePlural: true,
        icon: 'SlidersHorizontal'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Shared view} other {Shared views}}' }),
        href: '/admin/conversations/shared-views',
        createRouteName: 'new-shared-view',
        permission: 'shared_views:manage',
        isTitlePlural: true,
        icon: 'Eye'
      }
    ]
  },
  {
    titleMessage: msg({ message: 'Productivity' }),
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Macro} other {Macros}}' }),
        href: '/admin/conversations/macros',
        createRouteName: 'new-macro',
        permission: 'macros:manage',
        isTitlePlural: true,
        icon: 'Zap'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Automation} other {Automations}}' }),
        href: '/admin/automations',
        createRouteName: 'new-automation',
        permission: 'automations:manage',
        isTitlePlural: true,
        icon: 'Workflow'
      }
    ]
  },
  {
    titleMessage: msg({ message: '{count, plural, one {Teammate} other {Teammates}}' }),
    isTitlePlural: true,
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Agent} other {Agents}}' }),
        href: '/admin/teams/agents',
        createRouteName: 'new-agent',
        permission: 'users:manage',
        isTitlePlural: true,
        icon: 'UserRound'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Team} other {Teams}}' }),
        href: '/admin/teams/teams',
        createRouteName: 'new-team',
        permission: 'teams:manage',
        isTitlePlural: true,
        icon: 'UsersRound'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Role} other {Roles}}' }),
        href: '/admin/teams/roles',
        createRouteName: 'new-role',
        permission: 'roles:manage',
        isTitlePlural: true,
        icon: 'Shield'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Activity log} other {Activity logs}}' }),
        href: '/admin/teams/activity-log',
        permission: 'activity_logs:manage',
        isTitlePlural: true,
        icon: 'ScrollText'
      }
    ]
  },
  {
    titleMessage: msg({ message: '{count, plural, one {Notification} other {Notifications}}' }),
    isTitlePlural: true,
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Email} other {Emails}}', context: 'Communication channel' }),
        href: '/admin/notification',
        permission: 'notification_settings:manage',
        icon: 'Mail'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Template} other {Templates}}' }),
        href: '/admin/templates',
        createRouteName: 'new-template',
        permission: 'templates:manage',
        isTitlePlural: true,
        icon: 'FileText'
      }
    ]
  },
  {
    titleMessage: msg({ message: 'Security' }),
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {SSO} other {SSOs}}' }),
        href: '/admin/sso',
        createRouteName: 'new-sso',
        permission: 'oidc:manage',
        icon: 'KeyRound'
      }
    ]
  },
  {
    titleMessage: msg({ message: '{count, plural, one {Integration} other {Integrations}}' }),
    isTitlePlural: true,
    children: [
      {
        titleMessage: msg({ message: '{count, plural, one {Webhook} other {Webhooks}}' }),
        href: '/admin/webhooks',
        createRouteName: 'new-webhook',
        permission: 'webhooks:manage',
        isTitlePlural: true,
        icon: 'Webhook'
      },
      {
        titleMessage: msg({ message: '{count, plural, one {Context link} other {Context links}}' }),
        href: '/admin/context-links',
        createRouteName: 'new-context-link',
        permission: 'context_links:manage',
        isTitlePlural: true,
        icon: 'Link'
      }
    ]
  }
]

export const accountNavItems = [
  {
    titleMessage: msg({ message: '{count, plural, one {Profile} other {Profiles}}' }),
    href: '/account/profile',
    icon: 'CircleUser'
  },
  {
    titleMessage: msg({ message: '{count, plural, one {Notification} other {Notifications}}' }),
    href: '/account/notifications',
    isTitlePlural: true,
    icon: 'Bell'
  }
]

export const contactNavItems = [
  {
    titleMessage: msg({ message: '{count, plural, one {Contact} other {Contacts}}' }),
    allLabelMessage: msg({ message: 'All contacts' }),
    href: '/contacts',
    isTitlePlural: true,
    icon: 'Contact'
  }
]
