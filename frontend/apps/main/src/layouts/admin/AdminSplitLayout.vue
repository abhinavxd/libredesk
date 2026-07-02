<template>
  <div class="flex justify-between h-full">
    <div class="w-full xl:w-8/12 pr-6 relative">
      <slot name="content" />
    </div>
    <div class="hidden lg:block rounded w-3/12 p-2 space-y-2 self-stretch text-sm text-foreground/70">
      <slot name="help" />
    </div>
  </div>
</template>
