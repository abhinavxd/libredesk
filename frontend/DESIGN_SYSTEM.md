# Libredesk Design System

The single reference for colors, typography, spacing, radius, elevation, and component
conventions across both apps (agent dashboard + livechat widget).

**Source of truth:**
- Tokens (CSS variables): `shared-ui/assets/styles/main.scss` - `:root, .light` and `.dark` blocks
- Tailwind mapping: `tailwind.config.cjs` - `theme.extend.colors` and `borderRadius`
- Primitives: `shared-ui/components/ui/` (shadcn-vue)

**Golden rule:** never hardcode a value that a token exists for. Use `bg-success`, not
`bg-green-600`. Use `rounded-lg`, not `rounded`. Every token has a light and a dark value;
if you add one, define both.

---

## 1. Color tokens

All colors are HSL channel triples stored as CSS variables and consumed via
`hsl(var(--x))` (Tailwind classes like `bg-primary` do this for you). Opacity modifiers
work: `bg-success/10`, `text-primary/80`.

| Token | Light | Dark | Use for |
|---|---|---|---|
| `background` | `0 0% 100%` | `240 7% 11%` | app/content surface |
| `foreground` | `240 10% 3.9%` | `240 4% 94%` | primary text |
| `card` / `card-foreground` | `0 0% 100%` / `240 10% 3.9%` | `240 5% 13%` / `240 4% 94%` | raised card surface |
| `popover` / `popover-foreground` | `0 0% 100%` / `240 10% 3.9%` | `240 5% 15%` / `240 4% 94%` | menus, popovers, dropdowns |
| `primary` / `primary-foreground` | `234 52% 54%` / `46 33% 97%` | `235 58% 68%` / `235 35% 10%` | brand, active state, links, unread badges, primary buttons |
| `secondary` / `secondary-foreground` | `240 4.8% 95.9%` / `240 5.9% 10%` | `240 5% 16%` / `240 4% 94%` | secondary buttons, outgoing message bubbles |
| `muted` / `muted-foreground` | `240 4.8% 95.9%` / `240 3.8% 46.1%` | `240 5% 16%` / `240 4% 60%` | muted backgrounds, captions/meta text, secondary labels |
| `accent` / `accent-foreground` | `240 5% 89%` / `240 5.9% 10%` | `240 5% 19%` / `0 0% 98%` | hover and selected states |
| `destructive` / `destructive-foreground` | `0 84.2% 60.2%` / `0 0% 98%` | `1 100% 69%` / `0 0% 100%` | errors, delete, SLA breached, overdue, offline |
| `success` / `success-foreground` | `142 72% 37%` / `0 0% 98%` | `142 55% 55%` / `142 40% 10%` | positive/met, verified, online, connected, delivered |
| `warning` / `warning-foreground` | `38 92% 45%` / `0 0% 98%` | `38 92% 60%` / `38 60% 10%` | away, pending, SLA approaching, connecting |
| `border` | `240 6% 87%` | `240 5% 18%` | all borders/dividers |
| `input` | `240 6% 87%` | `240 5% 16%` | form field borders |
| `ring` | `234 52% 54%` | `235 58% 68%` | focus rings |
| `private` | `35 90% 94%` | `30 35% 18%` | private-note background tint |
| `canvas` | `240 6% 86%` | `240 7% 15%` | app gutter behind floating panels (deepest surface) |

**Sidebar tokens** (the left nav "chrome"): `sidebar-background` `240 5% 95%` (light) /
`240 5% 7%` (dark), plus `sidebar-foreground`, `sidebar-primary`, `sidebar-accent`,
`sidebar-accent-foreground`, `sidebar-border`, `sidebar-ring`.

**Chart tokens** (unovis): `--vis-primary-color: var(--primary)`,
`--vis-secondary-color: var(--success)`, `--vis-text-color: var(--muted-foreground)`.

### Semantic status mapping

Color carries meaning; use the semantic token, never a raw palette color.

- **success (green):** SLA met, identity verified, agent online, widget connected, message delivered/read
- **warning (amber):** agent away, SLA approaching/remaining, widget connecting, no-internet banner
- **destructive (red):** error, delete, SLA breached, SLA overdue
- **primary (indigo):** brand identity, active nav, links, unread count badges, primary actions
- **foreground / muted:** neutral data (counts, totals, timestamps). Do not color a number
  unless the color means something. See the reports dashboard: numbers are neutral, only
  met=success / breached=destructive are colored.

### The one exception: file-type icons

`features/conversation/message/attachment/BubbleAttachmentItem.vue` colors attachment icons
by file type (pdf=red, spreadsheet=green, doc=blue, archive=amber, audio=purple) using raw
palette classes. This is **intentional** - the color is file-type *identity* (a Gmail/Slack
convention), not status, and there is no semantic token for "blue = document." Leave it.

### Legitimate raw-hex usages (not chrome, do not tokenize)

- `components/editor/TextEditor.vue` - styles for rendered email HTML (emails are standalone documents, not themed)
- `features/admin/inbox/LivechatInboxForm.vue` / `LivechatWidgetPreview.vue` - the customer-configurable widget brand color and its defaults
- `features/conversation/ReplyBox.vue` - a `linear-gradient(#000 0 0)` CSS mask trick (not a color choice)

