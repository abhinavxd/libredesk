<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" alt="Zerodha Tech Badge" /></a>


# Libredesk

Open source, self-hosted customer support desk. Single binary app.

![image](docs/docs/images/hero.png)


Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**Live demo**](https://demo.libredesk.io/).

> **CAUTION:** This project is currently in **alpha**. Features and APIs may change and are not yet fully tested.

## Features

- **Multi Shared Inbox**  
  Libredesk supports multiple shared inboxes, letting you manage conversations across teams effortlessly.
- **Granular Permissions**  
  Create custom roles with granular permissions for teams and individual agents.
- **Smart Automation**  
  Eliminate repetitive tasks with powerful automation rules. Auto-tag, assign, and route conversations based on custom conditions.
- **CSAT Surveys**  
  Measure customer satisfaction with automated surveys.
- **Macros**  
  Save frequently sent messages as templates. With one click, send saved responses, set tags, and more.
- **Smart Organization**  
  Keep conversations organized with tags, custom statuses for conversations, and snoozing. Find any conversation instantly from the search bar.
- **Auto Assignment**  
  Distribute workload with auto assignment rules. Auto-assign conversations based on agent capacity or custom criteria.
- **SLA Management**  
  Set and track response time targets. Get notified when conversations are at risk of breaching SLA commitments.
- **Custom attributes**  
  Create custom attributes for contacts or conversations such as the subscription plan or the date of their first purchase. 
- **AI-Assist**  
  Instantly rewrite responses with AI to make them more friendly, professional, or polished.
- **Activity logs**  
  Track all actions performed by agents and admins—updates and key events across the system—for auditing and accountability.
- **Webhooks**  
  Integrate with external systems using real-time HTTP notifications for conversation and message events.
- **Command Bar**  
  Opens with a simple shortcut (CTRL+K) and lets you quickly perform actions on conversations.

And more checkout - [libredesk.io](https://libredesk.io)


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

See [installation docs](https://libredesk.io/docs/installation/)

__________________

### Binary
- Download the [latest release](https://github.com/abhinavxd/libredesk/releases) and extract the libredesk binary.
- Copy config.sample.toml to config.toml and edit as needed.
- `./libredesk --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./libredesk --set-system-user-password` to set the password for the System user.
- Run `./libredesk` and visit `http://localhost:9000` and login with username `System` and the password you set using the --set-system-user-password command.

See [installation docs](https://libredesk.io/docs/installation)
__________________


## Developers
If you are interested in contributing, refer to the [developer setup](https://libredesk.io/docs/developer-setup/). The backend is written in Go and the frontend is Vue js 3 with Shadcn for UI components.

## Development Status

Libredesk is under active development.  
Track roadmap and progress on the GitHub Project Board:   [https://github.com/users/abhinavxd/projects/1](https://github.com/users/abhinavxd/projects/1)


## Translators
You can help translate Libredesk into your language on [Crowdin](https://crowdin.com/project/libredesk).  
