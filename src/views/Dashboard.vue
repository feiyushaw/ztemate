<script setup lang="ts">
import { computed } from "vue";
import { snapshot as s, page, action, busy } from "../state";
import { speed, date, otp, uptime } from "../format";
import Icon from "../components/Icon.vue";
import SpeedChart from "../components/SpeedChart.vue";
import UsageCard from "../components/UsageCard.vue";
const signals = computed(() => [
  {
    name: "RSRP",
    value: s.value.status.rsrp,
    unit: "dBm",
    detail: "参考信号接收功率",
  },
  {
    name: "RSRQ",
    value: s.value.status.rsrq,
    unit: "dB",
    detail: "参考信号接收质量",
  },
  {
    name: "SINR",
    value: s.value.status.sinr,
    unit: "dB",
    detail: "信号与干扰加噪声比",
  },
]);
</script>
<template>
  <div class="dashboard page-enter">
    <section class="connection-strip card">
      <div class="connection-icon"><Icon name="Router" :size="26" /></div>
      <div class="grow">
        <h2>
          {{ s.settings.alias || "我的路由器" }}
          <span class="tag">{{ s.status.model || "型号待确认" }}</span>
        </h2>
        <p>
          {{ s.status.carrier || "运营商待确认"
          }}<span class="separator">·</span>{{ s.status.network || "等待连接"
          }}<span class="separator">·</span>{{ s.status.band || "频段未提供" }}
        </p>
      </div>
      <button
        class="button subtle"
        :disabled="!s.settings.address || s.mode === 'demo'"
        @click="action('OpenAdmin')"
      >
        管理后台 <Icon name="ArrowUpRight" :size="15" />
      </button>
    </section>
    <div class="metric-row">
      <section class="card metric">
        <div class="metric-label">
          <span class="small-icon blue"
            ><Icon name="ArrowDown" :size="16" /></span
          >下载速率
        </div>
        <div class="metric-number mono">{{ speed(s.status.rx) }}</div>
        <p>整个路由器的实时下载</p>
      </section>
      <section class="card metric">
        <div class="metric-label">
          <span class="small-icon teal"><Icon name="ArrowUp" :size="16" /></span
          >上传速率
        </div>
        <div class="metric-number mono">{{ speed(s.status.tx) }}</div>
        <p>整个路由器的实时上传</p>
      </section>
      <section class="card metric">
        <div class="metric-label">
          <span class="small-icon violet"
            ><Icon name="Network" :size="16" /></span
          >接入设备
        </div>
        <div class="metric-number mono">
          {{
            s.deviceWarning || s.state !== "online" ? "—" : s.devices.length
          }}
          <small>台</small>
        </div>
        <button class="text-button" @click="page = 'devices'">
          查看设备列表 <Icon name="ChevronRight" :size="13" />
        </button>
      </section>
    </div>
    <section class="card chart-card">
      <div class="section-heading">
        <div>
          <h2>网络速率</h2>
          <p>最近 60 秒 · 整个路由器</p>
        </div>
        <div class="chart-legend">
          <span><i class="blue-dot"></i>下载</span
          ><span><i class="teal-dot"></i>上传</span
          ><button
            class="icon-button"
            aria-label="刷新网络数据"
            :disabled="busy || s.mode === 'demo' || !s.lastSync"
            @click="action('Refresh')"
          >
            <Icon name="RefreshCw" :size="15" />
          </button>
        </div>
      </div>
      <SpeedChart :samples="s.history" />
    </section>
    <div class="detail-grid">
      <section class="card signal-card">
        <div class="section-heading">
          <h2>5G 信号详情</h2>
          <span class="tag">NR</span>
        </div>
        <div class="signal-grid">
          <div v-for="v in signals" :key="v.name">
            <span class="caption">{{ v.name }}</span>
            <div class="signal-number mono">
              {{ v.value ?? "—" }}
              <small v-if="v.value != null">{{ v.unit }}</small>
            </div>
            <span class="caption">{{
              v.value == null ? "设备未提供" : v.detail
            }}</span>
          </div>
        </div>
        <div class="card-foot">
          <Icon name="Activity" :size="14" /><span
            >连接时长：{{ uptime(s.status.uptime) }}</span
          >
        </div>
      </section>
      <UsageCard />
    </div>
    <section class="card latest-card">
      <div class="section-heading">
        <h2>
          最近短信 <span class="count">{{ s.unread }}</span>
        </h2>
        <button class="text-button" @click="page = 'sms'">
          全部短信 <Icon name="ChevronRight" :size="14" />
        </button>
      </div>
      <p v-if="s.smsWarning" class="inline-warning">{{ s.smsWarning }}</p>
      <button
        v-for="m in s.messages.slice(0, 2)"
        :key="m.id"
        class="message-preview"
        @click="page = 'sms'"
      >
        <span class="sender-icon"
          ><Icon name="MessageSquare" :size="19"
        /></span>
        <div class="grow">
          <div class="message-preview-heading">
            <strong>{{ m.sender }}</strong
            ><span v-if="otp(m.content)" class="tag blue-tag">验证码</span
            ><time>{{ date(m.date) }}</time>
          </div>
          <p>{{ m.content }}</p>
        </div>
        <Icon name="ChevronRight" :size="16" />
      </button>
      <div v-if="!s.messages.length" class="empty-inline">
        <Icon name="Mail" :size="22" /><span>短信同步后会显示在这里</span>
      </div>
    </section>
  </div>
</template>
