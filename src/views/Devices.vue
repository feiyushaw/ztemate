<script setup lang="ts">
import { snapshot as s } from "../state";
import Icon from "../components/Icon.vue";
</script>
<template>
  <div class="page-enter">
    <section class="card">
      <div class="section-heading devices-heading">
        <div>
          <h2>接入设备</h2>
          <p>路由器报告的当前设备列表</p>
        </div>
        <span class="tag">{{ s.devices.length }} 台设备</span>
      </div>
      <p v-if="s.deviceWarning" class="inline-warning">{{ s.deviceWarning }}</p>
      <div class="table-scroll">
        <table>
          <thead>
            <tr>
              <th>设备名称</th>
              <th>IP 地址</th>
              <th>MAC 地址</th>
              <th>接入方式</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(d, i) in s.devices" :key="d.mac || `${d.ip}-${i}`">
              <td>
                <span class="device-cell"
                  ><span class="sender-icon"
                    ><Icon
                      :name="
                        d.kind === '有线' ? 'EthernetPort' : 'Laptop'
                      " /></span
                  ><strong>{{ d.name || "未命名设备" }}</strong></span
                >
              </td>
              <td class="mono">{{ d.ip || "—" }}</td>
              <td class="mono muted">{{ d.mac || "—" }}</td>
              <td>{{ d.kind || "设备未提供" }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="!s.devices.length" class="large-empty">
        <Icon name="Network" :size="38" />
        <h2>暂无设备数据</h2>
        <p>
          {{
            s.deviceWarning
              ? "当前固件的设备列表接口尚未适配。"
              : "连接路由器后，这里会显示接入设备。"
          }}
        </p>
      </div>
    </section>
    <p class="page-note">
      <Icon name="Info" :size="14" />设备速率和限速功能尚未开放。
    </p>
  </div>
</template>
