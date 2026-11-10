package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type TrafficPlan struct {
	Enabled      bool    `json:"enabled"`
	AutoReset    bool    `json:"autoReset"`
	ResetDay     int     `json:"resetDay"`
	Kind         string  `json:"kind"`
	Unit         string  `json:"unit"`
	Limit        float64 `json:"limit"`
	AlertPercent float64 `json:"alertPercent"`
}
type TrafficLedger struct {
	Calibrated  bool        `json:"calibrated"`
	Plan        TrafficPlan `json:"plan"`
	Address     string      `json:"address"`
	UsedData    float64     `json:"usedData"`
	UsedSeconds float64     `json:"usedSeconds"`
	LastData    *float64    `json:"lastData"`
	LastSeconds *float64    `json:"lastSeconds"`
	Cycle       string      `json:"cycle"`
	Alerted     bool        `json:"alerted"`
	UpdatedAt   string      `json:"updatedAt"`
}
type TrafficView struct {
	Plan      TrafficPlan `json:"plan"`
	Used      float64     `json:"used"`
	Remaining float64     `json:"remaining"`
	Percent   float64     `json:"percent"`
	Cycle     string      `json:"cycle"`
	UpdatedAt string      `json:"updatedAt"`
}

func defaultPlan() TrafficPlan {
	return TrafficPlan{ResetDay: 1, Kind: "data", Unit: "GB", Limit: 150, AlertPercent: 80}
}
func multiplier(unit string) float64 {
	switch unit {
	case "MB":
		return 1024 * 1024
	case "GB":
		return 1024 * 1024 * 1024
	case "TB":
		return 1024 * 1024 * 1024 * 1024
	default:
		return 3600
	}
}
func cycleAt(now time.Time, day int) string {
	year, month, _ := now.Date()
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, now.Location()).Day()
	d := min(day, last)
	if now.Day() < d {
		month--
		last = time.Date(year, month+1, 0, 0, 0, 0, 0, now.Location()).Day()
		d = min(day, last)
	}
	return time.Date(year, month, d, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
}
func (l *TrafficLedger) view() TrafficView {
	used := l.UsedData
	if l.Plan.Kind == "time" {
		used = l.UsedSeconds
	}
	used /= multiplier(l.Plan.Unit)
	percent := 0.0
	if l.Plan.Limit > 0 {
		percent = used / l.Plan.Limit * 100
	}
	return TrafficView{Plan: l.Plan, Used: used, Remaining: math.Max(0, l.Plan.Limit-used), Percent: percent, Cycle: l.Cycle, UpdatedAt: l.UpdatedAt}
}

