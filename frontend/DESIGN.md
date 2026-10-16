# libredesk Design System

Reference for dashboard styling and the shared components used by the livechat widget.

**Source of truth:**

- Tokens (CSS variables): `shared-ui/assets/styles/main.scss` - `:root, .light` and `.dark` blocks
- Tailwind mapping: `tailwind.config.cjs` - `theme.extend.colors` and `borderRadius`
- Primitives: `shared-ui/components/ui/` (shadcn-vue)

Use existing semantic colors and radius classes when styling dashboard controls.
Color tokens define light and dark values, while radius uses one shared value.

---

## 1. Color tokens

Theme variables store HSL channels, which Tailwind consumes through `hsl(var(--x))`.
Opacity modifiers such as `bg-success/10` work with these semantic color classes.

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
| `destructive` / `destructive-foreground` | `2 47% 46%` / `9 100% 99%` | `4 92% 74%` / `12 38% 3%` | errors, delete, SLA breached, overdue |
| `success` / `success-foreground` | `142 72% 37%` / `0 0% 98%` | `142 55% 55%` / `142 40% 10%` | positive/met, verified, online, connected, delivered |
| `warning` / `warning-foreground` | `39 85% 43%` / `24 45% 2%` | `36 87% 62%` / `24 45% 2%` | away, pending, SLA approaching, connecting (as background) |
| `warning-600` | `35 92% 33%` | `36 87% 62%` | warning text and icons. Plain `warning` fails 4.5:1 on light |
| `link` | `210 100% 40%` | `210 90% 66%` | rendered content and toast links. Dashboard links use `.link-style` |
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

Status colors use semantic tokens, while neutral counts use foreground colors.

- **success (green):** SLA met, identity verified, agent online, widget connected, message delivered/read
- **warning (amber):** manual agent away, SLA approaching/remaining, widget connecting, no-internet banner
- **destructive (red):** error, delete, SLA breached, SLA overdue
- **primary (green):** brand identity, active nav, unread count badges, primary actions
- **foreground / muted:** offline indicators and neutral data (counts, totals, timestamps).
  Report counts use success for met SLAs and destructive for breached SLAs.

### Deliberate exceptions

- `features/conversation/message/attachment/BubbleAttachmentItem.vue` colors attachment
  uses red for PDF, green for spreadsheets, blue for documents, amber for archives, and purple for audio.
  These colors identify file types and remain separate from status tokens.
- `components/editor/TextEditor.vue` - styles for rendered email HTML. Emails are
  standalone documents with their own formatting.
- `features/admin/inbox/LivechatInboxForm.vue` / `LivechatWidgetPreview.vue` - the
  customer-configurable widget brand color and its defaults.
- `features/conversation/ReplyBox.vue` uses `linear-gradient(#000 0 0)` for a CSS mask.
- `StatusDot.vue` uses white check and clock strokes over status colors.
- `.email-light-canvas` keeps incoming email HTML white with dark text in dark mode.

---

## 2. Surfaces and depth

Both themes separate gutters, content surfaces, and raised panels:

```text
canvas (deepest gutter behind the panels)
  └─ background (app content + sidebar chrome, same value)
        └─ card / popover (lifted by border + shadow-sm)
```

In light mode, `card` is white against the slightly darker content background.
Dark cards and popovers use lighter surfaces to separate them from the background.

---

## 3. Typography

Geist remains the shared font for the dashboard and widget.
Typography classes live in `shared-ui/assets/styles/main.scss`, inside `@layer components`.

| Role | Class | Size | Weight | Line height |
|---|---|---|---|---|
| Page, sidebar, card, and form section headings | `ui-heading` | 16px | 600 | 24px |
| Nested headings and compact item titles | `ui-subheading` | 14px | 600 | 20px |
| Descriptions and form hints | `ui-description` | 14px | 400 | 20px |
| Dialog and sheet titles | `ui-dialog-heading` | 18px | 600 | 28px |
| Authentication, welcome, and contact profile headings | `ui-display-heading` | 24px | 600 | 32px |

