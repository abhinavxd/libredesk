# Mailbox frontend

Vue 3 application in `apps/main`, with shared components in `shared-ui`.
Run these commands from `frontend`:

```sh
pnpm install
pnpm dev:main
```

The development server proxies API and WebSocket requests to the backend on port 9000.
Override these with `LD_API_TARGET`, `LD_WS_TARGET`, and `LD_DEV_PORT` as needed.

```sh
pnpm build:main  # Production assets in dist/main
pnpm test:run    # Vitest unit tests
pnpm test:e2e:ci # Cypress against a running backend on localhost:9000
```

Cypress integration tests need a disposable installation, a System login, and MailHog
for SMTP tests. The CI workflow in `.github/workflows/frontend-ci.yml` supplies these.
Set `CYPRESS_SYSTEM_PASSWORD` and `CYPRESS_MAILHOG_URL` for another test installation.

Use `pnpm exec prettier --write <paths>` to format changed source files.

### Mock mail preview

With `pnpm dev:main` running, open `http://localhost:8000/mail-preview.html`.
This development-only entry renders the real inbox rows and message components with
in-memory emails. Open a row to clear its green unread strip, right-click to mark it
unread, and use **Receive mock mail** to exercise the live-message handler. The bell
enables native browser notifications; the labeled preview card always demonstrates
the sender and message content. **Reset demo** restores Albert's example email.
The preview never sends mail or connects to a backend and is excluded from the production entry.

Browser notifications are opt-in per account and browser. Use the bell beside the
account name to enable or disable them. They require HTTPS (localhost also works),
a supported desktop browser, and an open Libredesk tab. They use live mailbox events,
so they do not run after all Fernmail tabs are closed. Outgoing mail, internal notes,
and activity updates do not generate notifications.

The green favicon and the Fernmail title are the defaults. A site logo and site title
set in Admin → General replace them in the top bar, the tab title and the favicon.
The tab title prefixes the total unread message count, including mail outside the
current address, and removes the prefix when no unread messages remain.
