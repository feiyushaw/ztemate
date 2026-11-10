package zte

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeContent(t *testing.T) {
	for _, tc := range []struct {
		in, want string
		bad      bool
	}{{"4F60597D", "你好", false}, {"D83DDE00", "😀", false}, {"0041", "A", false}, {"00", "", true}, {"zzzz", "", true}, {"D800", "", true}, {"DC00", "", true}} {
		got, e := DecodeContent(tc.in)
		if (e != nil) != tc.bad || got != tc.want {
			t.Errorf("decode %q = %q, %v", tc.in, got, e)
		}
	}
}
func TestLoginAndReadOnlyRequests(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.Referer(), "/index.html") {
			t.Error("missing Referer")
		}
		if r.Method == "POST" {
			posts++
			r.ParseForm()
			if r.Form.Get("password") != HashPassword("test-password", "challenge") {
				t.Error("incorrect hash")
			}
			http.SetCookie(w, &http.Cookie{Name: "stok", Value: "test-session", Path: "/"})
			w.Write([]byte(`{"result":"0"}`))
			return
		}
		switch r.URL.Query().Get("cmd") {
		case "LD":
			w.Write([]byte(`{"LD":"challenge"}`))
		case "sms_data_total":
			// This firmware treats multi_data as an NV lookup, not an SMS command.
			if r.URL.Query().Get("multi_data") == "1" {
				w.Write([]byte(`{"sms_data_total":""}`))
				return
			}

			if _, e := r.Cookie("stok"); e != nil {
				t.Error("session not preserved")
			}
			w.Write([]byte(`{"messages":[{"id":"1","number":"10010","content":"4F60597D","date":"26,11,09,12,00,00","tag":"1"}]}`))
		default:
			w.Write([]byte(`{"network_type":"5G","realtime_rx_thrpt":"1024","Z5g_rsrp":""}`))
		}
	}))
	defer srv.Close()
	c, e := New(srv.URL)
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Login(context.Background(), "test-password"); e != nil {
		t.Fatal(e)
	}
	s, e := c.Status(context.Background())
	if e != nil || s.RSRP != nil || s.RX == nil || *s.RX != 1024 {
		t.Fatalf("status %#v %v", s, e)
	}
	ms, e := c.Messages(context.Background())
	if e != nil || len(ms) != 1 || ms[0].Content != "你好" || !ms[0].Unread {
		t.Fatalf("messages %#v %v", ms, e)
	}
	if posts != 1 {
		t.Fatalf("unexpected writes %d", posts)
	}
}
func TestRepeatedSMSPageIsError(t *testing.T) {
	rows := []map[string]string{}
	for i := 0; i < 50; i++ {
		rows = append(rows, map[string]string{"id": "same", "content": "0041"})
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"messages": rows})
	}))
	defer srv.Close()
	c, _ := New(srv.URL)
	if _, e := c.Messages(context.Background()); e == nil {
		t.Fatal("repeating page must not count as complete")
	}
}
func TestInvalidAddress(t *testing.T) {
	for _, a := range []string{"file:///etc/passwd", "http://user:pass@router", "http://router/path", "http://router?cmd=x", "router"} {
		if _, e := New(a); e == nil {
			t.Errorf("accepted %s", a)
		}
	}
}
func TestRejectRedirectAndHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "http://example.invalid", 302) }))
	defer srv.Close()
	c, _ := New(srv.URL)
	if _, e := c.Status(context.Background()); e == nil {
		t.Fatal("accepted redirect")
	}
}

func TestCommandQueryModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cmd := r.URL.Query().Get("cmd")
		if strings.Contains(cmd, ",") {
			if r.URL.Query().Get("multi_data") != "1" {
				t.Error("combined NV query must use multi_data")
			}
		} else {
			if _, ok := r.URL.Query()["multi_data"]; ok {
				t.Error("single command must omit multi_data")
			}
		}
		w.Write([]byte(`{"result":"0"}`))
	}))
	defer srv.Close()
	c, _ := New(srv.URL)
	for _, cmd := range []string{"LD", "loginfo", "station_list", "lan_station_list", "sms_data_total", "network_type,signalbar"} {
		if _, err := c.Get(context.Background(), cmd, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMissingMessagesDistinguishesExpiredSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cmd") == "loginfo" {
			w.Write([]byte(`{"loginfo":"no"}`))
		} else {
			w.Write([]byte(`{"sms_data_total":""}`))
		}
	}))
	defer srv.Close()
	c, _ := New(srv.URL)
	_, err := c.Messages(context.Background())
	if err == nil || !strings.Contains(err.Error(), "会话已失效") {
		t.Fatalf("expected actionable session error, got %v", err)
	}
}

func TestEmptySMSInboxIsSuccessful(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"messages":[]}`)) }))
	defer srv.Close()
	c, _ := New(srv.URL)
	ms, err := c.Messages(context.Background())
	if err != nil || len(ms) != 0 {
		t.Fatalf("empty inbox: %v %v", ms, err)
	}
}
