import { computed, ref } from "vue";
import type { Settings, Snapshot, TrafficPlan, TrafficView } from "./types";
const empty = (): Snapshot => ({
  traffic: {
    plan: {
      enabled: false,
      autoReset: true,
      resetDay: 1,
      kind: "data",
      unit: "GB",
      limit: 150,
      alertPercent: 80,
    },
    used: 0,
    remaining: 150,
    percent: 0,
    cycle: "",
    updatedAt: "",
  },
  trafficWarning: "",
  mode: "live",
  state: "unconfigured",
  error: "",
  smsWarning: "",
  deviceWarning: "",
  lastSync: "",
  settings: {
    address: "",
    alias: "我的路由器",
    notifications: false,
    openAtLogin: false,
    theme: "system",
  },
  status: {
    model: "",
    firmware: "",
    carrier: "",
    network: "",
    rsrp: null,
    rsrq: null,
    sinr: null,
    band: "",
    rx: null,
    tx: null,
    monthly: null,
    uptime: "",
  },
  messages: [],
  devices: [],
  history: [],
  unread: 0,
});
export const snapshot = ref<Snapshot>(empty());
export const toast = ref("");
export const busy = ref(false);
export const native = computed(() => !!window.mygo);
export const page = ref("overview");
let toastTimer: ReturnType<typeof setTimeout>;
export function tell(message: string) {
  toast.value = message;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (toast.value = ""), 5000);
}
async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  if (!window.mygo) throw Error("浏览器仅供界面预览，请在 Mac 应用中连接设备");
  return window.mygo.call<T>("Service." + method, ...args);
}
export async function refreshSnapshot() {
  if (window.mygo) {
    snapshot.value = await call<Snapshot>("Snapshot");
    applyTheme(snapshot.value.settings.theme);
  }
}
export function applyTheme(theme: Settings["theme"]) {
  document.documentElement.dataset.theme = theme;
}
export async function action(method: string, ...args: unknown[]) {
  if (busy.value) return false;
  busy.value = true;
  try {
    await call(method, ...args);
    await refreshSnapshot();
    return true;
  } catch (e) {
    tell(String(e instanceof Error ? e.message : e));
    try {
      await refreshSnapshot();
    } catch {
      /* Keep the original action error. */
    }
    return false;
  } finally {
    busy.value = false;
  }
}
export async function saveSettings(cfg: Settings) {
  if (!window.mygo) {
    snapshot.value.settings = { ...cfg };
    applyTheme(cfg.theme);
    tell("预览设置仅在当前页面生效");
    return true;
  }
  return action("SaveSettings", cfg);
}
export async function copyCode(id: string, code: string) {
  try {
    if (window.mygo) await call("CopyCode", id);
    else await navigator.clipboard.writeText(code);
    tell("验证码已复制");
  } catch (e) {
    tell(String(e));
  }
}
export async function demo() {
  if (window.mygo) {
    await action("Demo");
    return;
  }
  const now = Date.now();
  snapshot.value = {
    ...empty(),
    traffic: {
      plan: {
        enabled: true,
        autoReset: true,
        resetDay: 1,
        kind: "data",
        unit: "GB",
        limit: 150,
        alertPercent: 80,
      },
      used: 7.15,
      remaining: 142.85,
      percent: (7.15 / 150) * 100,
      cycle: "2026-11-01",
      updatedAt: new Date().toISOString(),
    },
    mode: "demo",
    state: "online",
    lastSync: new Date().toISOString(),
    status: {
      model: "MC888S",
      firmware: "演示固件 · 非实机数据",
      carrier: "中国联通",
      network: "5G SA",
      rsrp: -87,
      rsrq: -11,
      sinr: 23,
      band: "n78",
      rx: 12.4e6,
      tx: 1.2e6,
      monthly: 142.6e9,
      uptime: "90620",
    },
    messages: [
      {
        id: "demo-1",
        remoteId: "1",
        sender: "服务通知",
        content: "您的验证码为 123456，5 分钟内有效。请勿向他人泄露。",
        date: new Date().toISOString(),
        unread: true,
        archivedAt: "",
      },
      {
        id: "demo-2",
        remoteId: "2",
        sender: "10010",
        content: "【演示短信】您本月的套餐使用情况请以运营商账单为准。",
        date: new Date(now - 36e5).toISOString(),
        unread: false,
        archivedAt: "",
      },
      {
        id: "demo-3",
        remoteId: "3",
        sender: "服务通知",
        content: "欢迎使用 ZTEMate。网络与短信，触手可及。",
        date: new Date(now - 72e5).toISOString(),
        unread: false,
        archivedAt: "",
      },
    ],
    devices: [
      {
        name: "我的 Mac",
        ip: "192.0.2.10",
        mac: "02:00:00:00:00:01",
        kind: "Wi-Fi",
      },
      {
        name: "Ubuntu",
        ip: "192.0.2.20",
        mac: "02:00:00:00:00:02",
        kind: "有线",
      },
    ],
    history: Array.from({ length: 31 }, (_, i) => ({
      at: now - (30 - i) * 2000,
      rx:
        8e6 + 4e6 * Math.sin((30 - i) * 0.45) + 2e6 * Math.cos((30 - i) * 1.2),
      tx: (8e6 + 4e6 * Math.sin((30 - i) * 0.45)) * 0.12,
    })),
    unread: 1,
  };
}
export async function leaveDemo() {
  if (window.mygo) await action("LeaveDemo");
  else snapshot.value = empty();
}

export async function importTraffic() {
  return call<TrafficView>("ImportRouterTraffic");
}
export async function saveTraffic(plan: TrafficPlan, used: number) {
  if (window.mygo) return action("SaveTraffic", plan, used);
  snapshot.value.traffic = {
    plan: { ...plan },
    used,
    remaining: Math.max(0, plan.limit - used),
    percent: (used / plan.limit) * 100,
    cycle: "预览账期",
    updatedAt: new Date().toISOString(),
  };
  tell("本地计划预览已更新；浏览器预览不持久保存");
  return true;
}
