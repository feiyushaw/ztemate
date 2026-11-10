export interface Settings {
  address: string;
  alias: string;
  notifications: boolean;
  openAtLogin: boolean;
  theme: "system" | "light" | "dark";
}
export interface Message {
  id: string;
  remoteId: string;
  sender: string;
  content: string;
  date: string;
  unread: boolean;
  archivedAt: string;
}
export interface Device {
  name: string;
  ip: string;
  mac: string;
  kind: string;
}
export interface Sample {
  at: number;
  rx: number | null;
  tx: number | null;
}
export interface Snapshot {
  traffic: TrafficView;
  trafficWarning: string;
  mode: string;
  state: string;
  error: string;
  smsWarning: string;
  deviceWarning: string;
  lastSync: string;
  settings: Settings;
  status: {
    model: string;
    firmware: string;
    carrier: string;
    network: string;
    rsrp: number | null;
    rsrq: number | null;
    sinr: number | null;
    band: string;
    rx: number | null;
    tx: number | null;
    monthly: number | null;
    monthlySeconds?: number | null;
    uptime: string;
  };
  messages: Message[];
  devices: Device[];
  history: Sample[];
  unread: number;
}

export interface TrafficPlan {
  enabled: boolean;
  autoReset: boolean;
  resetDay: number;
  kind: "data" | "time";
  unit: string;
  limit: number;
  alertPercent: number;
}
export interface TrafficView {
  plan: TrafficPlan;
  used: number;
  remaining: number;
  percent: number;
  cycle: string;
  updatedAt: string;
}
