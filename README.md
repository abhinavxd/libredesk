# Fernmail

Fernmail is a self-hosted, address-first shared email application derived from LibreDesk. It keeps email receiving and replies, threading, attachments, rich replies, drafts, search, statuses, snoozing, private notes and mentions.

There is no global “All mail” page and no saved Views. The sidebar lists the email addresses a person may access, with green badges showing unread messages that need attention. **Compose New** starts an email from any of those addresses.

People have one of three roles. **Admins** manage settings and people and can use every address. **Agents** read and send from the addresses they are given. **Contributors** read their addresses, but their replies and new emails wait in the **Review** queue until an Admin or Agent approves them; a blue badge shows how many are waiting. Roles and addresses are set per person in **Admin → Workspace → Users**.

Email runs over IMAP and SMTP, including Google/Microsoft OAuth. This is a conversation-based shared mailbox, not a full bidirectional IMAP folder client.

Unread mail has a green edge and contributes to the browser tab count. Each address has a “Mark all as read” action for your own read state. Optional browser notifications show the sender and a message preview while Fernmail is open.

The new binary, module, and container image are named Fernmail. Existing `LIBREDESK_` configuration variables, database identifiers, and container names remain compatible with earlier installations. Custom site names and favicons are preserved when upgrading.

See [the mail-only upgrade notes](docs/mail-only.md) before upgrading an existing database.

## Installation

### Docker

The latest image is available in GitHub Container Registry at [`ghcr.io/jakedolan443/fernmail:latest`](https://github.com/jakedolan443/fernmail/pkgs/container/fernmail).

```shell
# Download the compose file and sample config file in the current directory.
curl -LO https://github.com/jakedolan443/fernmail/raw/main/docker-compose.yml
curl -LO https://github.com/jakedolan443/fernmail/raw/main/config.sample.toml

# Copy the config.sample.toml to config.toml and edit it as needed.
cp config.sample.toml config.toml

# Run the services in the background.
docker compose up -d

# Setting System user password.
docker exec -it libredesk_app ./fernmail --set-system-user-password
```

Go to `http://localhost:9000` and login with username `System` and the password you set using the `--set-system-user-password` command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)

__________________

### Binary
- Download the [latest release](https://github.com/jakedolan443/fernmail/releases) and extract the `fernmail` binary.
- Edit config.toml as needed.
- `./fernmail --install` to setup the Postgres DB.
- Run `./fernmail --set-system-user-password` to set the password for the System user.
- Run `./fernmail` and visit `http://localhost:9000` and login with email `System` and the password you set using the --set-system-user-password command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)
__________________

## Developers

For local development and setup, refer to the [developer setup](https://docs.libredesk.io/contributing/developer-setup).

The backend is written in Go and the frontend is Vue.js 3 with Shadcn UI.

### Addresses and access

An **Address** is the email identity agents see, receive mail at and reply from. A physical mailbox address owns one IMAP/SMTP transport; an alias shares that transport safely. Configure aliases, display names and access in **Admin → Channel → Addresses**. A primary mailbox address is created automatically when its transport is connected.

Admins can use every address. Everyone else needs access to an address, given to them in **Admin → Workspace → Users** or through a team; there are no addresses open to everyone. An address with historical conversations cannot be renamed, moved or deleted; disable it instead so existing replies retain the correct From identity.

Access restrictions apply on the server to conversation reads and replies, lists, search, drafts, counts, attachments and live message notifications. Upgrade existing databases before starting this build (`--upgrade`); fresh installations include the address schema automatically.
