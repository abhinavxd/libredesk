# Contributor role, review queue, Users menu and Compose New

Status: implemented on `feat/contributor-review`.

## Goals

1. A third role, **Contributor**, for people who may read mail but whose outgoing email must be approved.
2. A **review queue**: Admins and Agents approve or deny Contributor emails before they leave Fernmail.
3. One **Users** menu where Admins see every user, set their role and addresses, add, disable and delete users.
4. **Compose New**: start a new email from a chosen address. Admins and Agents send directly; Contributors go through review.
5. **Open addresses are retired.** Admins see every address; everyone else needs an explicit grant.

## Roles and permissions

| Permission | Contributor | Agent | Admin |
|---|:-:|:-:|:-:|
| `conversations:read`, `conversations:read_all`, `messages:read` | ✓ | ✓ | ✓ |
| `conversations:create` (Compose New) — **new** | ✓ | ✓ | ✓ |
| `reviews:submit` (replies and new emails are held for review) — **new** | ✓ | | |
| `messages:write` (send directly) | | ✓ | ✓ |
| `reviews:manage` (see queue, approve, deny) — **new** | | ✓ | ✓ |
| `messages:write_private`, `conversations:update_status` | | ✓ | ✓ |
| Admin-only management permissions (unchanged) | | | ✓ |

Sending rule, enforced on the server for replies and new emails alike:

- `messages:write` → the email is queued for delivery immediately.
- otherwise `reviews:submit` → the email becomes a review submission.
- otherwise → 403.

Contributors cannot write internal notes or change conversation status. They can mark mail read, since read state is personal.

## Address access

- The Admin role and the System user can access every address.
- Everyone else needs a direct user grant or a team grant. There is no "open to everyone" setting any more.
- **Upgrade preserves today's access.** For every address that is currently open, each enabled non-admin agent receives an explicit grant before the open flag is removed, so nobody loses mail on upgrade. Admins can then trim access in the Users menu.
- The same rule applies to the legacy transport-level fallback (`can_access_inbox`), so no open path remains.

## Users menu

Admin → Workspace → **Users** (permission `users:manage`). It replaces per-user access editing on the address form.

- A scrollable, searchable list of every user with name, email, role, address count and a disabled badge.
- Selecting a user opens an editor with:
  - **Role**: Admin, Agent or Contributor (one per user). Users holding a legacy custom role show that name until changed.
  - **Addresses**: a checklist of every address. When Admin is selected, it notes that Admins see all addresses.
  - **Enabled**: on or off.
- **Save** opens a confirm dialog summarising the changes (for example "Role: Agent → Admin, +2 addresses"), with Confirm and Cancel.
- **Add user**: name, email, role and addresses, with an optional set-password email (on by default). Confirm and Cancel.
- **Delete user**: a confirm dialog naming the user.
- Results and errors use the existing toasts.

Server-side safeguards (not just UI):

- The last enabled Admin cannot be demoted, disabled or deleted.
- You cannot change your own role, disable yourself or delete yourself.
- The built-in System user is not listed and cannot be edited.
- Saving a user's access disconnects their sessions so the new permissions apply at once (existing behaviour).

The address form loses its "Agents" checklist and "Restricted" toggle. It keeps Teams, since there is no team UI, and shows a read-only list of people with access that links to Users.

## Review queue

### Data

`outbound_reviews`:

| Column | Notes |
|---|---|
| `id`, `uuid`, timestamps | |
| `kind` | `reply` or `new` |
| `status` | `pending`, `approved`, `denied`, `withdrawn` |
| `author_id` | the Contributor |
| `address_id` | sending address; access is checked against it |
| `conversation_id` | set for replies; set on approval for new emails |
| `subject` | new emails only |
| `content`, `to`, `cc`, `bcc` | the email as submitted |
| `reviewer_id`, `reviewed_at`, `decision_note` | the decision |
| `message_id` | the delivered message after approval |

`outbound_review_media(review_id, media_id)` keeps attachments safe from upload cleanup while a submission exists.

A Contributor can have at most one pending reply per conversation. Pending submissions are not messages: they are never delivered, searched or counted as unread.

### Flow

