<template>
  <div class="min-h-screen flex flex-col bg-background">
    <main class="flex-1 flex items-center justify-center p-4">
      <div class="w-full max-w-[450px]">
        <slot></slot>
      </div>
    </main>
    <div class="flex justify-center mb-6">
      <a
        href="https://libredesk.io/"
        class="text-sm text-center text-muted-foreground"
        target="_blank"
        rel="noopener noreferrer"
      >
        Powered by Libredesk
      </a>
    </div>
  </div>
</template>

<script setup></script>
