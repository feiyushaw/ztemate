<script setup lang="ts">
import { snapshot as s, page } from "../state";
import Icon from "./Icon.vue";
</script>
<template>
  <section class="card usage-card">
    <div class="section-heading">
      <h2>本期{{ s.traffic.plan.kind === "time" ? "时长" : "流量" }}估算</h2>
      <Icon name="Gauge" />
    </div>
    <div class="usage-number mono">
      {{
        s.traffic.updatedAt || s.mode === "demo"
          ? s.traffic.used.toFixed(2)
          : "—"
      }}
      {{ s.traffic.plan.unit
      }}<small> / {{ s.traffic.plan.limit }} {{ s.traffic.plan.unit }}</small>
    </div>
    <div class="progress-track">
      <div
        :style="{ width: Math.min(s.traffic.percent, 100) + '%' }"
        :class="{ warning: s.traffic.percent >= s.traffic.plan.alertPercent }"
      ></div>
    </div>
    <div class="usage-meta">
      <span>{{
        s.traffic.plan.enabled
          ? `剩余 ${s.traffic.remaining.toFixed(2)} ${s.traffic.plan.unit}`
          : "本地提醒未开启"
      }}</span
      ><button class="text-button" @click="page = 'traffic'">
        流量管理 <Icon name="ChevronRight" :size="13" />
      </button>
    </div>
    <p class="caption">
      本地估算 · {{ s.traffic.plan.alertPercent }}% 提醒 · 非运营商账单
    </p>
  </section>
</template>
