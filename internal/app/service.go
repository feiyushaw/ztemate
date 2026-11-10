package app

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/feiyushaw/ztemate/internal/store"
	"github.com/feiyushaw/ztemate/internal/zte"
)

type Settings struct {
	Address       string `json:"address"`
	Alias         string `json:"alias"`
	Notifications bool   `json:"notifications"`
	OpenAtLogin   bool   `json:"openAtLogin"`
	Theme         string `json:"theme"`
}
type Sample struct {
	At int64    `json:"at"`
	RX *float64 `json:"rx"`
	TX *float64 `json:"tx"`
}
type Snapshot struct {
	Traffic        TrafficView     `json:"traffic"`
	TrafficWarning string          `json:"trafficWarning"`
	Mode           string          `json:"mode"`
	State          string          `json:"state"`
	Error          string          `json:"error"`
	SMSWarning     string          `json:"smsWarning"`
	DeviceWarning  string          `json:"deviceWarning"`
	LastSync       string          `json:"lastSync"`
	Settings       Settings        `json:"settings"`
	Status         zte.Status      `json:"status"`
	Messages       []store.Message `json:"messages"`
	Devices        []zte.Device    `json:"devices"`
	History        []Sample        `json:"history"`
	Unread         int             `json:"unread"`
}
type Hooks struct {
	Copy      func(string)
	OpenMain  func()
	OpenAdmin func(string) error
	Notify    func(string)
	SetLogin  func(bool) error
}
type Service struct {
	traffic                          TrafficLedger
	mu                               sync.Mutex
	pollMu                           sync.Mutex
	snap                             Snapshot
	db                               *store.Store
	client                           *zte.Client
	hooks                            Hooks
	active                           bool
	sleeping                         bool
	initialized                      bool
	lastStatus, lastSMS, lastDevices time.Time
	quit                             chan struct{}
}

func New(db *store.Store, h Hooks) (*Service, error) {
	s := &Service{db: db, hooks: h, quit: make(chan struct{}), snap: Snapshot{Mode: "live", State: "unconfigured", Settings: Settings{Alias: "我的路由器", Theme: "system"}, Messages: []store.Message{}, Devices: []zte.Device{}, History: []Sample{}}}
	if e := db.LoadSettings(&s.snap.Settings); e != nil {
		return nil, e
	}
	if s.snap.Settings.Address != "" {
		s.snap.State = "disconnected"
		ms, e := db.List(s.snap.Settings.Address)
		if e != nil {
			return nil, e
		}
		s.snap.Messages = ms
	}
	s.traffic.Plan = defaultPlan()
	if e := db.LoadDocument("traffic", &s.traffic); e != nil {
		return nil, e
	}
	s.snap.Traffic = s.traffic.view()
	return s, nil
}
func (s *Service) Start() {
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.poll()
			case <-s.quit:
				return
			}
		}
	}()
}
func (s *Service) Stop()            { close(s.quit) }
func (s *Service) SetActive(v bool) { s.mu.Lock(); s.active = v; s.mu.Unlock() }
func (s *Service) SetSleeping(v bool) {
	s.mu.Lock()
	s.sleeping = v
	s.mu.Unlock()
	if !v {
		go s.Refresh()
	}
}
func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.snap
	v.Messages = append([]store.Message{}, v.Messages...)
	v.Devices = append([]zte.Device{}, v.Devices...)
	v.History = append([]Sample{}, v.History...)
	return v
}
func (s *Service) Connect(address, password string) error {
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	c, e := zte.New(address)
	if e != nil {
		return e
	}
	s.mu.Lock()
	s.client = nil
	s.snap.Mode = "live"
	s.snap.State = "connecting"
	s.snap.Error = ""
	s.snap.Status = zte.Status{}
	s.snap.LastSync = ""
	s.snap.History = []Sample{}
	s.snap.Devices = []zte.Device{}
	s.snap.Messages = []store.Message{}
	s.snap.Unread = 0
	s.snap.SMSWarning = ""
	s.snap.DeviceWarning = ""
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if password == "" {
		e = fmt.Errorf("请输入路由器管理密码")
	} else {
		e = c.Login(ctx, password)
	}
	if e != nil {
		s.fail(e)
		return e
	}
	status, e := c.Status(ctx)
	if e != nil {
		s.fail(e)
		return e
	}
	s.mu.Lock()
	cfg := s.snap.Settings
	cfg.Address = c.Address()
	s.mu.Unlock()
	if e = s.db.SaveSettings(cfg); e != nil {
		s.fail(fmt.Errorf("无法保存连接配置：%w", e))
		return e
	}
	s.mu.Lock()
	s.snap.Settings = cfg
	s.traffic = TrafficLedger{Plan: defaultPlan()}
	if err := s.db.LoadDocument("traffic", &s.traffic); err != nil {
		s.snap.TrafficWarning = err.Error()
	}
	s.client = c
	s.snap.Status = status
	s.snap.State = "online"
	s.snap.LastSync = time.Now().Format(time.RFC3339)
	s.updateTrafficLocked(time.Now())
	s.initialized = false
	s.lastStatus = time.Now()
	s.lastSMS = time.Time{}
	s.lastDevices = time.Time{}
	s.mu.Unlock()
	go s.Refresh()
	return nil
}
func (s *Service) fail(e error) {
	s.mu.Lock()
	s.snap.State = "error"
	s.snap.Error = e.Error()
	s.mu.Unlock()
}
func (s *Service) Disconnect() {
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = nil
	s.snap.State = "paused"
	s.snap.Error = ""
}
func (s *Service) SaveSettings(cfg Settings) error {

	if cfg.Theme != "system" && cfg.Theme != "light" && cfg.Theme != "dark" {
		return fmt.Errorf("外观设置无效")
	}
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	cfg.Address = s.snap.Settings.Address
	s.mu.Unlock()
	if s.hooks.SetLogin != nil {
		if e := s.hooks.SetLogin(cfg.OpenAtLogin); e != nil {
			return fmt.Errorf("开机启动设置失败：%w", e)
		}
	}
	if e := s.db.SaveSettings(cfg); e != nil {
		return e
	}
	s.mu.Lock()
	s.snap.Settings = cfg
	s.mu.Unlock()
	return nil
}
func (s *Service) OpenMain() {
	if s.hooks.OpenMain != nil {
		s.hooks.OpenMain()
	}
}
func (s *Service) OpenAdmin() error {
	s.mu.Lock()
	a := s.snap.Settings.Address
	s.mu.Unlock()
	if a == "" {
		return fmt.Errorf("请先连接路由器")
	}
	if s.hooks.OpenAdmin != nil {
		return s.hooks.OpenAdmin(a)
	}
	return nil
}
func (s *Service) CopyCode(id string) error {
	snap := s.Snapshot()
	for _, m := range snap.Messages {
		if m.ID == id {
			code := ExtractOTP(m.Content)
			if code == "" {
				return fmt.Errorf("此短信没有明确的验证码")
			}
			if s.hooks.Copy != nil {
				s.hooks.Copy(code)
			}
			return nil
		}
	}
	return fmt.Errorf("短信不存在")
}

