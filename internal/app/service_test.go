package app

import (
	"github.com/feiyushaw/ztemate/internal/store"
	"path/filepath"
	"testing"
)

func TestOTP(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{"您的验证码为 123456，五分钟有效", "123456"}, {"账单编号 123456", ""}, {"code: 123456789", ""}, {"OTP: 4567", "4567"}} {
		if got := ExtractOTP(tc.in); got != tc.want {
			t.Errorf("%q: %s", tc.in, got)
		}
	}
}
func TestDemoDoesNotArchiveAndSettingsPersist(t *testing.T) {
	db, e := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	s, e := New(db, Hooks{})
	if e != nil {
		t.Fatal(e)
	}
	s.Demo()
	if s.Snapshot().Mode != "demo" {
		t.Fatal("not demo")
	}
	s.LeaveDemo()
	if len(s.Snapshot().Messages) != 0 {
		t.Fatal("demo leaked")
	}
	cfg := s.Snapshot().Settings
	cfg.Alias = "test router"
	if e = s.SaveSettings(cfg); e != nil {
		t.Fatal(e)
	}
	other, e := New(db, Hooks{})
	if e != nil {
		t.Fatal(e)
	}
	if other.Snapshot().Settings.Alias != "test router" {
		t.Fatal("settings not persistent")
	}
}
