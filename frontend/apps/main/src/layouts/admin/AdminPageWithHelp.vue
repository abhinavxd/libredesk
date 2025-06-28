<template>
  <div class="flex justify-between h-full">
    <div class="w-8/12">
      <slot name="content" />
    </div>
    <div class="rounded w-3/12 p-2 space-y-2 self-start">
      <slot name="help" />
    </div>
  </div>
</template>