1. **Submit.** A Contributor presses **Send for review**. The composer clears and a toast says "Sent for review". The thread shows their reply as a blue "Awaiting review" card, and the composer is locked with a **Withdraw** button until a decision is made.
2. **Notify.** Connected reviewers who can access the address get a toast, and their blue sidebar badge increases.
3. **Review.** **Review** in the left sidebar opens `/reviews`, which looks like an inbox:
   - The list shows pending submissions, oldest first, each with author, address, subject, preview and age.
   - The detail pane shows the earlier messages in the conversation (read-only), then the submission highlighted in blue with its recipients and attachments.
   - **Approve** and **Deny** each open a confirm dialog. Deny takes an optional reason.
4. **Approve.** The email is queued exactly as if the Contributor had sent it: From is the address, the author is the Contributor, and the approver is recorded. A new email gets its conversation at this point. The Contributor gets a toast.
5. **Deny.** The Contributor gets a toast with the reason. A reply goes back into their composer as a draft, including recipients and attachments, and the composer shows the reason. A new email appears as **Returned** in their Review list, and **Edit and resubmit** reopens Compose with everything filled in. They can also discard it.
6. **Withdraw.** While still pending, the Contributor can pull a submission back. It is handled like a denial with no reason.

Decisions are a single conditional update, so only one reviewer (or the withdrawing author) wins. Anyone else gets an "already reviewed" toast.

Contributors also see **Review** in the sidebar, listing their own pending and returned submissions. Their badge counts their pending items.

The badge uses a new semantic blue `--review` colour token (light and dark), not a hard-coded palette colour.

## Compose New

- A **Compose New** button sits at the top of the mail sidebar for anyone with `conversations:create`.
- The dialog has:
  - **From**: a dropdown limited to enabled addresses the user can access.
  - **To** (required), **CC**, **BCC**.
  - **Subject** (required).
  - **Body** and **attachments**.
- Admin or Agent: **Send** creates the conversation on that address, queues the email and opens the new conversation, with a toast.
- Contributor: the button reads **Send for review** and creates a `new` submission, with a toast.
- The server re-checks address access and that the address is enabled. The conversation's contact is the first To recipient.

## API

| Method and path | Permission | Purpose |
|---|---|---|
| `POST /api/v1/compose` | `conversations:create` | Send directly, or create a review submission |
| `POST /api/v1/conversations/{uuid}/reviews` | `reviews:submit` | Submit a reply for review |
| `GET /api/v1/reviews` | `reviews:manage` or `reviews:submit` | Queue for reviewers, own submissions for Contributors |
| `GET /api/v1/reviews/counts` | same | Badge count |
| `GET /api/v1/reviews/{uuid}` | same | Submission and conversation context |
| `POST /api/v1/reviews/{uuid}/approve` | `reviews:manage` | Approve |
| `POST /api/v1/reviews/{uuid}/deny` | `reviews:manage` | Deny with an optional reason |
| `POST /api/v1/reviews/{uuid}/withdraw` | author only | Withdraw |
| `PUT /api/v1/reviews/{uuid}` | author only | Edit and resubmit a returned new email |
| `DELETE /api/v1/reviews/{uuid}` | author only | Discard a returned new email |
| `POST /api/v1/reviews/{uuid}/dismiss` | author only | Hide a returned reply's notice from the composer |
| `GET /api/v1/conversations/{uuid}/reviews` | `conversations:read` | Review cards for a thread |
| `GET /api/v1/admin/users` | `users:manage` | Users with role, addresses and enabled state |
| `POST /api/v1/admin/users` | `users:manage` | Add a user |
| `PUT /api/v1/admin/users/{id}/access` | `users:manage` | Set role, addresses and enabled together |
| `DELETE /api/v1/admin/users/{id}` | `users:manage` | Delete a user |

Live updates go over the existing websocket: `review_created` and `review_updated`.

## Upgrade (migration v3.7.0)

1. Create the Contributor role. Add `conversations:create` and `reviews:manage` to Agent and Admin.
2. Grant explicit access to every currently open address for each enabled non-admin agent, then drop the open flag and update both access functions.
3. Create `outbound_reviews` and `outbound_review_media`, and exclude review attachments from upload cleanup.

Fresh installs get the same schema directly.

## Out of scope

- Reviewers editing a submission before approving.
- Contributor notes and status changes.
- A password-reset button in the Users menu.
- Team management.
