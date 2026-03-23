<script setup>
import AppSidebar from '../components/AppSidebar.vue'
</script>

<template>
  <div class="app-layout">
    <AppSidebar />
    <main class="app-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.app-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

@media (max-width: 768px) {
  .app-layout {
    flex-direction: column;
    height: 100vh;
    height: 100dvh; /* dynamic viewport height for mobile browsers */
  }
  .app-content {
    flex: 1;
    padding-bottom: 56px; /* space for bottom nav */
    min-height: 0;
  }
}
</style>
