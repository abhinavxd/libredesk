# Libredesk Design System

Reference for colors, typography, spacing, radius, elevation, and component conventions
across both apps (agent dashboard + livechat widget).

**Source of truth:**
- Tokens (CSS variables): `shared-ui/assets/styles/main.scss` - `:root, .light` and `.dark` blocks
- Tailwind mapping: `tailwind.config.cjs` - `theme.extend.colors` and `borderRadius`
- Primitives: `shared-ui/components/ui/` (shadcn-vue)

**Golden rule:** never hardcode a value that a token exists for. Use `bg-success`, not
`bg-green-600`. Use `rounded-lg`, not `rounded`. Every token has a light and a dark value;
if you add one, define both.

---

## 1. Color tokens

HSL channel triples in CSS variables, consumed via `hsl(var(--x))` (Tailwind classes like
`bg-primary` do it for you). Opacity modifiers work: `bg-success/10`, `text-primary/80`.

| Token | Light | Dark | Use for |
|---|---|---|---|
| `background` | `0 0% 99.2%` | `120 2.6% 7.6%` | app/content surface |
| `foreground` | `0 0% 1%` | `150 6% 93%` | primary text |
| `foreground-lighter` | `0 0% 41%` | `150 1% 60%` | idle sidebar nav items |
| `card` / `card-foreground` | `0 0% 100%` / `0 0% 1%` | `120 2% 10%` / `150 6% 93%` | raised card surface |
| `popover` / `popover-foreground` | `0 0% 100%` / `0 0% 1%` | `120 2% 12%` / `150 6% 93%` | menus, popovers, dropdowns |
| `primary` / `primary-foreground` | `152 39% 30%` / `0 0% 100%` | `152 58% 54%` / `120 2.6% 7.6%` | brand, active state, unread badges, primary buttons |
| `secondary` / `secondary-foreground` | `0 0% 96%` / `0 0% 1%` | `150 3% 12%` / `150 6% 93%` | secondary buttons, outgoing message bubbles |
| `muted` / `muted-foreground` | `0 0% 96%` / `0 0% 27%` | `150 3% 13%` / `120 1% 74%` | muted backgrounds, captions/meta text, secondary labels |
| `accent` / `accent-foreground` | `0 0% 95%` / `0 0% 1%` | `150 3% 15%` / `150 6% 93%` | hover and selected states |
| `destructive` / `destructive-foreground` | `2 47% 46%` / `9 100% 99%` | `4 92% 74%` / `12 38% 3%` | errors, delete, SLA breached, overdue, offline |
| `success` / `success-foreground` | `142 72% 37%` / `0 0% 98%` | `142 55% 55%` / `142 40% 10%` | positive/met, verified, online, connected, delivered |
| `warning` / `warning-foreground` | `39 85% 43%` / `24 45% 2%` | `36 87% 62%` / `24 45% 2%` | away, pending, SLA approaching, connecting (as background) |
| `warning-600` | `35 92% 33%` | `36 87% 62%` | warning as TEXT/icon color. Plain `warning` fails 4.5:1 on light |
| `link` | `210 100% 40%` | `210 90% 66%` | links inside rendered email/message content only. UI links use `.link-style` |
| `border` | `0 0% 91%` | `150 2% 16%` | all borders/dividers |
| `input` | `0 0% 85%` | `150 2% 19%` | form field borders |
| `ring` | `151 41% 45%` | `152 45% 33%` | focus rings |
| `private` | `35 90% 94%` | `30 35% 18%` | private-note background tint |
| `canvas` | `0 0% 82%` | `120 3% 4%` | app gutter behind floating panels (deepest surface) |

**Sidebar tokens** (left nav chrome): `sidebar-background` matches `background` in both
themes, plus `sidebar-foreground`, `sidebar-primary`, `sidebar-accent`,
`sidebar-accent-foreground`, `sidebar-border`, `sidebar-ring`.

**Chart tokens** (unovis): `--vis-primary-color: var(--primary)`,
`--vis-secondary-color: var(--success)`, `--vis-text-color: var(--muted-foreground)`.

### Semantic status mapping

Color carries meaning; use the semantic token, never a raw palette color.

- **success (green):** SLA met, identity verified, agent online, widget connected, message delivered/read
- **warning (amber):** agent away, SLA approaching/remaining, widget connecting, no-internet banner
- **destructive (red):** error, delete, SLA breached, SLA overdue
- **primary (green):** brand identity, active nav, unread count badges, primary actions
- **foreground / muted:** neutral data (counts, totals, timestamps). Do not color a number
  unless the color means something. On the reports dashboard numbers are neutral; only
  met=success and breached=destructive are colored.

### Deliberate exceptions

- `features/conversation/message/attachment/BubbleAttachmentItem.vue` colors attachment
  icons by file type (pdf=red, spreadsheet=green, doc=blue, archive=amber, audio=purple)
  with raw palette classes. The color is file-type identity, not status, and no token means
  "blue = document". Leave it.
- `components/editor/TextEditor.vue` - styles for rendered email HTML. Emails are
  standalone documents, not themed.
- `features/admin/inbox/LivechatInboxForm.vue` / `LivechatWidgetPreview.vue` - the
  customer-configurable widget brand color and its defaults.