Use these classes for the listed roles instead of repeating size and weight utilities.
Keep heading levels semantic, since an `h3` can still use the standard section heading style.
`CardTitle`, `DialogTitle`, `SheetTitle`, and shared description components apply their defaults automatically.

`sidebar-section-label` remains the 12px muted uppercase style for compact sidebar groups.
Field labels remain 14px medium, while timestamps and compact metadata can use 12px.
Use `tabular-nums` for numeric columns and metrics to keep their character widths stable.
Widget previews and rendered email HTML retain their own content typography.
Report metric values use 24px or 30px bold text with `tabular-nums`.

---

## 4. Radius

`--radius` = `0.5rem` (8px). The Tailwind scale derives from it:

| Class | Value | Use for |
|---|---|---|
| `rounded-xl` | radius + 4 (12px) | large surfaces, preview container, dashboard message bubbles |
| `rounded-2xl` | 16px (Tailwind) | customer widget frame and widget message bubbles |
| `rounded-lg` | radius (8px) | cards, containers, dialogs |
| `rounded-md` | radius - 2 (6px) | buttons, inputs, chips, badges, small interactive |
| `rounded-sm` | radius - 4 (4px) | tiny insets |
| `rounded-full` | - | avatars, status dots, count badges, pills |

Named radius classes follow the shared token and remain the default for dashboard controls.
The incoming email canvas retains Tailwind's fixed `rounded` class as an existing exception.
The reply area uses `rounded-t-xl` where its surface meets the message thread.

---

## 5. Elevation

| Class | Use for |
|---|---|
| `shadow-sm` | cards, `.box`, default buttons |
| `shadow-md` | popovers, dropdown menus, hover/floating elements |
| `shadow-lg` | dialogs, modals, the widget window |

Named shadow sizes describe surface elevation.
Badge default and destructive variants retain the primitive's existing `shadow` class.

---

## 6. Spacing

Spacing follows a 4px rhythm, with 16px, 24px, and 32px common between groups.
Section headings leave 16px before their controls, while forms use 24px or 32px between sections.

`FormGrid` fits columns with a 15rem minimum, reduced to 12rem by `compact`.
Forms needing stable widths use container queries and explicit columns instead of stretching lone fields.
A conditional Team field does not change Shared view Visibility's column width.

`AdminSplitLayout` adds a 16rem help column when the container reaches 56rem.
Help panels use `bg-muted` in light mode and `dark:bg-muted/40` in dark mode.
Preview layouts use 27rem at 56rem, expanding to 51.25rem at 94rem.

`AdminToolbar` places tabs in its left slot and search beside actions on the right.
Empty search controls occupy 8rem, while focused controls and queries occupy 15rem.
Pointer focus expands search over 150ms, while keyboard focus and reduced motion use an immediate change.

---

## 7. Utilities

Defined in `main.scss`. Prefer them over repeating the class list.
The typography table above lists the shared heading and description classes.

| Utility | Expands to | Use for |
|---|---|---|
| `.box` | `border shadow-sm rounded-lg` | the standard card surface |
| `.sidebar-section-label` | `text-xs font-medium uppercase tracking-wider text-muted-foreground` | sidebar group headers (Views, Team Inboxes) |
| `.link-style` | `text-muted-foreground underline underline-offset-4 hover:text-foreground transition-colors` | links in dashboard controls |

---

## 8. Components

Use the existing `shared-ui/components/ui/` primitives for styled buttons, inputs, and other controls.

**Button** sizes come from `buttonVariants` in the shared component:

| size | height | notes |
|---|---|---|
| `default` | h-9 | standard |
| `sm` | h-8 | dense (text-xs) |
| `xs` | h-7 | very dense |
| `lg` | h-10 | prominent |
| `icon` | h-9 w-9 | icon-only |

Variants: `default` (primary), `destructive`, `outline`, `secondary`, `ghost`, `link`.
Table action triggers use 32px squares and increase to 44px below the `md` breakpoint.
The conversation Details toggle uses 32px at `lg` and above, with a 16px icon.
Its target increases to 44px below `lg`.
The contact header collapse control uses a muted 14px icon within 28px desktop and 44px mobile targets.

