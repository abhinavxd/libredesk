<template>
  <div class="overflow-y-auto overscroll-contain min-h-0 flex-1">
    <div class="admin-content p-4 sm:p-6 min-h-full flex flex-col">
      <div
        class="flex flex-col items-center justify-center flex-grow"
        v-if="$route.name === 'admin'"
      >
        <div>{{ $t('admin.empty') }}</div>
      </div>
      <div class="flex-grow flex flex-col">
        <router-view />
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-content {
  container-type: inline-size;
}
</style>
