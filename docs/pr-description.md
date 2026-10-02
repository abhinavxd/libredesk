# PR title

fix: preferred frontend changes

## Summary

This personal-fork frontend pass addresses the five requested inbox preferences:

- Keep conversation avatars visible on hover; only replace the avatar with a checked selection box after the conversation is selected.
- Open conversations at the top, or at the latest unread reply when unread messages are present, instead of forcing the view to the bottom.
- Put From/To/Cc/Bcc metadata on one compact line outside the email bubble and reduce excess email padding.
- Make the reply/private-note composer vertically resizable with a persisted, bounded layout and a smaller default footprint.
- Replace the bright legacy green with a darker, more trustworthy teal-green Fernmail palette across light and dark themes.

## Screenshots

The red rectangles call out the relevant visual changes:

### Inbox list and message layout

![Inbox frontend preferences](https://raw.githubusercontent.com/jakedolan443/fernmail/fix/frontend_preferences/docs/screenshots/frontend-preferences-inbox.png)

### Conversation composer and theme

![Conversation frontend preferences](https://raw.githubusercontent.com/jakedolan443/fernmail/fix/frontend_preferences/docs/screenshots/frontend-preferences-conversation.png)

These are committed visual review captures from the frontend harness, with the changed areas highlighted in red.

## Verification

- `pnpm run build:main`
- `pnpm test:run` — 45 files, 806 tests
- Targeted ESLint — no new errors; only the existing `Conversation` component-name and unused-route-param warnings remain.
