<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" alt="Zerodha Tech Badge" /></a>

<br>
<picture>
  <source
    media="(prefers-color-scheme: dark)"
    srcset="https://s3.ap-south-1.amazonaws.com/libredesk.io/libredesk_white.png?v=2">
  <source
    media="(prefers-color-scheme: light)"
    srcset="https://s3.ap-south-1.amazonaws.com/libredesk.io/libredesk_black.png?v=3">
  <img
    alt="LibreDesk"
    src="https://s3.ap-south-1.amazonaws.com/libredesk.io/libredesk_white.png?v=4"
    width="250">
</picture>

<br> Open source, self-hosted customer support software for email, live chat, and WhatsApp. Distributed as a single binary.

![image](https://libredesk.io/hero-dark.png?v=1)


Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**live demo**](https://demo.libredesk.io/).

## Features

### Inbox and channels

- **Shared inbox:** Handle email, live chat, and WhatsApp conversations in one place.
- **Live chat:** Add a real-time chat widget to your website.
- **WhatsApp:** Connect a number through the Meta Cloud API.
- **Inbox organization:** Use teams, tags, custom statuses, custom attributes, snoozing, and search.

### Help center and AI

- **Help center:** Publish a searchable, multilingual knowledge base.
- **AI assistant:** Answer live chat questions using your knowledge base and hand conversations to an agent when needed.
- **Agent copilot:** Draft replies, summarize conversations, and find answers without leaving the inbox.

### Workflow and reporting

- **Automations:** Tag, assign, and route conversations using rules you define.
- **Macros:** Send saved replies and apply conversation actions in one step.
- **Auto assignment:** Distribute incoming conversations by agent capacity or your own criteria.
- **SLA management:** Set response targets and get notified about conversations at risk of breaching them.
- **CSAT and analytics:** Collect customer ratings and track response times, resolution rates, and agent activity.

### Administration and integrations

- **Permissions and SSO:** Create roles with per-action permissions and sign in through Google, Microsoft, or another OIDC provider.
- **Activity logs:** Review actions performed by agents and admins.
- **API and webhooks:** Connect LibreDesk to other systems and workflows.

See [libredesk.io](https://libredesk.io) for the full feature set, or try the [live demo](https://demo.libredesk.io/).


## Installation

### Railway (1-click deploy)

The fastest way to get a libredesk instance running. Railway provisions the app, Postgres, and Redis for you.

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/libredesk?referralCode=4gdm5b&utm_medium=integration&utm_source=template&utm_campaign=generic)

__________________

### Docker

Download `setup.sh` from the [latest release](https://github.com/abhinavxd/libredesk/releases/latest/download/setup.sh) into an empty directory, then run:

```sh
sh setup.sh local
```

Open http://localhost:9000/setup and create your admin account using the temporary code printed in the terminal.

For a server with a domain:

```sh
sh setup.sh production https://support.example.com
```

See [installation instructions](INSTALL.md) for HTTPS configuration, dependency checks, and upgrades.

### Binary

Extract the [release archive](https://github.com/abhinavxd/libredesk/releases), configure PostgreSQL and Redis in `config.toml`, then run:

```sh
mkdir -p uploads
./libredesk --install
./libredesk --upgrade --yes
./libredesk --check
./libredesk
```

Open `/setup` and enter the temporary code printed on startup. See [installation instructions](INSTALL.md) for configuration details.

## Developers

- If you are interested in contributing, **please read [CONTRIBUTING.md](./CONTRIBUTING.md) first**.
- For local development and setup, refer to the [developer setup](https://docs.libredesk.io/contributing/developer-setup).
- For planned features and project direction, see [ROADMAP.md](./ROADMAP.md).

The backend is written in Go and the frontend is Vue.js 3 with Shadcn UI.



## Translators
You can help translate libredesk into your language on [Crowdin](https://crowdin.com/project/libredesk).  
