# Libredesk Frontend - Multi-App Setup

This frontend supports both the main Libredesk application and a chat widget as separate Vue applications sharing common UI components.

## Project Structure

```
frontend/
├── apps/
│   ├── main/          # Main Libredesk application
│   │   ├── src/
│   │   └── index.html
│   └── widget/        # Chat widget application
│       ├── src/
│       └── index.html
├── shared-ui/         # Shared UI components (shadcn/ui)
│   ├── components/
│   │   └── ui/        # shadcn/ui components
│   ├── lib/           # Utility functions
│   └── assets/        # Shared styles
└── package.json
```

## Development

Check Makefile for available commands.

## Shared UI Components

The `shared-ui` directory contains all the shadcn/ui components that can be used in both apps.

### Using Shared Components

```vue
<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@shared-ui/components/ui/card'
import { Input } from '@shared-ui/components/ui/input'
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>Example Card</CardTitle>
    </CardHeader>
    <CardContent>
      <Input placeholder="Type something..." />
      <Button>Submit</Button>
    </CardContent>
  </Card>
</template>
```

### Path Aliases

- `@shared-ui` - Points to the shared-ui directory
- `@main` - Points to apps/main/src
- `@widget` - Points to apps/widget/src
- `@` - Points to the current app's src directory (context-dependent)
