# Localization

## For translators

> [!CAUTION]
> Translations for libredesk are managed through [Crowdin](
> https://crowdin.com/project/libredesk). Please use it to translate the app.
> Do not submit a pull request to update translation files unless you have been
> asked to do so.

## For developers

### How it works

The Go backend uses [leonelquinteros/gotext](
https://github.com/leonelquinteros/gotext). The Vue frontends (interface
and chat widget) use [Lingui](https://github.com/lingui/js-lingui) with the
`@lingui/format-po-gettext` formatter.

English source messages are the message IDs: there is no English catalog.
Unsupported languages and missing translations fall back to the English source.

There are three domains:

| Source file | Translation file | Usage |
| --- | --- | --- |
| `locales/backend.pot` | `locales/<language>/LC_MESSAGES/backend.po` | Messages used by the Go backend. |
| `locales/frontend.pot` | `locales/<language>/LC_MESSAGES/frontend.po` | Messages used by the Vue frontends (interface and chat widget). |
| `locales/templates.pot` | `locales/<language>/LC_MESSAGES/templates.po` | Messages used by server-rendered HTML templates (help center and CSAT pages). |

`make i18n-compile` generates `locales/<language>/LC_MESSAGES/frontend.json`.
The Go backend embeds and serves these compiled messages to both the libredesk
frontend and chat widget.

### Writing translatable messages

#### Backend and Go-rendered HTML templates

Keep message IDs and domain names as string literals. Use Go printf arguments,
not named brace placeholders or concatenated sentences:

```go
i18n.Get("Could not load %s", name)
i18n.Get("%[1]s assigned the conversation to %[2]s", actor, assignee)
i18n.GetN("Attachment", "Attachments", len(attachments))
locale.GetND("templates", "%d result", "%d results", count, count)
```

The count selects the plural form: pass it again as a formatting argument when
the message contains `%d`. Explicit argument indexes allow translators to
reorder arguments. In Go HTML templates, always select the template domain:

```gotemplate
{{ .L.GetD "templates" "Search" }}
{{ .L.GetND "templates" "%d result" "%d results" .Total .Total }}
```

#### Frontend

Write English messages directly in templates:

```vue
<button>{{ i18n._('Save') }}</button>
<p>{{ i18n._('Hello {name}', { name }) }}</p>
```

In JavaScript, use the same syntax with `useI18n()` from `@shared-ui/i18n`:

```js
const i18n = useI18n()
i18n._('Hello {name}', { name })
```

Write plurals directly in templates with `ngettext`, and use `pgettext` when
the same English text has different meanings:

```vue
<template>
  <p>{{ i18n.ngettext('{count} file', '{count} files', fileCount) }}</p>
  <button>{{ i18n.pgettext('Button label', 'Open') }}</button>
</template>
```

Declare messages containing literal `{{…}}` in the script. Vue treats these
delimiters as template expressions. Define the message with `defineMessage` in
the script, then translate the resulting descriptor in the template:

```vue
<script setup>
import { defineMessage } from '@lingui/core/macro'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()
const hint = defineMessage({
  message: "Use '{{name}}' to insert the contact's name."
})
</script>

<template>
  <p>{{ i18n._(hint) }}</p>
</template>
```

> [!TIP]
> The ICU single quotes around `'{{name}}'` escape the braces, so the rendered
> message contains the literal text `{{name}}`.

Use `TranslatedText` with named slots to insert links or other Vue nodes into a
translated message. Do not render HTML from a translation with `v-html`:

```vue
<script setup>
import { defineMessage } from '@lingui/core/macro'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()
const helpMessage = defineMessage({
  message: 'Need help? Read {documentation}.'
})
</script>

<template>
  <p>
    <TranslatedText :message="helpMessage">
      <template #documentation>
        <a href="https://docs.libredesk.io/">{{ i18n._('the documentation') }}</a>
      </template>
    </TranslatedText>
  </p>
</template>
```

> [!TIP]
> The `{documentation}` placeholder is replaced by the Vue content in the
> `documentation` slot. Translators can move the placeholder within the
> sentence. The link itself remains defined in the component. `TranslatedText`
> is registered globally by the application's translation plugin.

Translate navigation labels when rendering them. For routes, menus and other
static configuration, use Lingui's `defineMessage` macro to declare the English
text alongside the item. It creates a message descriptor that can be translated
later using the currently selected language:

```js
import { defineMessage } from '@lingui/core/macro'

const item = {
  routeName: 'reports-overview',
  titleMessage: defineMessage({ message: 'Overview' })
}
```

Render the label with `i18n._(item.titleMessage)`. Keep identifiers such as
`routeName` unchanged: they identify application features, not text to
translate.

### Best practices for writing messages

Follow these guidelines when writing messages:

- **Reuse messages that have the same meaning.** No context is needed just
  because a message appears in different handlers or components.

- **Add a context when the same English text needs different translations.**
  This can happen when a word has different meanings, serves as both a verb
  and a noun, or needs different grammatical agreement depending on the object
  it describes. For example, "Link" can mean an action or a hyperlink, and
  "New" may be translated differently for a team and a conversation.
  Use a short English description of the meaning, rather than a component name
  or file path:

  Backend (`GetC` takes the message before the context):

  ```go
  i18n.GetC("Link", "Link an article action")
  i18n.GetC("Link", "Hyperlink")
  ```

  Frontend (`pgettext` takes the context before the message):

  ```js
  i18n.pgettext('Link an article action', 'Link')
  i18n.pgettext('Hyperlink', 'Link')
  ```

- **Include punctuation inside the translatable message.** Do not append periods,
  colons or other punctuation after translation. Translators need to be able to
  adapt punctuation and spacing to their language.

  Backend:

  ```go
  // Correct
  i18n.Get("Name:")
  // Incorrect
  i18n.Get("Name") + ":"
  ```

  Frontend:

  ```vue
  <!-- Correct -->
  <label>{{ i18n._('Name:') }}</label>
  <!-- Incorrect -->
  <label>{{ i18n._('Name') }}:</label>
  ```

- **Translate complete sentences and use placeholders for variables.**
  Translators need to be able to change word order. Use indexed printf
  arguments in Go and named variables in the frontend, rather than joining
  translated fragments with names or other values.

  Backend:

  ```go
  i18n.Get("%[1]s assigned the conversation to %[2]s",
      actor,
      assignee)
  ```

  Frontend:

  ```js
  i18n._('{actor} assigned the conversation to {assignee}', { actor, assignee })
  ```

- **Pass plural counts and other values separately.** In Go, the third argument
  selects the plural form. Pass the count again among the formatting arguments
  to include it in the message. `GetNC` also accepts a context before those
  formatting arguments:

  Backend:

  ```go
  i18n.GetNC("%[1]d file uploaded by %[2]s", "%[1]d files uploaded by %[2]s",
      fileCount,
      "Upload summary",
      fileCount,
      uploaderName)
  ```

  In the frontend, the third argument to `ngettext` both selects the plural
  form and supplies `{count}`. The optional fourth argument supplies other
  variables, and the fifth supplies a context:

  Frontend:

  ```js
  i18n.ngettext(
    '{count} file uploaded by {name}',
    '{count} files uploaded by {name}',
    fileCount,
    { name: uploaderName },
    'Upload summary'
  )
  ```

### Maintaining translation catalogs

> [!CAUTION]
> Never commit any `.po`, `.mo` or `frontend.json` files to the repository.
> Translation updates in Git must come from [Crowdin](
> https://crowdin.com/project/libredesk) exports. Edit translations in Crowdin.

#### Updating source messages

Install only the tools needed for the part you are working on:

- Backend: Go.
- Frontend: Node.js 24 or newer and pnpm. Run `make frontend-build` from the
  repository root if you have not already done so to install dependencies.

Then, from the repository root:

```sh
make i18n-extract-backend    # Update backend.pot and templates.pot.
make i18n-extract-frontend   # Update frontend.pot.
make i18n-extract            # Update all .pot files.
```

Commit **only** the updated `.pot` files.

#### Local development and validation

Compiled `.mo` and frontend JSON files are generated locally and ignored by Git.
Run the compilation target for your part before its verification target.

You can run these commands from the repository root. Each target without a
suffix runs both its `-backend` and `-frontend` targets:

| Command | Backend target | Frontend target |
| --- | --- | --- |
| `make i18n-extract` | `i18n-extract-backend`: extract Go and HTML messages into `backend.pot` and `templates.pot`. | `i18n-extract-frontend`: extract Vue/JS messages into `frontend.pot`. |
| `make i18n-update` | `i18n-update-backend`: extract and merge the two backend domains into their local `.po` files. | `i18n-update-frontend`: extract and merge the frontend domain into its local `.po` files. |
| `make i18n-compile` | `i18n-compile-backend`: validate and compile missing or outdated `.mo` files. | `i18n-compile-frontend`: install frontend dependencies and compile JSON catalogs, including the English source fallback. |
| `make i18n-check` | `i18n-check-backend`: verify source coverage, `.po` validity and generated `.mo` files. | `i18n-check-frontend`: verify source coverage, `.po` validity and generated JSON files. |

> [!NOTE]
> The `i18n-update`, `i18n-check` and `i18n-compile` targets require tools from
> [GNU gettext](https://www.gnu.org/software/gettext/): `msgmerge`, `msgattrib`,
> `msgfmt` and `msgcmp`. Install the `gettext`
> package using your favorite package manager on Linux and macOS, or download
> pre-built binaries for Windows [here](
> https://mlocati.github.io/articles/gettext-iconv-windows.html).