var otp = regexp.MustCompile(`(?i)(?:验证码|校验码|动态码|确认码|verification code|code|otp)[^0-9\n]{0,16}([0-9]{4,8})(?:[^0-9]|$)`)

func ExtractOTP(text string) string {
	m := otp.FindStringSubmatch(text)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
func (s *Service) Refresh() {
	s.mu.Lock()
	s.lastStatus = time.Time{}
	s.lastSMS = time.Time{}
	s.lastDevices = time.Time{}
	s.mu.Unlock()
	s.poll()
}
func (s *Service) poll() {
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	c := s.client
	active := s.active
	skip := s.sleeping || s.snap.State == "paused"
	mode := s.snap.Mode
	lastStatus, lastSMS, lastDevices := s.lastStatus, s.lastSMS, s.lastDevices
	s.mu.Unlock()
	if skip || mode == "demo" || c == nil {
		return
	}
	interval := 30 * time.Second
	if active {
		interval = 2 * time.Second
	}
	now := time.Now()
	if now.Sub(lastStatus) >= interval {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		v, e := c.Status(ctx)
		cancel()
		s.mu.Lock()
		s.lastStatus = now
		if e != nil {
			s.snap.State = "error"
			s.snap.Error = e.Error()
			s.appendSample(Sample{At: now.UnixMilli()})
		} else {
			s.snap.State = "online"
			s.snap.Error = ""
			s.snap.Status = v
			s.updateTrafficLocked(now)
			s.snap.LastSync = now.Format(time.RFC3339)
			s.appendSample(Sample{At: now.UnixMilli(), RX: v.RX, TX: v.TX})
		}
		s.mu.Unlock()
		if e != nil {
			return
		}
	}
	if now.Sub(lastSMS) >= 10*time.Second {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		ms, e := c.Messages(ctx)
		cancel()
		var archived []store.Message
		newCount := 0
		if e == nil {
			old, err := s.db.List(c.Address())
			e = err
			if e == nil {
				known := map[string]bool{}
				for _, m := range old {
					known[m.RemoteID+"|"+m.Date+"|"+m.Sender+"|"+m.Content] = true
				}
				for _, m := range ms {
					if !known[m.RemoteID+"|"+m.Date+"|"+m.Sender+"|"+m.Content] {
						newCount++
					}
				}
				_, e = s.db.Save(c.Address(), ms)
				if e == nil {
					archived, e = s.db.List(c.Address())
				}
			}
		}
		s.mu.Lock()
		s.lastSMS = now
		s.snap.SMSWarning = ""
		notify := false
		if e != nil {
			s.snap.SMSWarning = e.Error()
		} else {
			s.snap.Messages = archived
			s.snap.Unread = 0
			for _, m := range ms {
				if m.Unread {
					s.snap.Unread++
				}
			}
			notify = s.initialized && newCount > 0 && s.snap.Settings.Notifications
			s.initialized = true
		}
		s.mu.Unlock()
		if notify && s.hooks.Notify != nil {
			s.hooks.Notify(fmt.Sprintf("收到 %d 条新短信，打开 ZTEMate 查看。", newCount))
		}
	}
	if now.Sub(lastDevices) >= 30*time.Second {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ds, e := c.Devices(ctx)
		cancel()
		s.mu.Lock()
		s.lastDevices = now
		s.snap.DeviceWarning = ""
		if e != nil {
			s.snap.DeviceWarning = e.Error()
		} else {
			s.snap.Devices = ds
		}
		s.mu.Unlock()
	}
}
func (s *Service) appendSample(v Sample) {
	s.snap.History = append(s.snap.History, v)
	cutoff := v.At - 60000
	for len(s.snap.History) > 0 && s.snap.History[0].At < cutoff {
		s.snap.History = s.snap.History[1:]
	}
}
func ptr(v float64) *float64 { return &v }
func (s *Service) Demo() {
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = nil
	s.snap.Mode = "demo"
	s.snap.State = "online"
	s.snap.Error = ""
	s.snap.SMSWarning = ""
	s.snap.DeviceWarning = ""
	now := time.Now()
	s.snap.LastSync = now.Format(time.RFC3339)
	s.snap.Status = zte.Status{Model: "MC888S", Firmware: "演示固件 · 非实机数据", Carrier: "中国联通", Network: "5G SA", RSRP: ptr(-87), RSRQ: ptr(-11), SINR: ptr(23), Band: "n78", RX: ptr(12.4e6), TX: ptr(1.2e6), Monthly: ptr(142.6e9), Uptime: "90620"}
	s.snap.History = []Sample{}
	for i := 30; i >= 0; i-- {
		v := 8e6 + 4e6*math.Sin(float64(i)*.45) + 2e6*math.Cos(float64(i)*1.2)
		s.snap.History = append(s.snap.History, Sample{At: now.Add(-time.Duration(i) * 2 * time.Second).UnixMilli(), RX: ptr(v), TX: ptr(v * .12)})
	}
	s.snap.Messages = []store.Message{{ID: "demo-1", Message: zte.Message{RemoteID: "1", Sender: "服务通知", Content: "您的验证码为 123456，5 分钟内有效。请勿向他人泄露。", Date: now.Format("2006-01-02 15:04:05"), Unread: true}}, {ID: "demo-2", Message: zte.Message{RemoteID: "2", Sender: "10010", Content: "【演示短信】您本月的套餐使用情况请以运营商账单为准。", Date: now.Add(-time.Hour).Format("2006-01-02 15:04:05")}}, {ID: "demo-3", Message: zte.Message{RemoteID: "3", Sender: "服务通知", Content: "欢迎使用 ZTEMate。网络与短信，触手可及。", Date: now.Add(-2 * time.Hour).Format("2006-01-02 15:04:05")}}}
	s.snap.Devices = []zte.Device{{Name: "我的 Mac", IP: "192.0.2.10", MAC: "02:00:00:00:00:01", Kind: "Wi-Fi"}, {Name: "Ubuntu", IP: "192.0.2.20", MAC: "02:00:00:00:00:02", Kind: "有线"}}
	l := TrafficLedger{Plan: defaultPlan(), UsedData: 7.15 * multiplier("GB"), Cycle: cycleAt(now, 1)}
	l.Plan.Enabled = true
	l.Plan.AutoReset = true
	s.snap.Traffic = l.view()
	s.snap.Unread = 1
}
func (s *Service) LeaveDemo() {
	s.pollMu.Lock()
	defer s.pollMu.Unlock()
	s.mu.Lock()
	cfg := s.snap.Settings
	s.snap = Snapshot{Mode: "live", State: "disconnected", Settings: cfg, Messages: []store.Message{}, Devices: []zte.Device{}, History: []Sample{}}
	s.traffic = TrafficLedger{Plan: defaultPlan()}
	if e := s.db.LoadDocument("traffic", &s.traffic); e != nil {
		s.snap.TrafficWarning = e.Error()
	}
	s.snap.Traffic = s.traffic.view()
	s.mu.Unlock()
	if strings.TrimSpace(cfg.Address) != "" {
		ms, e := s.db.List(cfg.Address)
		s.mu.Lock()
		if e == nil {
			s.snap.Messages = ms
		} else {
			s.snap.SMSWarning = e.Error()
		}
		s.mu.Unlock()
	}
}
