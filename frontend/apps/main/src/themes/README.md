# Themes

Drop-in directory for custom UI themes. Auto-discovered by `useTheme.js` via
Vite glob — adding a theme requires **no edits to upstream files**.

## Convention

Each theme lives in its own subdirectory:

```
themes/
  <name>/
    theme.js      # required — exports the theme descriptor
    theme.scss    # optional — your stylesheet
```

`theme.js` exports a default object:

```js
export default {
  id: 'fresh',     // matches the [data-theme="..."] selector
  label: 'Fresh'   // shown in the theme switcher dropdown
}
```

`theme.scss` (or `.css`) opts in via the attribute selector:

```scss
[data-theme="fresh"] {
  --primary: 173 80% 40%;
  --sidebar-background: 173 30% 95%;
  // ...override any CSS variables or write theme-scoped rules
}
```

The default theme uses no `data-theme` attribute, so the bare upstream UI is
unchanged when no themes are installed.

## How the switcher behaves

- 1 theme registered (default only) → switcher UI is hidden
- 2+ themes registered → palette icon appears in the sidebar

## Limitations

This mechanism is for **visual theming via CSS only**. Replacing Vue
components (custom layouts, different message bubble, etc.) is out of scope
and would need a separate component-override system.
