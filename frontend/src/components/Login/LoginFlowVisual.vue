<template>
  <svg
    class="login-flow-svg"
    viewBox="0 0 460 112"
    preserveAspectRatio="xMidYMid meet"
    aria-hidden="true"
    focusable="false"
  >
    <g class="flow-tracks">
      <path
        id="flow-track-a"
        d="M 4 72 C 72 18, 142 20, 214 57 S 354 104, 456 30"
      />
      <path
        id="flow-track-b"
        d="M 4 29 C 78 88, 148 94, 223 58 S 360 15, 456 74"
      />
      <path
        id="flow-track-c"
        d="M 22 101 C 100 67, 162 43, 230 51 S 344 79, 438 17"
      />
    </g>

    <g class="flow-signals">
      <path
        class="flow-signal flow-signal--violet"
        pathLength="100"
        d="M 4 72 C 72 18, 142 20, 214 57 S 354 104, 456 30"
      />
      <path
        class="flow-signal flow-signal--cyan"
        pathLength="100"
        d="M 4 29 C 78 88, 148 94, 223 58 S 360 15, 456 74"
      />
      <path
        class="flow-signal flow-signal--green"
        pathLength="100"
        d="M 22 101 C 100 67, 162 43, 230 51 S 344 79, 438 17"
      />
    </g>

    <g class="flow-nodes">
      <circle class="flow-node flow-node--cyan" cx="92" cy="56" r="3.5" />
      <rect class="flow-node flow-node--violet" x="210" y="53" width="8" height="8" rx="2" />
      <circle class="flow-node flow-node--green" cx="350" cy="67" r="3.5" />
      <circle class="flow-node flow-node--coral" cx="406" cy="37" r="2.5" />
    </g>
  </svg>
</template>

<script lang="ts" setup>
defineOptions({ name: 'LoginFlowVisual' })
</script>

<style scoped lang="scss">
.login-flow-svg {
  width: 100%;
  height: 100%;
  display: block;
  overflow: visible;
}

.flow-tracks path,
.flow-signal {
  fill: none;
  vector-effect: non-scaling-stroke;
}

.flow-tracks path {
  stroke: var(--login-flow-track, rgba(88, 101, 126, 0.16));
  stroke-width: 1;
}

.flow-signal {
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-dasharray: 11 89;
  animation: signal-travel 8s linear infinite;
}

.flow-signal--violet {
  stroke: var(--login-flow-violet, #7164ed);
}

.flow-signal--cyan {
  stroke: var(--login-flow-cyan, #2d9fb5);
  animation-delay: -3.2s;
  animation-duration: 9.5s;
}

.flow-signal--green {
  stroke: var(--login-flow-green, #3aaa7c);
  animation-delay: -5.7s;
  animation-duration: 10.5s;
}

.flow-node {
  transform-box: fill-box;
  transform-origin: center;
  animation: node-breathe 3.4s ease-in-out infinite;
}

.flow-node--violet {
  fill: var(--login-flow-violet, #7164ed);
}

.flow-node--cyan {
  fill: var(--login-flow-cyan, #2d9fb5);
  animation-delay: -1.1s;
}

.flow-node--green {
  fill: var(--login-flow-green, #3aaa7c);
  animation-delay: -2.1s;
}

.flow-node--coral {
  fill: var(--login-flow-coral, #d9796d);
  animation-delay: -2.8s;
}

@keyframes signal-travel {
  to {
    stroke-dashoffset: -100;
  }
}

@keyframes node-breathe {
  0%,
  100% {
    opacity: 0.48;
    transform: scale(0.82);
  }

  50% {
    opacity: 1;
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .flow-signal,
  .flow-node {
    animation: none;
  }
}
</style>
