export function speed(n: number | null | undefined) {
  if (n == null || !Number.isFinite(n)) return "—";
  if (n >= 1e6) return (n / 1e6).toFixed(2) + " MB/s";
  if (n >= 1e3) return (n / 1e3).toFixed(1) + " KB/s";
  return Math.round(n) + " B/s";
}
export function bytes(n: number | null | undefined) {
  return n == null ? "—" : (n / 1e9).toFixed(1) + " GB";
}
export function otp(text: string) {
  return (
    text.match(
      /(?:验证码|校验码|动态码|确认码|verification code|code|otp)[^0-9\n]{0,16}([0-9]{4,8})(?:[^0-9]|$)/i,
    )?.[1] ?? ""
  );
}
export function date(text: string) {
  if (!text) return "时间未提供";
  const d = new Date(
    text.replace(
      /^(\d{2}),(\d{2}),(\d{2}),(\d{2}),(\d{2}),(\d{2}).*/,
      "20$1-$2-$3T$4:$5:$6",
    ),
  );
  return isNaN(d.getTime())
    ? text
    : d.toLocaleString("zh-CN", {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
}
export function uptime(raw: string) {
  const n = Number(raw);
  if (!raw || !Number.isFinite(n)) return "设备未提供";
  return `${Math.floor(n / 3600)} 小时 ${Math.floor((n % 3600) / 60)} 分钟`;
}
export const stateLabel: Record<string, string> = {
  online: "在线",
  error: "连接异常",
  connecting: "连接中",
  disconnected: "未连接",
  unconfigured: "未配置",
  paused: "监控已暂停",
};
