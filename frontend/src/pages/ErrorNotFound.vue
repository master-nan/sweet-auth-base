<template>
  <div class="fullscreen bg-primary text-white text-center q-pa-md flex flex-center">
    <div>
      <div style="font-size: 30vh">404</div>

      <div class="text-h2" style="opacity: 0.4">
        哎呀，这里什么都没有...
        <!--        Oops. Nothing here...-->
      </div>

      <q-btn
        class="q-mt-xl"
        color="white"
        text-color="primary"
        unelevated
        to="/admin/home"
        label="返回"
        no-caps
      />
    </div>
  </div>
</template>

<script setup lang="ts"></script>
