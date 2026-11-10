// Package zte implements read-only GoForm access for explicitly configured routers.
package zte

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

type Client struct {
	base *url.URL
	http *http.Client
}
type Message struct {
	RemoteID string `json:"remoteId"`
	Sender   string `json:"sender"`
	Content  string `json:"content"`
	Date     string `json:"date"`
	Unread   bool   `json:"unread"`
}
type Device struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	MAC  string `json:"mac"`
	Kind string `json:"kind"`
}
type Status struct {
	MonthlySeconds *float64 `json:"monthlySeconds"`
	Model          string   `json:"model"`
	Firmware       string   `json:"firmware"`
	Carrier        string   `json:"carrier"`
	Network        string   `json:"network"`
	RSRP           *float64 `json:"rsrp"`
	RSRQ           *float64 `json:"rsrq"`
	SINR           *float64 `json:"sinr"`
	Band           string   `json:"band"`
	RX             *float64 `json:"rx"`
	TX             *float64 `json:"tx"`
	Monthly        *float64 `json:"monthly"`
	Uptime         string   `json:"uptime"`
}

func New(address string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(address))
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("请输入完整的路由器地址，例如 http://192.168.1.1")
	}
	jar, _ := cookiejar.New(nil)
	c := &Client{base: u, http: &http.Client{Timeout: 8 * time.Second, Jar: jar}}
	c.http.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	return c, nil
}
func (c *Client) Address() string { return strings.TrimRight(c.base.String(), "/") }
func HashPassword(password, ld string) string {
	a := sha256.Sum256([]byte(password))
	b := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(a[:])) + ld))
	return strings.ToUpper(hex.EncodeToString(b[:]))
}
func (c *Client) request(ctx context.Context, method, path string, values url.Values) (map[string]json.RawMessage, error) {
	u := *c.base
	u.Path = path
	var body io.Reader
	if method == http.MethodGet {
		u.RawQuery = values.Encode()
	} else {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", c.Address()+"/index.html")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接路由器，请检查网络与 macOS 本地网络权限：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, errors.New("会话失效，请重新连接")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("路由器返回 HTTP %d", resp.StatusCode)
	}
	var data map[string]json.RawMessage
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&data); err != nil || len(data) == 0 {
		return nil, errors.New("返回内容不是有效的 GoForm 数据，当前固件尚未适配")
	}
	if r := str(data, "result"); r == "failure" || r == "fail" {
		return nil, errors.New("路由器拒绝请求，请检查会话或固件兼容性")
	}
	return data, nil
}
func (c *Client) Get(ctx context.Context, cmd string, extra url.Values) (map[string]json.RawMessage, error) {
	if extra == nil {
		extra = url.Values{}
	}
	extra.Set("cmd", cmd)
	// multi_data switches GoForm to bulk NV reads. Command endpoints such
	// as sms_data_total must remain in their own single-command mode.
	if strings.Contains(cmd, ",") {
		extra.Set("multi_data", "1")
	} else {
		extra.Del("multi_data")
	}
	extra.Set("isTest", "false")
	return c.request(ctx, http.MethodGet, "/goform/goform_get_cmd_process", extra)
}
func (c *Client) Login(ctx context.Context, password string) error {
	d, e := c.Get(ctx, "LD", nil)
	if e != nil {
		return e
	}
	ld := str(d, "LD")
	if ld == "" {
		return errors.New("固件没有提供 LD 登录参数，尚未适配")
	}
	d, e = c.request(ctx, http.MethodPost, "/goform/goform_set_cmd_process", url.Values{"goformId": {"LOGIN"}, "password": {HashPassword(password, ld)}})
	if e != nil {
		return e
	}
	if result := str(d, "result"); result != "0" && result != "4" {
		return errors.New("登录失败，请检查管理密码或固件版本；应用不会自动重试密码")
	}
	return nil
}
func str(d map[string]json.RawMessage, key string) string {
	var s string
	if json.Unmarshal(d[key], &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(d[key], &n) == nil {
		return n.String()
	}
	return ""
}
func number(d map[string]json.RawMessage, key string) *float64 {
	s := str(d, key)
	v, e := strconv.ParseFloat(s, 64)
	if e != nil || s == "" {
		return nil
	}
	return &v
}
func (c *Client) Status(ctx context.Context) (Status, error) {
	d, e := c.Get(ctx, "model_name,wa_inner_version,network_provider,network_provider_fullname,network_type,Z5g_rsrp,Z5g_rsrq,Z5g_SINR,nr5g_action_band,Z5g_CELLINFO_band,realtime_rx_thrpt,realtime_tx_thrpt,monthly_rx_bytes,monthly_tx_bytes,monthly_time,realtime_time", nil)
	if e != nil {
		return Status{}, e
	}
	s := Status{Model: str(d, "model_name"), Firmware: str(d, "wa_inner_version"), Carrier: str(d, "network_provider"), Network: str(d, "network_type"), RSRP: number(d, "Z5g_rsrp"), RSRQ: number(d, "Z5g_rsrq"), SINR: number(d, "Z5g_SINR"), Band: str(d, "nr5g_action_band"), RX: number(d, "realtime_rx_thrpt"), TX: number(d, "realtime_tx_thrpt"), Uptime: str(d, "realtime_time")}
	s.MonthlySeconds = number(d, "monthly_time")
	if v := str(d, "network_provider_fullname"); v != "" {
		s.Carrier = v
	}
	if s.Band == "" {
		s.Band = str(d, "Z5g_CELLINFO_band")
	}
	rx, tx := number(d, "monthly_rx_bytes"), number(d, "monthly_tx_bytes")
	if rx != nil && tx != nil {
		v := *rx + *tx
		s.Monthly = &v
	}
	if s.Network == "" && s.Model == "" && s.RX == nil && s.RSRP == nil {
		return s, errors.New("未取得有效状态，请重新连接或检查固件兼容性")
	}
	return s, nil
}

// DecodeContent rejects malformed UTF-16 rather than silently corrupting an archive.
func DecodeContent(s string) (string, error) {
	raw, e := hex.DecodeString(s)
	if e != nil || len(raw)%2 != 0 {
		return "", errors.New("无效的 UTF-16BE Hex 短信")
	}
	units := make([]uint16, len(raw)/2)
	for i := range units {
		units[i] = uint16(raw[i*2])<<8 | uint16(raw[i*2+1])
	}
	for i := 0; i < len(units); i++ {
		v := units[i]
		if v >= 0xD800 && v <= 0xDBFF {
			if i+1 >= len(units) || units[i+1] < 0xDC00 || units[i+1] > 0xDFFF {
				return "", errors.New("无效的 UTF-16 代理对")
			}
			i++
		} else if v >= 0xDC00 && v <= 0xDFFF {
			return "", errors.New("无效的 UTF-16 代理对")
		}
	}
	return string(utf16.Decode(units)), nil
}
func (c *Client) Messages(ctx context.Context) ([]Message, error) {
	all := []Message{}
	seen := map[string]bool{}
	for page := 0; page < 100; page++ {
		d, e := c.Get(ctx, "sms_data_total", url.Values{"page": {strconv.Itoa(page)}, "data_per_page": {"50"}, "mem_store": {"1"}, "tags": {"10"}, "order_by": {"order by id desc"}})
		if e != nil {
			return nil, e
		}
		raw, ok := d["messages"]
		if !ok {
			auth, authErr := c.Get(ctx, "loginfo", nil)
			if authErr == nil {
				if _, exists := auth["loginfo"]; exists && str(auth, "loginfo") != "ok" {
					return nil, errors.New("短信访问的登录会话已失效，请在设置中重新连接；网络状态可在未登录时继续显示")
				}
			}
			keys := make([]string, 0, len(d))
			for key := range d {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			return nil, fmt.Errorf("短信响应格式不匹配（返回字段：%s），请反馈此提示以继续适配", strings.Join(keys, ", "))
		}
		var rows []map[string]json.RawMessage
		if e = json.Unmarshal(raw, &rows); e != nil {
			return nil, errors.New("短信列表格式尚未适配")
		}
		added := 0
		for _, r := range rows {
			m := Message{RemoteID: str(r, "id"), Sender: str(r, "number"), Date: str(r, "date"), Unread: str(r, "tag") == "1"}
			content := str(r, "content")
			m.Content, e = DecodeContent(content)
			if e != nil {
				return nil, fmt.Errorf("短信 %s 编码未适配：%w", m.RemoteID, e)
			}
			key := m.RemoteID + "|" + m.Sender + "|" + m.Date + "|" + content
			if !seen[key] {
				seen[key] = true
				all = append(all, m)
				added++
			}
		}
		if len(rows) < 50 {
			return all, nil
		}
		if added == 0 {
			return nil, errors.New("路由器重复返回同一页短信，同步未完成")
		}
	}
	return nil, errors.New("短信超过安全分页上限，同步未完成")
}
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	out := []Device{}
	seen := map[string]bool{}
	for _, group := range []struct{ command, kind string }{{"station_list", "Wi-Fi"}, {"lan_station_list", "有线"}} {
		d, e := c.Get(ctx, group.command, nil)
		if e != nil {
			return nil, e
		}
		raw, ok := d[group.command]
		if !ok {
			return nil, fmt.Errorf("固件未提供 %s 设备列表", group.kind)
		}
		var rows []map[string]json.RawMessage
		if json.Unmarshal(raw, &rows) != nil {
			return nil, errors.New("接入设备列表格式尚未适配")
		}
		for _, r := range rows {
			name := str(r, "hostname")
			if name == "" {
				name = str(r, "host_name")
			}
			mac, ip := str(r, "mac_addr"), str(r, "ip_addr")
			key := mac + "|" + ip
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Device{Name: name, IP: ip, MAC: mac, Kind: group.kind})
		}
	}
	return out, nil
}