---

## 2. Surfaces & depth

Light mode uses three tiers so panels float instead of being flat white-on-white:

```
canvas (240 6% 86%, deepest gutter)
  └─ chrome  (sidebar 240 5% 95%, gray nav)
  └─ content (background, white 100%)
        └─ card / popover (white, lifted by border + shadow-sm)
```

Dark mode inverts the relationship (chrome is *darker* than content) and was already
healthy, so it was largely left alone.

`.box` utility (main.scss) = `border shadow-sm rounded-lg` - the standard card.

---

## 3. Typography

Sizes in use (Tailwind): `text-xs` 12 · `text-sm` 14 · `text-base` 16 · `text-lg` 18 ·
`text-xl` 20 · `text-2xl` 24 · `text-3xl` 30. Weights: 400 body · 500 labels/medium ·
600 headings · 700 rare emphasis. Font: Instrument Sans.

| Role | Style |
|---|---|
| Page / panel title (Inbox, Admin, page headers) | `text-xl font-semibold` |
| KPI / stat value (reports, tiles) | `text-2xl` (or `text-3xl`) `font-bold tabular-nums` |
| Section label (sidebar groups, reports section titles) | `text-xs`/`text-sm` `font-medium uppercase tracking-wider text-muted-foreground` |
| Body | `text-sm` |
| Caption / meta / helper | `text-xs text-muted-foreground` |

Use `tabular-nums` for any numeric column, timer, or metric to prevent width jitter.

---

## 4. Radius

`--radius` = `0.5rem` (8px). Tailwind scale derives from it:

| Class | Value | Use for |
|---|---|---|
| `rounded-xl` | radius + 4 (12px) | large surfaces, widget window preview |
| `rounded-lg` | radius (8px) | cards, containers, dialogs, message bubbles |
| `rounded-md` | radius - 2 (6px) | buttons, inputs, chips, badges, small interactive |
| `rounded-sm` | radius - 4 (4px) | tiny insets |
| `rounded-full` | - | avatars, status dots, count badges, pills |

**Never use bare `rounded`** - it is Tailwind's fixed 4px and ignores the token.

---

## 5. Elevation

| Class | Use for |
|---|---|
| `shadow-sm` | cards, `.box`, default buttons |
| `shadow-md` | popovers, dropdown menus, hover/floating elements |
| `shadow-lg` | dialogs, modals, the widget window |

**Never use bare `shadow`.** Depth in light mode comes mostly from the canvas/surface
tiers and borders, not heavy shadows.

---

## 6. Spacing

Follow a 4 / 8px rhythm for padding and gaps. Vertical section spacing tiers: 16 / 24 / 32 /
48. Keep the dense-desk density - this is a high-volume support tool, not a marketing page.

---

## 7. Components

Reuse `shared-ui/components/ui/` (shadcn-vue) primitives; do not hand-roll styled
`<button>`/`<input>` when a primitive exists.

**Button** (`ui/button`) - use the `size` variant, never an ad-hoc `h-*`:

| size | height | notes |
|---|---|---|
| `default` | h-9 | standard |
| `sm` | h-8 | dense (text-xs) |
| `xs` | h-7 | very dense |
| `lg` | h-10 | prominent |
| `icon` | h-9 w-9 | icon-only |

Variants: `default` (primary), `destructive`, `outline`, `secondary`, `ghost`, `link`.
(Note: `h-8 w-8` ghost triggers used for dense table/row actions are an intentional pattern.)

**Sidebar section labels:** collapsible group headers (Views, Team Inboxes, Shared Views)
use `text-xs font-medium uppercase tracking-wider text-muted-foreground` so they read as
group headers, distinct from the primary nav items (which are normal weight with icons).

---

## 8. Conventions checklist

Before merging UI work:

- [ ] No hardcoded palette colors (`bg-green-600`, `text-blue-500`, ...) - use tokens. Only exception: file-type icons.
- [ ] No bare `rounded` or `shadow` - use the scale.
- [ ] Color used on a number/element means something (semantic), not decoration.
- [ ] Both light and dark verified.
- [ ] New shared words go through i18n (`i18n/en-US.json`); reused nouns in `globals.terms`.
- [ ] Reused an existing `ui/` primitive rather than building a variant.

---

## 9. Not yet enforced (options for later)

The system currently relies on review + this doc. Two ways to make drift impossible:

1. **Lint gate** - a CI/pre-commit script that greps source for the banned patterns above
   and fails the build. Cheap, loud, catches everything. Standard for teams that want the
   system to hold.
2. **Delete the default palette** - override `theme.colors` in `tailwind.config.cjs` to keep
   only `transparent/current/inherit/white/black` + the semantic tokens, so `bg-green-600`
   literally does not compile. Strongest lock, but requires tokenizing the file-type icons
   first (they would break), and fails quietly (no class = no style) rather than with a message.

Neither is implemented yet. #1 is the usual choice; #2 is the structural version.
