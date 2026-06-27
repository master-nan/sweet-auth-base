<template>
  <div class="login-illustration" aria-hidden="true">
    <div class="panel panel-main">
      <div class="panel-toolbar">
        <span />
        <span />
        <span />
      </div>
      <div class="panel-grid">
        <div v-for="item in 9" :key="item" />
      </div>
    </div>
    <div class="panel panel-side">
      <q-icon name="hub" />
      <div class="side-line" />
      <div class="side-line short" />
    </div>
    <div class="status-card status-a">
      <q-icon name="rule" />
      <span>权限</span>
    </div>
    <div class="status-card status-b">
      <q-icon name="table_chart" />
      <span>元数据</span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'LoginIllustration' })
</script>

<style scoped lang="scss">
.login-illustration {
  position: relative;
  width: min(520px, 78vw);
  aspect-ratio: 1.12;
}

.panel {
  position: absolute;
  border: 1px solid rgba(25, 118, 210, 0.22);
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 24px 80px rgba(25, 118, 210, 0.18);
  backdrop-filter: blur(10px);
}

.panel-main {
  inset: 18% 13% 14% 4%;
  border-radius: 8px;
  padding: 22px;
  animation: float-main 5.2s ease-in-out infinite;
}

.panel-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;

  span {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #1976d2;
    opacity: 0.75;
  }
}

.panel-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;

  div {
    height: 54px;
    border-radius: 6px;
    background: linear-gradient(135deg, rgba(25, 118, 210, 0.18), rgba(38, 166, 154, 0.2));
    animation: tile-glow 3.6s ease-in-out infinite;
  }

  div:nth-child(2n) {
    animation-delay: 0.35s;
  }

  div:nth-child(3n) {
    animation-delay: 0.7s;
  }
}

.panel-side {
  right: 2%;
  top: 8%;
  width: 150px;
  min-height: 170px;
  border-radius: 8px;
  padding: 22px;
  color: #1976d2;
  animation: float-side 4.6s ease-in-out infinite;

  .q-icon {
    font-size: 42px;
    margin-bottom: 28px;
  }
}

.side-line {
  height: 10px;
  border-radius: 99px;
  background: rgba(25, 118, 210, 0.2);
  margin-bottom: 12px;
}

.side-line.short {
  width: 62%;
  background: rgba(38, 166, 154, 0.25);
}

.status-card {
  position: absolute;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 118px;
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 16px 42px rgba(15, 23, 42, 0.12);
  color: #334155;
  font-weight: 600;
}

.status-card .q-icon {
  color: #26a69a;
  font-size: 20px;
}

.status-a {
  left: 0;
  top: 20%;
}

.status-b {
  right: 10%;
  bottom: 9%;
}

@keyframes float-main {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-12px);
  }
}

@keyframes float-side {
  0%,
  100% {
    transform: translateY(0) rotate(1deg);
  }
  50% {
    transform: translateY(10px) rotate(-1deg);
  }
}

@keyframes tile-glow {
  0%,
  100% {
    opacity: 0.72;
  }
  50% {
    opacity: 1;
  }
}

</style>
