<script setup lang="ts">
import { computed } from "vue";
import type { Sample } from "../types";
const props = withDefaults(
  defineProps<{ samples: Sample[]; compact?: boolean }>(),
  { compact: false },
);
const w = 680,
  h = 160;
const max = computed(
  () =>
    Math.max(1e6, ...props.samples.flatMap((s) => [s.rx ?? 0, s.tx ?? 0])) *
    1.15,
);
function path(key: "rx" | "tx") {
  const last = props.samples.at(-1)?.at ?? Date.now();
  let d = "",
    pen = false,
    previous = 0;
  for (const s of props.samples) {
    const n = s[key];
    if (n == null || !Number.isFinite(n)) {
      pen = false;
      continue;
    }
    const x = ((s.at - last + 60000) / 60000) * w,
      y = h - (n / max.value) * (h - 12);
    if (x < 0) continue;
    d += `${pen && s.at - previous <= 35000 ? "L" : "M"}${x.toFixed(2)},${y.toFixed(2)} `;
    pen = true;
    previous = s.at;
  }
  return d;
}
</script>
<template>
  <div
    class="chart"
    :class="{ compact }"
    role="img"
    aria-label="最近 60 秒上传和下载速率，断线处留空"
  >
    <div v-if="!compact" class="chart-scale">
      <span>{{ (max / 1e6).toFixed(0) }} MB/s</span
      ><span>{{ (max / 2e6).toFixed(0) }}</span
      ><span>0</span>
    </div>
    <svg :viewBox="`0 0 ${w} ${h}`" preserveAspectRatio="none">
      <path
        v-for="y in [h / 4, h / 2, h * 0.75, h - 1]"
        :key="y"
        :d="`M0 ${y} H${w}`"
        class="grid-line"
      />
      <path :d="path('rx')" class="rx-line" />
      <path :d="path('tx')" class="tx-line" />
    </svg>
    <div v-if="!samples.length" class="chart-empty">
      连接路由器后显示实时曲线
    </div>
    <div v-if="!compact" class="chart-time">
      <span>60 秒前</span><span>30 秒前</span><span>现在</span>
    </div>
  </div>
</template>
