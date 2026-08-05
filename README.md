<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" alt="Zerodha Tech Badge" /></a>

<br>
<picture>
  <source
    media="(prefers-color-scheme: dark)"
    srcset="https://s3.ap-south-1.amazonaws.com/libredesk.io/libredesk_white.png?q=1">
  <source
    media="(prefers-color-scheme: light)"
    srcset="https://s3.ap-south-1.amazonaws.com/libredesk.io/libredesk_black.png">
  <img
    alt="LibreDesk"
    src="https://s3.ap-south-1.amazonaws.com/libredesk.io/libredesk_white.png"
    width="250">
</picture>

<br> Modern, open source, self-hosted omnichannel customer support desk. Live chat, email, and more in a single binary.

![image](https://libredesk.io/hero-dark-v2.png?q=3)


Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**live demo**](https://demo.libredesk.io/).

## Features

- **Omnichannel inbox**  
  Live chat and email in one inbox. Every conversation lands in the same place, whichever channel it came from.
- **Live chat widget**  
  Embed a real-time chat widget on your website. Replies go out from the same inbox your team already works in.
- **AI assistant**  
  Answer live chat conversations automatically with an AI assistant grounded in your knowledge base. Hands off to a human when it can't help.
- **Agent copilot**  
  Draft replies, summarize conversations, and look up answers from the knowledge base without leaving the inbox.
- **Automations**  
  Rules that run on conversation events. Tag, assign, and route conversations based on conditions you define.
- **Granular permissions**  
  Role-based access control. Create custom roles with per-action permissions for teams and individual agents.
- **CSAT & analytics**  
  Send CSAT surveys automatically after a conversation closes. Track response times, resolution rates, and agent activity.
- **Custom attributes**  
  Create custom attributes for contacts or conversations such as the subscription plan or the date of their first purchase.
- **Macros**  
  Save replies you send often. One macro can send the message, set tags, and assign the conversation to a team.
- **Organization**  
  Tags, custom statuses, and snoozing to keep the inbox in order. Search covers every conversation.
- **Auto assignment**  
  Assign incoming conversations automatically, based on agent capacity or on criteria you define.
- **SLA management**  
  Set and track response time targets. Get notified when conversations are at risk of breaching SLA commitments.
- **SSO logins**  
  Google, Microsoft, and any OIDC provider are supported out of the box.
- **API**  
  HTTP/JSON APIs and webhooks for custom integrations and workflows.
- **Activity logs**  
  Track all actions performed by agents and admins, for auditing and accountability.
- **Command bar**  
  Opens with a simple shortcut (CTRL+K) and lets you quickly perform actions on conversations.

And more — checkout [libredesk.io](https://libredesk.io) or try the [live demo](https://demo.libredesk.io/).


## Installation

### Docker

The latest image is available on DockerHub at [`libredesk/libredesk:latest`](https://hub.docker.com/r/libredesk/libredesk/tags?page=1&ordering=last_updated&name=latest)

```shell
# Download the compose file and sample config file in the current directory.
curl -LO https://github.com/abhinavxd/libredesk/raw/main/docker-compose.yml
curl -LO https://github.com/abhinavxd/libredesk/raw/main/config.sample.toml

# Copy the config.sample.toml to config.toml and edit it as needed.
cp config.sample.toml config.toml

# Run the services in the background.
docker compose up -d

# Setting System user password.
docker exec -it libredesk_app ./libredesk --set-system-user-password
```

Go to `http://localhost:9000` and login with username `System` and the password you set using the `--set-system-user-password` command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)

__________________

### Binary
- Download the [latest release](https://github.com/abhinavxd/libredesk/releases) and extract the libredesk binary.
- Edit config.toml as needed.
- `./libredesk --install` to setup the Postgres DB.
- Run `./libredesk --set-system-user-password` to set the password for the System user.
- Run `./libredesk` and visit `http://localhost:9000` and login with email `System` and the password you set using the --set-system-user-password command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)
__________________

## Developers

- If you are interested in contributing, **please read [CONTRIBUTING.md](./CONTRIBUTING.md) first**.
- For local development and setup, refer to the [developer setup](https://docs.libredesk.io/contributing/developer-setup).
- For planned features and project direction, see [ROADMAP.md](./ROADMAP.md).

The backend is written in Go and the frontend is Vue.js 3 with Shadcn UI.



## Translators
You can help translate libredesk into your language on [Crowdin](https://crowdin.com/project/libredesk).  