Every button inside a `<form>` that is not the submit button needs an explicit
`type="button"`. HTML defaults to `type="submit"`, so a typeless button submits the form on
click and gets activated when the user presses Enter in any input. This includes buttons in
child components rendered inside a parent's form.

**Badge** variants: `default`, `secondary`, `destructive`, `success`, `outline`.
Approved WhatsApp templates use `success`, matching enabled statuses elsewhere in admin.

`AlertDialogAction` accepts `variant="destructive"` for confirmations that delete or remove data.

Main sidebar items use `font-medium text-foreground-lighter` and `bg-sidebar-accent` for active states.
Hover uses `bg-sidebar-accent/50`, while group headers use `.sidebar-section-label`.
Nested sidebar links use `text-sidebar-foreground` with the same active surface.

**Tooltip** uses `bg-foreground text-background`, 12px text, and `rounded-md`.

**Table row actions** use `can-hover:opacity-0` and `can-hover:group-hover/row:opacity-100`.
The variant requires both hover support and a fine pointer.
Focus and open menus reveal the controls, while touch devices keep them visible.
`ReorderButtons` applies the same hover and keyboard focus visibility within `group/reorder` rows.
Prechat fields and featured help articles use this component.

**Inputs and selects** use 36px height, with 16px text below `sm` and 14px above it.
`TimeField` shares these dimensions and renders keyboard editable hour and minute segments through Reka UI.
Visible labels identify fields. Placeholders add examples or format guidance rather than repeating labels.
`SelectTag` hides its placeholder after selection and retains its accessible input label.

Juno's persona selector uses 28px height and 12px text within the conversation sidebar.
Its empty heading uses `ui-heading`, while the description and suggestion labels use 14px text.
Suggestion buttons wrap their labels and retain a minimum height of 36px.

Inbox channel choices use outlined buttons inside `FormGrid`, stacking when the content area becomes narrow.
WhatsApp callback details appear after the inbox has a webhook URL.

**StatusDot** renders a success check for `online` and automatic `away`.
Manual away states use a warning clock, while offline uses a muted outline circle.

**LoadingOverlay** owns page loading feedback, while nested DataTable suppresses its duplicate spinner.
Standalone tables retain their loading spinner and keep the busy state accessible.

Message delivery spinners appear after 100ms while a message remains pending.
Completion or removal cancels the delay and hides the spinner immediately.
`MessageBubble` uses the existing `delayedLoading` helper with zero minimum visible time.
[Ant Design's delayed spinner](https://ant.design/components/spin/) documents this behavior for avoiding brief loading flashes.

Sender avatars sit beside names above message bubbles at desktop and mobile widths.
Consecutive messages from the same sender share one name and avatar within the grouping window.
File and image attachments share 112px square tiles, with controls revealed per attachment on hover or keyboard focus.
Touch devices keep attachment controls visible.
Message content width limits apply within `.native-html`, preserving fixed attachment dimensions.
Older messages load when the thread reaches the top, preserving the visible message position.

Conversation rows show the nearest pending SLA badge first in the header icon group.
The badge shows a timer and duration, or an alert icon when overdue.
Last message age stays at the far right.
Waiting shows a 12px clock and elapsed time beside ownership, with the exact start time on hover.
Agent avatars use circles throughout navigation and conversation lists. Team avatars use rounded squares with initials.
Team forms keep Emoji on its own compact row and align the remaining fields in equal columns.
The emoji defaults to 👥 and is required when saving.
The sidebar New note button uses the `xs` size with 12px text and a 14px plus icon.

---

## 9. Checklist

Before merging UI work:

- [ ] Colors use semantic tokens or the documented content and icon exceptions.
- [ ] Radius and shadow use named defaults or the documented component exceptions.
- [ ] Every non-submit button in a form has `type="button"`.
- [ ] Colored counts correspond to named statuses such as met or breached.
- [ ] Light and dark themes match the documented surfaces.
- [ ] Headings and descriptions use the shared typography classes.
- [ ] Desktop and narrow layouts keep headings aligned with their icons and controls.
- [ ] UI copy reuses existing i18n keys, with English maintained in `i18n/en-US.json`.
- [ ] Reused an existing `ui/` primitive rather than building a variant.
