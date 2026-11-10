<script setup lang="ts">
import { computed } from "vue";
import { snapshot as s, action, copyCode, demo, page } from "../state";
import { speed, otp, stateLabel } from "../format";
import Icon from "../components/Icon.vue";
import SpeedChart from "../components/SpeedChart.vue";
const latest = computed(() => s.value.messages.find((m) => otp(m.content)));
function open() {
  page.value = "overview";
  action("OpenMain");
}
</script>
<template>
  <div class="tray-view">
    <header class="tray-header">
      <span class="app-mark small"><Icon name="Radio" :size="20" /></span
      ><strong>ZTEMate</strong><span class="grow"></span
      ><span class="state-pill" :class="{ online: s.state === 'online' }"
        ><i></i>{{ s.mode === "demo" ? "演示" : stateLabel[s.state] }}</span
      >
    </header>
    <div class="tray-device">
      <strong>{{ s.status.model || s.settings.alias }}</strong
      ><span
        >{{ s.status.carrier || "等待连接" }} ·
        {{ s.status.network || "—" }}</span
      >
    </div>
    <div v-if="s.mode === 'demo'" class="demo-note">演示模式 · 非实机数据</div>
    <div class="tray-speeds">
      <div>
        <span><Icon name="ArrowDown" :size="14" />下载</span
        ><strong class="mono">{{ speed(s.status.rx) }}</strong>
      </div>
      <div>
        <span><Icon name="ArrowUp" :size="14" />上传</span
        ><strong class="mono">{{ speed(s.status.tx) }}</strong>
      </div>
    </div>
    <SpeedChart :samples="s.history" compact />
    <section class="tray-otp">
      <span class="eyebrow">最新验证码</span>
      <div v-if="latest" class="otp-display">
        <div>
          <strong class="mono">{{ otp(latest.content) }}</strong
          ><span>{{ latest.sender }}</span>
        </div>
        <button
          class="button primary"
          @click="copyCode(latest.id, otp(latest.content))"
        >
          <Icon name="Copy" :size="14" />复制
        </button>
      </div>
      <p v-else class="empty-text">暂无验证码</p>
    </section>
    <div class="tray-info">
      <Icon name="Signal" :size="16" /><span>5G RSRP</span
      ><strong class="mono">{{
        s.status.rsrp == null ? "—" : `${s.status.rsrp} dBm`
      }}</strong>
    </div>
    <p v-if="s.error" class="inline-warning">{{ s.error }}</p>
    <button v-if="s.state === 'unconfigured'" class="text-button" @click="demo">
      体验演示模式
    </button>
    <footer class="tray-footer">
      <span>{{ s.unread }} 条未读短信</span
      ><button class="text-button" @click="open">
        打开主窗口 <Icon name="ArrowUpRight" :size="14" />
      </button>
    </footer>
  </div>
</template>
