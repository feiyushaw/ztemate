<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import {
  snapshot as s,
  busy,
  native,
  importTraffic,
  saveTraffic,
  tell,
} from "../state";
import type { TrafficPlan } from "../types";
import Icon from "../components/Icon.vue";
const plan = reactive<TrafficPlan>({ ...s.value.traffic.plan });
const used = ref(Number(s.value.traffic.used.toFixed(3))),
  importing = ref(false),
  changed = ref(false);
const percent = computed(() =>
  plan.limit > 0 ? (used.value / plan.limit) * 100 : 0,
);
const remaining = computed(() => Math.max(0, plan.limit - used.value));
const threshold = computed(() => (plan.limit * plan.alertPercent) / 100);
const units: Record<string, number> = {
  MB: 1024 ** 2,
  GB: 1024 ** 3,
  TB: 1024 ** 4,
  小时: 3600,
};
watch(
  () => plan.unit,
  (unit, previous) => {
    if (unit === "小时" || previous === "小时") return;
    const ratio = (units[previous] || 1) / (units[unit] || 1);
    plan.limit = Number((plan.limit * ratio).toFixed(6));
    used.value = Number((used.value * ratio).toFixed(6));
  },
  { flush: "sync" },
);
function changeKind() {
  if (plan.kind === "time") {
    plan.unit = "小时";
    plan.limit = 720;
    used.value = Number(
      ((s.value.status.monthlySeconds || 0) / 3600).toFixed(3),
    );
  } else {
    plan.unit = "GB";
    plan.limit = 150;
    used.value = Number(((s.value.status.monthly || 0) / 1024 ** 3).toFixed(3));
  }
}
async function loadRouter() {
  importing.value = true;
  try {
    const v = await importTraffic();
    Object.assign(plan, v.plan);
    used.value = Number(v.used.toFixed(3));
    tell("路由器计划已读入表单，保存后用于本地估算");
  } catch (e) {
    tell(String(e));
  } finally {
    importing.value = false;
  }
}
async function save() {
  if (await saveTraffic({ ...plan }, used.value)) {
    changed.value = true;
    tell(
      s.value.mode === "demo"
        ? "演示计划已更新，不会修改设备"
        : "本地流量计划已保存，不会修改路由器",
    );
    setTimeout(() => (changed.value = false), 3000);
  }
}
</script>
<template>
  <div class="traffic-page page-enter">
    <section class="card traffic-summary">
      <div class="section-heading">
        <div>
          <h2>套餐使用情况</h2>
          <p>根据路由器计数器估算，非运营商账单</p>
        </div>
        <span class="tag">{{
          plan.kind === "data" ? "流量套餐" : "时间套餐"
        }}</span>
      </div>
      <div class="traffic-totals">
        <div>
          <span class="caption">当前已用</span
          ><strong class="mono"
            >{{ Number(used).toFixed(2) }} <small>{{ plan.unit }}</small
            ><span> / {{ plan.limit }} {{ plan.unit }}</span></strong
          >
        </div>
        <div class="remaining">
          <span class="caption">套餐剩余</span
          ><strong class="mono"
            >{{ remaining.toFixed(2) }} <small>{{ plan.unit }}</small></strong
          >
        </div>
      </div>
      <div class="traffic-progress">
        <div
          :style="{ width: Math.min(percent, 100) + '%' }"
          :class="{ warning: percent >= plan.alertPercent }"
        ></div>
        <i
          :style="{ left: plan.alertPercent + '%' }"
          :title="`${plan.alertPercent}% 提醒阈值`"
        ></i>
      </div>
      <div class="usage-meta">
        <span>已用 {{ percent.toFixed(1) }}%</span
        ><span
          >{{ plan.alertPercent }}% 时提醒 · {{ threshold.toFixed(2) }}
          {{ plan.unit }}</span
        >
      </div>
    </section>
    <p v-if="s.trafficWarning" class="inline-warning">{{ s.trafficWarning }}</p>
    <section class="card settings-card">
      <div class="section-heading">
        <div>
          <h2>流量计划</h2>
          <p>仅保存到这台 Mac，不修改路由器设置</p>
        </div>
        <button
          class="button subtle"
          :disabled="
            importing ||
            busy ||
            !native ||
            s.mode === 'demo' ||
            s.state !== 'online'
          "
          @click="loadRouter"
        >
          <Icon
            name="RefreshCw"
            :size="14"
            :class="{ spinning: importing }"
          />读取路由器计划
        </button>
      </div>
      <form @submit.prevent="save">
        <label class="setting-row"
          ><div>
            <strong>流量管理</strong>
            <p>开启本地用量统计与阈值提醒</p>
          </div>
          <input
            v-model="plan.enabled"
            type="checkbox"
            role="switch"
            class="switch" /></label
        ><label class="setting-row"
          ><div>
            <strong>按账期清零</strong>
            <p>仅清零本地累计值，保留短信和设备配置</p>
          </div>
          <input
            v-model="plan.autoReset"
            type="checkbox"
            role="switch"
            class="switch" /></label
        ><label class="setting-row"
          ><div>
            <strong>清零日期</strong>
            <p>不足指定日期的月份，使用当月最后一天</p>
          </div>
          <span class="number-input"
            >每月<input
              v-model.number="plan.resetDay"
              type="number"
              min="1"
              max="31"
              step="1"
              required
              :disabled="!plan.autoReset"
              aria-label="清零日期"
            />日</span
          ></label
        >
        <fieldset class="setting-row">
          <legend class="sr-only">套餐形式</legend>
          <div>
            <strong>套餐形式</strong>
            <p>选择流量或在线时长</p>
          </div>
          <div class="segmented">
            <label
              ><input
                v-model="plan.kind"
                type="radio"
                value="data"
                @change="changeKind"
              /><span>流量</span></label
            ><label
              ><input
                v-model="plan.kind"
                type="radio"
                value="time"
                @change="changeKind"
              /><span>时间</span></label
            >
          </div>
        </fieldset>
        <div class="setting-row">
          <div>
            <strong>已用{{ plan.kind === "data" ? "流量" : "时间" }}</strong>
            <p>校准本地起始用量，不更改设备计数器</p>
          </div>
          <div class="number-input">
            <input
              v-model.number="used"
              type="number"
              min="0"
              max="1000000000000"
              step="any"
              required
              aria-label="已用用量"
            /><select
              v-if="plan.kind === 'data'"
              v-model="plan.unit"
              aria-label="用量单位"
            >
              <option>MB</option>
              <option>GB</option>
              <option>TB</option></select
            ><span v-else>小时</span>
          </div>
        </div>
        <label class="setting-row"
          ><div>
            <strong>套餐额度</strong>
            <p>按照你的运营商套餐填写</p>
          </div>
          <span class="number-input"
            ><input
              v-model.number="plan.limit"
              type="number"
              min="0.001"
              max="1000000000"
              step="any"
              required
              aria-label="套餐总额度"
            />{{ plan.unit }}</span
          ></label
        ><label class="setting-row"
          ><div>
            <strong>提醒阈值</strong>
            <p>每个账期首次达到阈值时发送系统通知</p>
          </div>
          <span class="number-input"
            ><input
              v-model.number="plan.alertPercent"
              type="number"
              min="1"
              max="100"
              step="1"
              required
              aria-label="提醒百分比"
            />%</span
          ></label
        >
        <div class="traffic-save">
          <p class="caption">
            {{
              plan.enabled
                ? `达到 ${threshold.toFixed(2)} ${plan.unit} 时提醒，不自动断网。`
                : "流量管理关闭时不发送阈值提醒。"
            }}
          </p>
          <button class="button primary" :disabled="busy">
            <Icon :name="changed ? 'Check' : 'Gauge'" :size="15" />{{
              changed ? "本地计划已保存" : "保存本地计划"
            }}
          </button>
        </div>
      </form>
    </section>
    <div class="notice">
      <Icon name="Info" :size="16" /><span
        >本地提醒依赖此应用运行。账期边界离线时，恢复采集的读数作为新基线。流量单位采用
        1024 换算；准确余量请查询运营商。</span
      >
    </div>
  </div>
</template>
