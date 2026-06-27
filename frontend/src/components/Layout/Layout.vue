<template>
  <router-view v-slot="{ Component }">
    <keep-alive :max="20" :include="keepAliveStore.getKeepAliveList">
      <component :is="Component" />
    </keep-alive>
  </router-view>
</template>

<script lang="ts" setup>
defineOptions({ name: 'MyLayout' })
import { useKeepAliveStore } from 'src/stores/keep-alive'

const keepAliveStore = useKeepAliveStore()
</script>