- `features/conversation/ReplyBox.vue` - a `linear-gradient(#000 0 0)` CSS mask trick, not
  a color choice.

---

## 2. Surfaces and depth

Both themes use three tiers so panels read as floating, not flat:

```text
canvas (deepest gutter behind the panels)
  └─ background (app content + sidebar chrome, same value)
        └─ card / popover (lifted by border + shadow-sm)
```

In light mode `card` is pure white against a near-white background. In dark mode `card` and
`popover` are lighter than the background, so lift comes from the surface, not the shadow.

---

## 3. Typography

Font: Instrument Sans. Sizes: `text-xs` 12 · `text-sm` 14 · `text-base` 16 · `text-lg` 18 ·
`text-xl` 20 · `text-2xl` 24 · `text-3xl` 30. Weights: 400 body · 500 labels · 600 headings ·
700 rare emphasis.

| Role | Style |
|---|---|
| Page / panel title | `text-xl font-semibold` |
| KPI / stat value | `text-2xl` (or `text-3xl`) `font-bold tabular-nums` |
| Section label | `.sidebar-section-label` utility, or the classes it applies |
| Body | `text-sm` |
| Caption / meta / helper | `text-xs text-muted-foreground` |

Use `tabular-nums` for any numeric column, timer, or metric to stop width jitter.

---

## 4. Radius

`--radius` = `0.5rem` (8px). The Tailwind scale derives from it:

| Class | Value | Use for |
|---|---|---|
| `rounded-xl` | radius + 4 (12px) | large surfaces, widget window preview, dashboard message bubbles |
| `rounded-lg` | radius (8px) | cards, containers, dialogs |
| `rounded-md` | radius - 2 (6px) | buttons, inputs, chips, badges, small interactive |
| `rounded-sm` | radius - 4 (4px) | tiny insets |
| `rounded-full` | - | avatars, status dots, count badges, pills |

**Never use bare `rounded`.** It is Tailwind's fixed 4px and ignores the token.

---

## 5. Elevation

| Class | Use for |
|---|---|
| `shadow-sm` | cards, `.box`, default buttons |
| `shadow-md` | popovers, dropdown menus, hover/floating elements |
| `shadow-lg` | dialogs, modals, the widget window |

**Never use bare `shadow`.** Depth comes from the surface tiers and borders, not heavy
shadows.

---

## 6. Spacing

Follow a 4 / 8px rhythm for padding and gaps. Vertical section spacing tiers: 16 / 24 / 32 /
48. Keep the dense-desk feel; this is a high-volume support tool, not a marketing page.

---

## 7. Utilities

Defined in `main.scss`. Prefer them over repeating the class list.

| Utility | Expands to | Use for |
|---|---|---|
| `.box` | `border shadow-sm rounded-lg` | the standard card surface |
| `.sidebar-section-label` | `text-xs font-medium uppercase tracking-wider text-muted-foreground` | sidebar group headers (Views, Team Inboxes) |
| `.link-style` | `text-muted-foreground underline underline-offset-4 hover:text-foreground` | UI links. Not brand-colored, so links in chrome stay quiet |

---

## 8. Components

Reuse `shared-ui/components/ui/` primitives; do not hand-roll a styled `<button>`/`<input>`
when one exists.

**Button** - use the `size` variant, never an ad-hoc `h-*`:

| size | height | notes |
|---|---|---|
| `default` | h-9 | standard |
| `sm` | h-8 | dense (text-xs) |
| `xs` | h-7 | very dense |
| `lg` | h-10 | prominent |
| `icon` | h-9 w-9 | icon-only |

Variants: `default` (primary), `destructive`, `outline`, `secondary`, `ghost`, `link`.
`h-8 w-8` ghost triggers for dense table row actions are an intentional pattern.

Every button inside a `<form>` that is not the submit button needs an explicit
`type="button"`. HTML defaults to `type="submit"`, so a typeless button submits the form on
click and gets activated when the user presses Enter in any input. This includes buttons in
child components rendered inside a parent's form.

**Badge** variants: `default`, `secondary`, `destructive`, `success`, `outline`.

**AlertDialogAction** takes a `variant` prop, so destructive confirms use
`variant="destructive"` instead of a hand-written class.

**Sidebar nav items** are `font-medium text-foreground-lighter`; hover and active states use
`bg-sidebar-accent` (hover at `/50`). Group headers use `.sidebar-section-label`.

**Tooltip** is `bg-foreground text-background`, not brand-colored.

**Table row actions** stay hidden until hover: `[@media(hover:hover)]:opacity-0` with
`group-hover/row:opacity-100`, plus `focus-within:!opacity-100` and
`[&:has([data-state=open])]:!opacity-100`. The media query keeps them visible on touch and
focus-within keeps them reachable by keyboard.

---

## 9. Checklist

Before merging UI work:

- [ ] No hardcoded palette colors (`bg-green-600`, `text-blue-500`, ...). Only exception: file-type icons.
- [ ] No bare `rounded` or `shadow`.
- [ ] Every non-submit button in a form has `type="button"`.
- [ ] Color on a number or element means something, and is not decoration.
- [ ] Both light and dark verified.
- [ ] New shared words go through i18n (`i18n/en-US.json`); reused nouns in `globals.terms`.
- [ ] Reused an existing `ui/` primitive rather than building a variant.
