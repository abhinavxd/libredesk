# Mail-only fork

The UI, HTTP routes and background workers for contacts/CRM, ticket assignment and priorities, automations, macros, SLA/business-hours tracking, CSAT, reports, activity logs, custom attributes, context links, public help centers, chat widgets and AI have been removed.

Retained: email inbox connections and OAuth, reply with CC/BCC, attachments and image controls, drafts, threading, transcripts, search, read state, custom statuses and snoozing, internal notes and mentions, account authentication, SSO, mailbox configuration, APIs and webhooks for the retained features. Since v3.7.0, Compose New starts a new conversation from an address; see below.

## Existing installations

Back up the database before running the normal `--upgrade` command. Migration v3.0.0 removes retired feature tables and queued support notifications, clears assignment/priority/SLA state and macro draft metadata, disables non-email inboxes, and removes obsolete notification templates. It preserves email conversations, messages, attachments, drafts and message sender identities. Tags are subsequently removed by v3.2.0.

The legacy `users` sender records and some compatibility columns/types remain to preserve message foreign keys and historical migrations. They no longer provide a contact directory, editable contact profiles, contact notes, blocking, export or CRM enrichment. New sender records store only an email address and display name, and are reused without updating a profile. Conversation responses expose this minimal identity as `correspondent`. The new-message endpoint takes `email` instead of `contact_email` and no longer accepts assignment or custom-field inputs as features.

Historical chat rows remain in storage rather than deleting message history during an upgrade; chat inboxes cannot run and are excluded from mailbox access.

Account authentication and internal permission checks remain infrastructure. The custom-role/team management UI has been removed. Existing accounts with mailbox read access receive shared mailbox access during the upgrade because assignment-based mail visibility no longer has a user-facing workflow.

## Development

Build the frontend with `cd frontend && pnpm build:main`, then build the Go backend. There is no widget build. Run `go test ./...` and `cd frontend && pnpm test:run`. Set `LIBREDESK_TEST_DB_DSN` to a disposable PostgreSQL instance to run the database-backed tests as well. `make build-backend` builds the Go package; `make build` also builds and embeds the frontend.

Migration v3.1.0 removes the notification inbox, preferences, email alert queue, browser push subscriptions, alert templates and notification permissions. Account welcome/password-reset email transport is separate and uses the account_email configuration namespace; existing SMTP settings are migrated automatically. Mailbox unread counts, live updates, mentions, and action/error feedback remain available.

Migration v3.2.0 retires tags, their reply prompts, activity records, permissions and webhook subscriptions.

Migration v3.5.0 introduces first-class **Addresses**. A physical mailbox owns one IMAP/SMTP transport; aliases share that transport. Existing transport addresses and configured aliases are promoted into the address model, and existing conversations are linked to the address that received them. Legacy email-alias Views are converted to direct user/team address grants before the Views table, its type, UI and API are permanently removed. Legacy View grants are promoted only when the transport was unrestricted. Restricted transports retain their existing allowed users; a View cannot widen that access. Administrators retain recovery access. Back up before upgrading.


## Mail reliability and draft ownership (v3.6.0)

Stop all application instances, back up the database, run `./fernmail --config config.toml --upgrade`, then start the updated instances. Fresh installations already include this schema. Views remain removed: no saved-View routes, tables, or workspace controls are restored.

IMAP progress is now persisted per transport, server/account/folder, and UIDVALIDITY. The first connection after upgrading scans the selected folders from their beginning, including mail older than the former rolling lookback window. Existing address-local Message-IDs are deduplicated. A mailbox UIDVALIDITY reset starts a fresh scan. Failed messages are retained for retry, and a full queue pauses scanning without skipping deferred mail. `message.incoming_queue_size` bounds staged messages, including their attachment payloads; `message.max_incoming_message_size` still limits an individual raw message. The old inbox lookback setting is no longer used.

Outgoing messages are claimed in PostgreSQL so multiple application processes cannot send the same queued message concurrently. A crash or SMTP error after delivery begins leaves a visible uncertain-delivery state. Verify whether the recipient received the message before confirming a retry: SMTP cannot guarantee exactly-once delivery across a broken connection. Successful sends whose completion cannot be recorded are held for reconciliation instead of automatically resent.

Browser uploads now belong to their uploader before a reply is sent. Attachments referenced by saved drafts are protected from collection, and draft expiry follows the last edit. Old unlinked uploads have no trustworthy uploader history, so attachments in pre-upgrade drafts must be reuploaded before sending; draft text and attachments already linked to historical messages are preserved. The migration retains old draft file references for their normal retention period without inventing ownership.

The corrected v3.5.0 migration preserves transport restrictions when retiring Views. If an installation already ran the previous v3.5.0 migration, review its address grants in Admin → Channel → Addresses. The old View records are gone, so the upgrade cannot distinguish a wrongly promoted grant from a later intentional administrator grant.

Each address's menu offers **Mark all as read**. This marks the current user's visible conversations across all pages and statuses. It does not mark teammates' mail read, change conversation statuses, or hide messages arriving afterward. Other open tabs for the same user refresh their read state.

### Regression tests

Use a disposable PostgreSQL database with a user allowed to create databases. Tests create isolated databases and remove them afterward. An explicitly configured but unreachable database is a test failure rather than silently skipped coverage.

```sh
export LIBREDESK_TEST_DB_DSN='postgres://testuser:testpass@localhost:5432/fernmail_test?sslmode=disable'
go test -race ./...
go vet ./...
cd frontend
pnpm test:run
pnpm build:main
# Against a disposable installed app with a System user:
CYPRESS_BASE_URL=http://localhost:9000 CYPRESS_SYSTEM_PASSWORD='your-test-password' pnpm test:e2e:ci
```

The regressions exercise real PostgreSQL permission boundaries, migrations, concurrent ingestion/delivery claims, draft attachment ownership and retention, mailbox recovery with an in-process IMAP server, and browser state during send/navigation, reconnect, and failed draft saves.

## Contributors, review queue and Compose New (v3.7.0)

Back up the database and run `./fernmail --config config.toml --upgrade` before starting the new version.

- **Contributor role.** Contributors read the addresses they are given. Their replies and new emails wait in the Review queue until an Admin or Agent who can use the address approves them. A denied reply returns to its author's reply box as a draft, with the reviewer's reason. Admins and Agents gain the `reviews:manage` and `conversations:create` permissions.
- **Open addresses are retired.** Admins can use every address; everyone else needs a direct or team grant. To keep today's access, the upgrade gives every enabled non-admin agent an explicit grant to each address that was open to everyone. Review these in **Admin → Workspace → Users** afterwards.
- **Users.** One screen lists everyone, with their role, addresses and whether they can sign in. It also adds and deletes people. The last enabled Admin cannot be demoted, disabled or deleted, and nobody can change their own role. Per-person access has moved here from the address form, which keeps team grants.
- **Compose New** sends a new email from a chosen address. Admins and Agents send directly; a Contributor's email goes to review first.

See [contributor-review.md](contributor-review.md) for the full design.
