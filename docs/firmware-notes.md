# MC888S 固件线索

检查日期：2026-11-09。来源为用户提供的管理地址上可访问的静态 JavaScript。没有输入密码，没有执行设备配置写操作。

- 型号配置：MC888S。
- 网页版本：WEB_MC888SV1.0.0B11。网页版本不等于完整基带或设备固件版本。
- `js/config/config.js`：`WEB_ATTR_IF_SUPPORT_SHA256=2`。
- `js/service.js`：登录使用 LD；结果 0 和 4 在网页中视为成功。
- `js/util.js`：摘要为大写十六进制 SHA-256。
- 登录摘要：SHA256_UPPER(SHA256_UPPER(password) + LD)。
- 运营商名称字段：`network_provider_fullname`。
- 5G 频段字段：`Z5g_CELLINFO_band`。
- 无线设备：`station_list`；有线设备：`lan_station_list`。
- 短信：`sms_data_total`，字段 `messages`；tag=1 在网页中表示未读。
- 流量计划读字段：`data_volume_limit_switch,data_volume_limit_unit,data_volume_limit_size,data_volume_alert_percent,monthly_tx_bytes,monthly_rx_bytes,monthly_time,wan_auto_clear_flow_data_switch,traffic_clear_date`。
- `js/status/traffic_alert.js`：套餐大小为 `数值_单位因子`；MB 因子 1、GB 因子 1024、TB 因子 1048576，基数为 MiB。

首版只实现登录和只读 GoForm 请求。流量管理在本地保存计划与统计；不会调用设备的 `DATA_LIMIT_SETTING`、流量校准或计数器清零接口。

维护者随后确认 0.1.1 在该 MC888S 上连接和短信读取正常。其他字段仍需逐项核对，其他固件不自动视为兼容。
