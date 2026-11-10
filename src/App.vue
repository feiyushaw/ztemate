<script setup lang="ts">
import { computed, onMounted, onUnmounted } from "vue";
import {
  snapshot as s,
  page,
  toast,
  native,
  busy,
  refreshSnapshot,
  demo,
  leaveDemo,
  action,
  tell,
} from "./state";
import { stateLabel } from "./format";
import Icon from "./components/Icon.vue";
import Dashboard from "./views/Dashboard.vue";
import Messages from "./views/Messages.vue";
import Devices from "./views/Devices.vue";
import Settings from "./views/Settings.vue";
import Traffic from "./views/Traffic.vue";
import TrayPopover from "./views/TrayPopover.vue";
const tray = new URLSearchParams(location.search).get("view") === "tray";
const nav = [
  {
    id: "overview",
    icon: "LayoutDashboard",
    name: "总览",
    sub: "你的网络，一目了然",
  },
  {
    id: "sms",
    icon: "MessageSquare",
    name: "短信",
    sub: "消息与验证码，集中管理",
  },
  {
    id: "traffic",
    icon: "Gauge",
    name: "流量",
    sub: "套餐用量与提醒，心中有数",
  },
  {
    id: "devices",
    icon: "Laptop",
    name: "设备",
    sub: "查看连接到路由器的设备",
  },
  { id: "settings", icon: "Settings", name: "设置", sub: "连接、偏好与隐私" },
];
const current = computed(() => nav.find((n) => n.id === page.value) || nav[0]);
const view = computed(
  () =>
    ({
      overview: Dashboard,
      sms: Messages,
      devices: Devices,
      traffic: Traffic,
      settings: Settings,
    })[page.value] || Dashboard,
);
const syncText = computed(() =>
  s.value.lastSync
    ? `最近同步 ${new Date(s.value.lastSync).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" })}`
    : "等待首次同步",
);
let timer: ReturnType<typeof setInterval>;
let fetching = false;
let bridgeErrorShown = false;
async function sync() {
  if (fetching) return;
  fetching = true;
  try {
    await refreshSnapshot();
    bridgeErrorShown = false;
  } catch {
    if (!bridgeErrorShown) {
      tell("无法同步应用状态，请重新打开窗口");
      bridgeErrorShown = true;
    }
  } finally {
    fetching = false;
  }
}
onMounted(() => {
  sync();
  timer = setInterval(sync, 2000);
});
onUnmounted(() => clearInterval(timer));
</script>
<template>
  <div v-if="tray" class="tray-shell"><TrayPopover /></div>
  <div v-else class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="app-mark"><Icon name="Radio" :size="25" /></span>
        <div><strong>ZTEMate</strong><span>网络与短信，触手可及</span></div>
      </div>
      <div class="nav-caption">工作空间</div>
      <nav aria-label="主导航">
        <button
          v-for="n in nav"
          :key="n.id"
          :class="{ active: page === n.id }"
          :aria-current="page === n.id ? 'page' : undefined"
          @click="page = n.id"
        >
          <Icon :name="n.icon" :size="19" /><span>{{ n.name }}</span
          ><span v-if="n.id === 'sms' && s.unread" class="nav-count">{{
            s.unread
          }}</span>
        </button>
      </nav>
      <div class="sidebar-bottom">
        <div class="privacy-note">
          <Icon name="ShieldCheck" :size="16" /><span>数据仅存储于本机</span>
        </div>
        <button class="sidebar-device" @click="page = 'settings'">
          <span class="device-avatar"><Icon name="Router" :size="20" /></span>
          <div>
            <strong>{{ s.status.model || "连接你的路由器" }}</strong
            ><span
              ><i
                class="status-dot"
                :class="{ online: s.state === 'online' }"
              ></i
              >{{ s.mode === "demo" ? "演示模式" : stateLabel[s.state] }}</span
            >
          </div>
          <Icon name="ChevronRight" :size="14" />
        </button>
        <div class="version">ZTEMate <span>v0.1.1</span></div>
      </div>
    </aside>
    <main class="main-pane">
      <header class="page-header">
        <div>
          <div class="eyebrow">YOUR NETWORK, AT A GLANCE</div>
          <h1>{{ current.name }}</h1>
          <p>{{ current.sub }}</p>
        </div>
        <div class="header-actions">
          <span class="sync-time">{{ syncText }}</span
          ><span class="state-pill" :class="{ online: s.state === 'online' }"
            ><i></i
            >{{ s.mode === "demo" ? "演示模式" : stateLabel[s.state] }}</span
          >
        </div>
      </header>
      <div class="page-content">
        <div v-if="!native" class="notice">
          <Icon name="Info" :size="16" /><span
            >浏览器界面预览。设备连接、归档和系统通知在 Mac 应用中运行。</span
          ><button v-if="s.mode !== 'demo'" class="text-button" @click="demo">
            查看演示
          </button>
        </div>
        <div v-if="s.mode === 'demo'" class="notice demo-banner">
          <Icon name="Play" :size="15" /><span
            >正在体验演示数据，不代表实际路由器状态。</span
          ><button class="text-button" @click="leaveDemo">退出演示</button>
        </div>
        <div v-if="s.error" class="notice error-banner" role="alert">
          <Icon name="Info" :size="16" /><span
            >{{ s.error
            }}<small v-if="s.lastSync">
              · 当前保留最后一次数据，数据已过期</small
            ></span
          ><button class="text-button" @click="page = 'settings'">
            检查连接
          </button>
        </div>
        <section
          v-if="s.state === 'unconfigured' && page === 'overview'"
          class="welcome card"
        >
          <div class="welcome-illustration">
            <Icon name="Router" :size="46" /><span class="welcome-signal"
              ><Icon name="Wifi" :size="20"
            /></span>
          </div>
          <span class="eyebrow">你好，ZTEMate</span>
          <h2>你的网络，从这里开始</h2>
          <p>
            连接中兴路由器，在菜单栏查看网络状态。<br />让验证码和重要消息，随时在手边。
          </p>
          <div>
            <button class="button primary" @click="page = 'settings'">
              连接路由器 <Icon name="ArrowUpRight" :size="16" /></button
            ><button class="button subtle" @click="demo">先体验一下</button>
          </div>
          <span class="welcome-foot"
            ><Icon name="ShieldCheck" :size="14" />本地连接 · 短信归档 ·
            无云端账户</span
          >
        </section>
        <component v-else :is="view" />
      </div>
      <footer class="main-footer">
        <span><Icon name="ShieldCheck" :size="12" />本地优先，安心连接</span
        ><button
          :disabled="busy || s.mode === 'demo' || !s.lastSync"
          @click="action('Refresh')"
        >
          <Icon name="RefreshCw" :size="12" />刷新数据
        </button>
      </footer>
    </main>
  </div>
  <Transition name="toast"
    ><div v-if="toast" class="toast" role="status">
      <Icon name="Info" :size="16" />{{ toast }}
    </div></Transition
  >
</template>
