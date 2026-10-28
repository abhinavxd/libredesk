# libredesk Design System

Update this guide only for verified shared styles and component usage. Keep screen decisions in PRs or audit notes.

Styling and component usage for the agent dashboard and shared widget UI.

## Sources

Read these files before changing shared styles:

- [Theme tokens and typography](shared-ui/assets/styles/main.scss)
- [Tailwind mappings](tailwind.config.cjs)
- [UI primitives](shared-ui/components/ui/)

Keep token values in their source files. Use the classes and components below when building UI.

## Colors and surfaces

Use semantic theme tokens for dashboard controls. The shared stylesheet provides light and dark values.
Pair background tokens with their foreground tokens, such as `bg-primary text-primary-foreground`.
Opacity modifiers such as `bg-success/10` work with these classes.

| Token | Use |
| --- | --- |
| `canvas` | Gutters behind content panels |
| `background` / `foreground` | Main surfaces and primary text |
| `card` / `card-foreground` | Cards and raised panels |
| `popover` / `popover-foreground` | Menus, popovers, and dropdowns |
| `primary` / `primary-foreground` | Primary actions and active states |
| `secondary` / `secondary-foreground` | Secondary controls and surfaces |
| `muted` / `muted-foreground` | Subtle surfaces, descriptions, and metadata |
| `accent` / `accent-foreground` | Hover and selected surfaces |
| `foreground-lighter` | Idle sidebar navigation text |
| `destructive` / `destructive-foreground` | Errors, removal, and breached targets |
| `success` / `success-foreground` | Successful, verified, or connected states |
| `warning` / `warning-foreground` | Warning backgrounds |
| `warning-600` | Warning text and icons |
| `link` | Links inside rendered content |
| `border` | Borders and dividers |
| `input` | Field borders |
| `ring` | Keyboard focus indicators |
| `private` | Private content tint |

Use `warning-600` for readable warning text in light mode.
Use `foreground` or `muted-foreground` for neutral counts and timestamps.
Pair small status colors with text, an icon, or a shape.

Use `sidebar-*` tokens for navigation surfaces and states.
Charts inherit `primary`, `success`, and `muted-foreground` through the shared Unovis variables.

Keep customer branding within customer surfaces and their previews.
Preserve authored colors in rendered email content. Use `.email-light-canvas` for a light email surface.

## Typography

Geist is the shared font. Use typography classes for these roles:

| Role | Class | Size | Weight | Line height |
| --- | --- | --- | --- | --- |
| Page, card, sidebar, and form section headings | `ui-heading` | 16px | 600 | 24px |
| Nested headings and compact item titles | `ui-subheading` | 14px | 600 | 20px |
| Descriptions and form hints | `ui-description` | 14px | 400 | 20px |
| Dialog and sheet titles | `ui-dialog-heading` | 18px | 600 | 28px |
| Large introductory headings | `ui-display-heading` | 24px | 600 | 32px |

Keep heading levels semantic when applying these classes.
`CardTitle`, `DialogTitle`, `SheetTitle`, and shared description components provide their default styles.
Use `tabular-nums` for metrics and numeric columns. Use 12px text for compact metadata.

## Spacing and layout

Use a 4px spacing rhythm. Common group spacing is 16px, 24px, or 32px.
Leave 16px between section headings and controls, with 24px or 32px between form sections.
Keep icon and value pairs together, with larger gaps between separate groups.

Use [FormGrid](shared-ui/components/FormGrid.vue) for responsive form fields.
Pass `compact` for narrower fields. Use container queries when columns need stable widths.
Keep short fields compact and preserve room for names, URLs, and translated labels.
Group conditional controls with their related fields.

Use [AdminSplitLayout](apps/main/src/layouts/admin/AdminSplitLayout.vue) for forms with help or a preview.
Put the form in `#content` and supporting content in `#help`.
Pass `fixedHelp` for the preview layout. The component manages responsive columns.

Use [AdminToolbar](apps/main/src/layouts/admin/AdminToolbar.vue) for list controls.
Put tabs in `#left` and actions in `#actions`.
Pass `searchable` and bind `v-model:search` for search controls.

## Radius and elevation

Use named Tailwind classes. The shared `--radius` value defines the control radius scale.

| Radius class | Use |
| --- | --- |
| `rounded-sm` | Small insets |
| `rounded-md` | Controls, chips, and badges |
| `rounded-lg` | Cards, containers, and dialogs |
| `rounded-xl` | Large surfaces |
| `rounded-2xl` | Widget frames and bubbles |
| `rounded-full` | Avatars, dots, and pills |

| Shadow class | Use |
| --- | --- |
| `shadow-sm` | Cards and standard controls |
| `shadow-md` | Popovers and floating controls |
| `shadow-lg` | Dialogs, modals, and widget windows |

## Buttons and indicators

Use the shared `Button` component for styled actions.
Reserve `default` for primary actions. Use `outline`, `secondary`, or `ghost` for supporting actions.
Use `destructive` for removal and `link` for text actions.

| Button size | Dimensions | Use |
| --- | --- | --- |
| `default` | 36px high | Standard controls |
| `sm` | 32px high | Dense controls with 12px text |
| `xs` | 28px high | Compact controls |
| `lg` | 40px high | Prominent actions |
| `icon` | 36px square | Icon actions |

Standard button icons use `size-4`. Keep icons within a group consistent in size, stroke, and color.
Give icon actions an accessible label and hide decorative icons with `aria-hidden="true"`.
Inside forms, give every supporting button `type="button"` and submit buttons `type="submit"`.

Use `Badge` variants: `default`, `secondary`, `destructive`, `success`, or `outline`.
Use `AlertDialogAction` with `variant="destructive"` for removal confirmations.
Use the shared `Tooltip` for its inverted surface, 12px text, and `rounded-md` corners.

## Fields and interaction states

Use shared `Input`, `Select`, combobox, and `TimeField` components to inherit control styles.
Standard fields use 36px height, with 16px text below `sm` and 14px above it.
Use visible labels. Placeholders provide examples or format guidance.
Use `text-muted-foreground` for empty prompts and 12px left padding for select and combobox text.

Keep focus rings visible. Reveal hover actions on keyboard focus and keep touch actions visible.
Use the existing `can-hover` variant for styles requiring a mouse or fine pointer.
Use shared component transitions and preserve their reduced motion behavior.

Use `LoadingOverlay` for page loading and `Spinner` for inline feedback.
Pass `reserve-height` when a page needs space for a centered loading indicator.

## Shared utilities

| Utility | Style |
| --- | --- |
| `.box` | Border, small shadow, and large radius |
| `.sidebar-section-label` | 12px muted uppercase text with wider tracking |
| `.link-style` | Muted underlined text with foreground hover color |

Use these utilities when their shared style fits the control.

## Admin tables

Show timestamps as `28 Aug 2026, 3:46 AM`. Keep seconds and the timezone in the full timestamp on hover.

Align headers and cells left. Let data columns share available width naturally, keep statuses and dates on one line, and keep action columns compact.
Wrap long text to three lines and expose the full value through its title. Keep horizontal overflow inside the table.
Measure rendered rows so wrapping and resizing preserve virtual scrolling.
Use compact sorting controls without hover backgrounds. Show unsorted arrows on hover or keyboard focus, and keep the active sort direction visible.