// advance preserves accumulated use if the router resets its monthly counter.
// If a billing boundary was missed offline, the first resumed sample becomes
// the new baseline; the app cannot reconstruct the exact offline allocation.
func (l *TrafficLedger) advance(now time.Time, data, seconds *float64) bool {
	cycle := cycleAt(now, l.Plan.ResetDay)
	rollover := l.Plan.AutoReset && l.Cycle != "" && cycle != l.Cycle
	if rollover {
		l.UsedData = 0
		l.UsedSeconds = 0
		l.Alerted = false
		l.LastData = copyNumber(data)
		l.LastSeconds = copyNumber(seconds)
	}
	if l.LastData != nil && data != nil {
		delta := *data - *l.LastData
		if delta < 0 {
			delta = *data
		}
		l.UsedData += delta
	} else if data != nil && !rollover && !l.Calibrated {
		l.UsedData = *data
	}
	if l.LastSeconds != nil && seconds != nil {
		delta := *seconds - *l.LastSeconds
		if delta < 0 {
			delta = *seconds
		}
		l.UsedSeconds += delta
	} else if seconds != nil && !rollover && !l.Calibrated {
		l.UsedSeconds = *seconds
	}
	if data != nil {
		l.LastData = copyNumber(data)
	}
	if seconds != nil {
		l.LastSeconds = copyNumber(seconds)
	}
	l.Cycle = cycle
	l.UpdatedAt = now.Format(time.RFC3339)
	reached := l.Plan.Enabled && l.Plan.Limit > 0 && l.view().Percent >= l.Plan.AlertPercent && !l.Alerted
	if reached {
		l.Alerted = true
	}
	return reached
}
func validatePlan(p TrafficPlan, used float64) error {
	if p.ResetDay < 1 || p.ResetDay > 31 {
		return fmt.Errorf("清零日期必须在 1–31 日之间")
	}
	if p.Kind != "data" && p.Kind != "time" {
		return fmt.Errorf("套餐形式无效")
	}
	if (p.Kind == "data" && p.Unit != "MB" && p.Unit != "GB" && p.Unit != "TB") || (p.Kind == "time" && p.Unit != "小时") {
		return fmt.Errorf("套餐单位无效")
	}
	if p.Limit <= 0 || p.Limit > 1e9 || used < 0 || used > 1e12 || math.IsNaN(p.Limit) || math.IsNaN(used) || p.AlertPercent < 1 || p.AlertPercent > 100 {
		return fmt.Errorf("请填写有效的用量、套餐额度和提醒阈值（1–100%%）")
	}
	return nil
}
func (s *Service) SaveTraffic(p TrafficPlan, used float64) error {
	if e := validatePlan(p, used); e != nil {
		return e
	}
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	l := TrafficLedger{Calibrated: true, Plan: p, Address: s.snap.Settings.Address, LastData: s.snap.Status.Monthly, LastSeconds: s.snap.Status.MonthlySeconds, Cycle: cycleAt(time.Now(), p.ResetDay), UpdatedAt: time.Now().Format(time.RFC3339)}
	if p.Kind == "data" {
		l.UsedData = used * multiplier(p.Unit)
	} else {
		l.UsedSeconds = used * 3600
	}
	if s.snap.Mode != "demo" {
		if e := s.db.SaveDocument("traffic", l); e != nil {
			return e
		}
	}
	s.traffic = l
	s.snap.Traffic = l.view()
	return nil
}
func (s *Service) ImportRouterTraffic() (TrafficView, error) {
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	c := s.client
	s.mu.Unlock()
	if c == nil {
		return TrafficView{}, fmt.Errorf("请先连接实际路由器")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	d, e := c.Get(ctx, "data_volume_limit_switch,data_volume_limit_unit,data_volume_limit_size,data_volume_alert_percent,monthly_tx_bytes,monthly_rx_bytes,monthly_time,wan_auto_clear_flow_data_switch,traffic_clear_date", nil)
	if e != nil {
		return TrafficView{}, e
	}
	value := func(k string) string { var v string; _ = json.Unmarshal(d[k], &v); return v }
	num := func(k string) float64 { v, _ := strconv.ParseFloat(value(k), 64); return v }
	if value("data_volume_limit_unit") == "" {
		return TrafficView{}, fmt.Errorf("当前固件未提供流量计划")
	}
	p := defaultPlan()
	p.Enabled = value("data_volume_limit_switch") == "1"
	p.AutoReset = value("wan_auto_clear_flow_data_switch") == "1"
	p.ResetDay = int(num("traffic_clear_date"))
	p.AlertPercent = num("data_volume_alert_percent")
	p.Kind = value("data_volume_limit_unit")
	used := 0.0
	if p.Kind == "data" {
		parts := strings.Split(value("data_volume_limit_size"), "_")
		if len(parts) != 2 {
			return TrafficView{}, fmt.Errorf("套餐额度格式尚未适配")
		}
		size, _ := strconv.ParseFloat(parts[0], 64)
		factor, _ := strconv.ParseFloat(parts[1], 64)
		p.Unit = "GB"
		p.Limit = size * factor / 1024
		used = (num("monthly_rx_bytes") + num("monthly_tx_bytes")) / multiplier("GB")
	} else {
		p.Kind = "time"
		p.Unit = "小时"
		p.Limit = num("data_volume_limit_size")
		used = num("monthly_time") / 3600
	}
	if e = validatePlan(p, used); e != nil {
		return TrafficView{}, fmt.Errorf("路由器计划不完整，请手动配置：%w", e)
	}
	return TrafficView{Plan: p, Used: used, Remaining: math.Max(0, p.Limit-used), Percent: used / p.Limit * 100}, nil
}
func (s *Service) updateTrafficLocked(now time.Time) {
	if s.traffic.Address == "" {
		s.traffic.Address = s.snap.Settings.Address
	}
	if s.traffic.Address != s.snap.Settings.Address {
		s.traffic = TrafficLedger{Plan: defaultPlan(), Address: s.snap.Settings.Address}
	}
	alert := s.traffic.advance(now, s.snap.Status.Monthly, s.snap.Status.MonthlySeconds)
	s.snap.Traffic = s.traffic.view()
	if e := s.db.SaveDocument("traffic", s.traffic); e != nil {
		s.snap.TrafficWarning = "无法保存流量统计：" + e.Error()
	}
	if alert && s.hooks.Notify != nil {
		body := fmt.Sprintf("套餐使用达到 %.1f%%。当前为本地估算，请以运营商账单为准。", s.snap.Traffic.Percent)
		go s.hooks.Notify(body)
	}
}

func copyNumber(v *float64) *float64 {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}
