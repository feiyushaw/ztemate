<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import {
  snapshot as s,
  page,
  busy,
  action,
  saveSettings,
  native,
  demo,
  leaveDemo,
  applyTheme,
} from "../state";
import Icon from "../components/Icon.vue";
const cfg = reactive({ ...s.value.settings }),
  password = ref(""),
  address = ref(s.value.settings.address),
  saved = ref(false);
watch(
  () => s.value.settings,
  (value) => {
    Object.assign(cfg, value);
    address.value = value.address;
  },
  { deep: true },
);
async function connect() {
  await action("Connect", address.value, password.value);
  password.value = "";
}
async function save() {
  const ok = await saveSettings({ ...cfg });
  if (!ok) return;
  saved.value = true;
  setTimeout(() => (saved.value = false), 2000);
}
</script>
<template>
  <div class="settings-page page-enter">
    <section class="card settings-card">
      <div class="section-heading">
        <div>
          <h2>路由器连接</h2>
          <p>通过局域网连接你的中兴路由器</p>
        </div>
        <Icon name="Router" />
      </div>
      <form @submit.prevent="connect">
        <label class="field"
          >设备名称<input
            v-model="cfg.alias"
            maxlength="40"
            placeholder="我的路由器" /></label
        ><label class="field"
          >管理地址<input
            v-model="address"
            type="url"
            placeholder="http://192.168.1.1"
            required
            :disabled="busy"
          /><span>使用路由器管理页的实际地址，不自动猜测网关。</span></label
        ><label class="field"
          >管理密码<input
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="请输入路由器管理密码"
            required
            :disabled="busy"
          /><span>密码仅用于当前登录，不写入配置或数据库。</span></label
        >
        <div class="form-actions">
          <button class="button primary" :disabled="busy || !native">
            <Icon
              :name="busy ? 'LoaderCircle' : 'Wifi'"
              :class="{ spinning: busy }"
              :size="16"
            />{{ busy ? "正在连接…" : "连接并登录" }}</button
          ><button
            type="button"
            class="button subtle"
            :disabled="busy || !native"
            @click="action('Disconnect')"
          >
            暂停监控
          </button>
        </div>
      </form>
    </section>
    <section class="card settings-card">
      <div class="section-heading">
        <div>
          <h2>个性化设置</h2>
          <p>让 ZTEMate 按你的习惯工作</p>
        </div>
        <Icon name="Settings" />
      </div>
      <label class="setting-row"
        ><div>
          <strong>新短信通知</strong>
          <p>仅提醒新消息，通知隐藏短信正文</p>
        </div>
        <input
          v-model="cfg.notifications"
          class="switch"
          type="checkbox"
          role="switch" /></label
      ><label class="setting-row"
        ><div>
          <strong>开机自动启动</strong>
          <p>登录 Mac 时启动 ZTEMate</p>
        </div>
        <input
          v-model="cfg.openAtLogin"
          class="switch"
          type="checkbox"
          role="switch"
      /></label>
      <div class="setting-row">
        <div>
          <strong>外观</strong>
          <p>选择界面的显示模式</p>
        </div>
        <select
          v-model="cfg.theme"
          aria-label="界面外观"
          @change="applyTheme(cfg.theme)"
        >
          <option value="system">跟随系统</option>
          <option value="light">浅色</option>
          <option value="dark">深色</option>
        </select>
      </div>
      <div class="setting-row">
        <div>
          <strong>流量计划</strong>
          <p>设置套餐额度、账期和用量提醒</p>
        </div>
        <button class="text-button" @click="page = 'traffic'">
          打开流量管理 <Icon name="ChevronRight" :size="14" />
        </button>
      </div>
      <div class="form-actions">
        <button class="button primary" :disabled="busy" @click="save">
          <Icon :name="saved ? 'Check' : 'Settings'" :size="15" />保存设置
        </button>
      </div>
    </section>
    <section class="card settings-card about-card">
      <div class="section-heading">
        <div>
          <h2>关于 ZTEMate</h2>
          <p>v0.1.1 · MyGo + Vue · Apache-2.0</p>
        </div>
        <span class="tag">本地优先</span>
      </div>
      <p class="caption">
        短信保存在当前 Mac 的应用数据目录。演示数据不写入数据库。
      </p>
      <div class="form-actions">
        <button v-if="s.mode !== 'demo'" class="button subtle" @click="demo">
          体验演示模式</button
        ><button v-else class="button subtle" @click="leaveDemo">
          退出演示模式
        </button>
      </div>
    </section>
  </div>
</template>
